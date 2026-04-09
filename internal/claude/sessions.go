package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type Message struct {
	Model   string          `json:"model"`
	Usage   Usage           `json:"usage"`
	Content json.RawMessage `json:"content"`
}

type Entry struct {
	Type      string  `json:"type"`
	Message   Message `json:"message"`
	Timestamp string  `json:"timestamp"`
	SessionID string  `json:"sessionId"`
	CWD       string  `json:"cwd"`
	Version   string  `json:"version"`
	GitBranch string  `json:"gitBranch"`
}

type Session struct {
	ID             string
	Project        string
	Model          string
	StartTime      time.Time
	EndTime        time.Time
	Usage          Usage
	Turns          int
	Cost           float64
	Errors         int
	Interruptions  int
	ToolCounts     map[string]int
	FirstInputSize int
	LastInputSize  int
	Summary        string
}

func ClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func ProjectDirs() ([]string, error) {
	projectsDir := filepath.Join(ClaudeDir(), "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(projectsDir, e.Name()))
		}
	}
	return dirs, nil
}

func projectDirToName(dir string) string {
	base := filepath.Base(dir)
	// Convert "-Users-zhubert-Code-myproject" to "~/Code/myproject"
	parts := strings.Split(base, "-")
	if len(parts) >= 3 && parts[0] == "" && parts[1] == "Users" {
		// Skip empty, "Users", username
		remaining := parts[3:]
		if len(remaining) == 0 {
			return "~"
		}
		return "~/" + strings.Join(remaining, "/")
	}
	return base
}

func ParseSession(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	projectName := projectDirToName(filepath.Dir(path))

	sess := &Session{
		ID:         sessionID,
		Project:    projectName,
		ToolCounts: make(map[string]int),
	}

	var firstTime, lastTime time.Time

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Timestamp != "" {
			t, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
			if err == nil {
				if firstTime.IsZero() || t.Before(firstTime) {
					firstTime = t
				}
				if t.After(lastTime) {
					lastTime = t
				}
			}
		}

		switch entry.Type {
		case "assistant":
			usage := entry.Message.Usage
			if usage.InputTokens == 0 && usage.OutputTokens == 0 {
				continue
			}

			sess.Turns++
			sess.Usage.InputTokens += usage.InputTokens
			sess.Usage.OutputTokens += usage.OutputTokens
			sess.Usage.CacheCreationInputTokens += usage.CacheCreationInputTokens
			sess.Usage.CacheReadInputTokens += usage.CacheReadInputTokens

			if sess.Model == "" && entry.Message.Model != "" {
				sess.Model = entry.Message.Model
			}

			// Track context growth
			inputSize := usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
			if sess.FirstInputSize == 0 {
				sess.FirstInputSize = inputSize
			}
			sess.LastInputSize = inputSize

			// Count tool uses
			parseToolUses(entry.Message.Content, sess)

		case "user":
			parseUserEntry(line, sess)
		}
	}

	sess.StartTime = firstTime
	sess.EndTime = lastTime
	sess.Cost = ComputeCost(sess.Model, sess.Usage)

	return sess, nil
}

func parseToolUses(raw json.RawMessage, sess *Session) {
	var content []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &content) != nil {
		return
	}
	for _, c := range content {
		if c.Type == "tool_use" && c.Name != "" {
			sess.ToolCounts[c.Name]++
		}
	}
}

func parseUserEntry(line []byte, sess *Session) {
	var raw struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &raw) != nil {
		return
	}

	// Content can be a string or an array
	var contentStr string
	if json.Unmarshal(raw.Message.Content, &contentStr) == nil {
		if sess.Summary == "" && len(contentStr) > 5 && !strings.HasPrefix(contentStr, "[Request") {
			sess.Summary = truncate(contentStr, 120)
		}
		return
	}

	var content []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		IsError bool   `json:"is_error"`
		Content string `json:"content"`
	}
	if json.Unmarshal(raw.Message.Content, &content) != nil {
		return
	}

	for _, c := range content {
		switch c.Type {
		case "text":
			if strings.Contains(c.Text, "[Request interrupted by user") {
				sess.Interruptions++
			}
			if sess.Summary == "" && len(c.Text) > 5 && !strings.HasPrefix(c.Text, "[Request") {
				sess.Summary = truncate(c.Text, 120)
			}
		case "tool_result":
			if c.IsError {
				sess.Errors++
			}
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func AllSessions() ([]*Session, error) {
	dirs, err := ProjectDirs()
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			continue
		}
		for _, f := range files {
			sess, err := ParseSession(f)
			if err != nil || sess.Turns == 0 {
				continue
			}
			sessions = append(sessions, sess)
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.Before(sessions[j].StartTime)
	})

	return sessions, nil
}

func SessionsSince(since time.Time) ([]*Session, error) {
	all, err := AllSessions()
	if err != nil {
		return nil, err
	}
	var filtered []*Session
	for _, s := range all {
		if s.StartTime.After(since) || s.StartTime.Equal(since) {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}
