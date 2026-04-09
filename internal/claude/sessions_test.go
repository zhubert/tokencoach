package claude

import (
	"path/filepath"
	"testing"
	"time"
)

func TestProjectDirToName(t *testing.T) {
	tests := []struct {
		dir  string
		want string
	}{
		{"/home/user/.claude/projects/-Users-zhubert-Code-myproject", "~/Code/myproject"},
		{"/home/user/.claude/projects/-Users-zhubert-Code-deep-nested-thing", "~/Code/deep/nested/thing"},
		{"/home/user/.claude/projects/-Users-zhubert", "~"},
		{"/home/user/.claude/projects/something-else", "something-else"},
	}
	for _, tt := range tests {
		got := projectDirToName(tt.dir)
		if got != tt.want {
			t.Errorf("projectDirToName(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is to..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestParseSession_Basic(t *testing.T) {
	path := filepath.Join("testdata", "basic.jsonl")
	sess, err := ParseSession(path)
	if err != nil {
		t.Fatalf("ParseSession(%q) error: %v", path, err)
	}

	if sess.ID != "basic" {
		t.Errorf("ID = %q, want %q", sess.ID, "basic")
	}

	if sess.Turns != 2 {
		t.Errorf("Turns = %d, want 2", sess.Turns)
	}

	if sess.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", sess.Model, "claude-sonnet-4-20250514")
	}

	// Check token accumulation
	if sess.Usage.InputTokens != 3000 {
		t.Errorf("InputTokens = %d, want 3000", sess.Usage.InputTokens)
	}
	if sess.Usage.OutputTokens != 1300 {
		t.Errorf("OutputTokens = %d, want 1300", sess.Usage.OutputTokens)
	}
	if sess.Usage.CacheCreationInputTokens != 6000 {
		t.Errorf("CacheCreationInputTokens = %d, want 6000", sess.Usage.CacheCreationInputTokens)
	}
	if sess.Usage.CacheReadInputTokens != 5000 {
		t.Errorf("CacheReadInputTokens = %d, want 5000", sess.Usage.CacheReadInputTokens)
	}

	// Check tool counts
	if sess.ToolCounts["Edit"] != 1 {
		t.Errorf("ToolCounts[Edit] = %d, want 1", sess.ToolCounts["Edit"])
	}

	// Check context growth
	if sess.FirstInputSize != 6000 { // 1000 + 0 + 5000
		t.Errorf("FirstInputSize = %d, want 6000", sess.FirstInputSize)
	}
	if sess.LastInputSize != 8000 { // 2000 + 5000 + 1000
		t.Errorf("LastInputSize = %d, want 8000", sess.LastInputSize)
	}

	// Check timestamps
	wantStart := time.Date(2025, 4, 7, 10, 0, 0, 0, time.UTC)
	if !sess.StartTime.Equal(wantStart) {
		t.Errorf("StartTime = %v, want %v", sess.StartTime, wantStart)
	}

	wantEnd := time.Date(2025, 4, 7, 10, 0, 15, 0, time.UTC)
	if !sess.EndTime.Equal(wantEnd) {
		t.Errorf("EndTime = %v, want %v", sess.EndTime, wantEnd)
	}

	// Check summary (from first user message)
	if sess.Summary != "Help me refactor the auth module" {
		t.Errorf("Summary = %q, want %q", sess.Summary, "Help me refactor the auth module")
	}

	// Check cost is computed
	if sess.Cost <= 0 {
		t.Errorf("Cost = %v, want > 0", sess.Cost)
	}

	// No errors or interruptions
	if sess.Errors != 0 {
		t.Errorf("Errors = %d, want 0", sess.Errors)
	}
	if sess.Interruptions != 0 {
		t.Errorf("Interruptions = %d, want 0", sess.Interruptions)
	}
}

func TestParseSession_ErrorsAndInterruptions(t *testing.T) {
	path := filepath.Join("testdata", "errors_and_interruptions.jsonl")
	sess, err := ParseSession(path)
	if err != nil {
		t.Fatalf("ParseSession(%q) error: %v", path, err)
	}

	if sess.Turns != 4 {
		t.Errorf("Turns = %d, want 4", sess.Turns)
	}

	if sess.Errors != 2 {
		t.Errorf("Errors = %d, want 2", sess.Errors)
	}

	if sess.Interruptions != 2 {
		t.Errorf("Interruptions = %d, want 2", sess.Interruptions)
	}

	if sess.Model != "claude-opus-4-20250514" {
		t.Errorf("Model = %q, want %q", sess.Model, "claude-opus-4-20250514")
	}

	// Tool counts
	if sess.ToolCounts["Bash"] != 2 {
		t.Errorf("ToolCounts[Bash] = %d, want 2", sess.ToolCounts["Bash"])
	}
	if sess.ToolCounts["Read"] != 1 {
		t.Errorf("ToolCounts[Read] = %d, want 1", sess.ToolCounts["Read"])
	}

	// Summary should be first user message
	if sess.Summary != "Fix the login bug" {
		t.Errorf("Summary = %q, want %q", sess.Summary, "Fix the login bug")
	}
}

func TestParseSession_Empty(t *testing.T) {
	path := filepath.Join("testdata", "empty.jsonl")
	sess, err := ParseSession(path)
	if err != nil {
		t.Fatalf("ParseSession(%q) error: %v", path, err)
	}

	if sess.Turns != 0 {
		t.Errorf("Turns = %d, want 0", sess.Turns)
	}

	if sess.Cost != 0 {
		t.Errorf("Cost = %v, want 0", sess.Cost)
	}
}

func TestParseSession_StringContent(t *testing.T) {
	path := filepath.Join("testdata", "string_content.jsonl")
	sess, err := ParseSession(path)
	if err != nil {
		t.Fatalf("ParseSession(%q) error: %v", path, err)
	}

	if sess.Summary != "This is a string content message" {
		t.Errorf("Summary = %q, want %q", sess.Summary, "This is a string content message")
	}

	if sess.Turns != 1 {
		t.Errorf("Turns = %d, want 1", sess.Turns)
	}
}

func TestParseSession_ToolsHeavy(t *testing.T) {
	path := filepath.Join("testdata", "tools_heavy.jsonl")
	sess, err := ParseSession(path)
	if err != nil {
		t.Fatalf("ParseSession(%q) error: %v", path, err)
	}

	if sess.ToolCounts["Read"] != 3 {
		t.Errorf("ToolCounts[Read] = %d, want 3", sess.ToolCounts["Read"])
	}
	if sess.ToolCounts["Grep"] != 1 {
		t.Errorf("ToolCounts[Grep] = %d, want 1", sess.ToolCounts["Grep"])
	}
	if sess.ToolCounts["Glob"] != 1 {
		t.Errorf("ToolCounts[Glob] = %d, want 1", sess.ToolCounts["Glob"])
	}

	// Context growth: first turn 1000+0+5000=6000, last turn 3000+5000+2000=10000
	if sess.FirstInputSize != 6000 {
		t.Errorf("FirstInputSize = %d, want 6000", sess.FirstInputSize)
	}
	if sess.LastInputSize != 10000 {
		t.Errorf("LastInputSize = %d, want 10000", sess.LastInputSize)
	}
}

func TestParseSession_NonexistentFile(t *testing.T) {
	_, err := ParseSession("testdata/nonexistent.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
