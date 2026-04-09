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
	Model string `json:"model"`
	Usage Usage  `json:"usage"`
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
	ID        string
	Project   string
	Model     string
	StartTime time.Time
	EndTime   time.Time
	Usage     Usage
	Turns     int
	Cost      float64
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
		ID:      sessionID,
		Project: projectName,
	}

	var firstTime, lastTime time.Time

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
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

		if entry.Type != "assistant" {
			continue
		}

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
	}

	sess.StartTime = firstTime
	sess.EndTime = lastTime
	sess.Cost = ComputeCost(sess.Model, sess.Usage)

	return sess, nil
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
