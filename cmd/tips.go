package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zhubert/tokencoach/internal/claude"
	"github.com/zhubert/tokencoach/internal/display"
)

func startSpinner(msg string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Printf("\r\033[K")
				return
			default:
				fmt.Printf("\r  %s %s...", frames[i%len(frames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return func() { close(done) }
}

func renderBox(title string, lines []string) string {
	w := display.GetTermWidth()
	boxWidth := w - 4
	if boxWidth > 76 {
		boxWidth = 76
	}

	titleLine := lipgloss.NewStyle().Width(boxWidth).Bold(true).Foreground(display.ColorAccent).Align(lipgloss.Center).Render(title)
	content := strings.Join(lines, "\n")

	box := display.RoundedBox(content, boxWidth)
	return titleLine + "\n" + box
}

type sessionTip struct {
	SessionIndex int    `json:"session_index"`
	Header       string `json:"header"`
	Description  string `json:"description"`
	Metrics      string `json:"metrics"`
	Tip          string `json:"tip"`
}

type promptMatch struct {
	Quote   string `json:"quote"`
	Better  string `json:"better"`
}

func renderSessionBlock(s sessionTip, pm *promptMatch) string {
	w := display.GetTermWidth()
	blockWidth := w - 4
	if blockWidth > 76 {
		blockWidth = 76
	}

	header := lipgloss.NewStyle().Width(blockWidth).Bold(true).Foreground(display.ColorAccent).Align(lipgloss.Center).Render(s.Header)

	metricsLine := lipgloss.NewStyle().Foreground(display.ColorDim).Padding(1, 0).Render(s.Metrics)
	tipLine := lipgloss.NewStyle().Bold(true).Foreground(display.ColorTip).Render("Tip: ") +
		lipgloss.NewStyle().Foreground(display.ColorTip).Render(s.Tip)

	content := s.Description + "\n" + metricsLine + "\n" + tipLine

	if pm != nil && pm.Quote != "" {
		quoteLine := lipgloss.NewStyle().Foreground(display.ColorWarn).Padding(1, 0, 0, 0).Render("You wrote: \"" + pm.Quote + "\"")
		betterLine := lipgloss.NewStyle().Foreground(display.ColorTip).Render("Better: \"" + pm.Better + "\"")
		content += "\n" + quoteLine + "\n" + betterLine
	}

	box := display.RoundedBox(content, blockWidth)

	return header + "\n" + box
}

var (
	tipsDays  int
	tipsModel string
	tipsTop   int
)

var tipsCmd = &cobra.Command{
	Use:   "tips",
	Short: "AI-generated tips to improve your sessions",
	RunE:  runTips,
}

type sessionSummary struct {
	Index            int            `json:"index"`
	Time             string         `json:"time"`
	Project          string         `json:"project"`
	Summary          string         `json:"summary"`
	Model            string         `json:"model"`
	Cost             float64        `json:"cost"`
	Turns            int            `json:"turns"`
	Errors           int            `json:"errors"`
	Interruptions    int            `json:"interruptions"`
	ToolCounts       map[string]int `json:"tool_counts"`
	OutputTokens     int            `json:"output_tokens"`
	ContextGrowthPct int            `json:"context_growth_pct"`
}

type historicalStats struct {
	TotalSessions  int     `json:"total_sessions"`
	TotalCost      float64 `json:"total_cost"`
	AvgSessionCost float64 `json:"avg_session_cost"`
	MinSessionCost float64 `json:"min_session_cost"`
	MaxSessionCost float64 `json:"max_session_cost"`
	AvgTurns       float64 `json:"avg_turns"`
	AvgErrors      float64 `json:"avg_errors"`
	DaysOfData     int     `json:"days_of_data"`
}

type tipsSummary struct {
	PeriodDays     int              `json:"period_days"`
	PeriodTotal    float64          `json:"period_total"`
	PeriodSessions int              `json:"period_sessions"`
	AvgSessionCost float64          `json:"avg_session_cost"`
	Historical     historicalStats  `json:"historical_baseline"`
	Sessions       []sessionSummary `json:"top_sessions"`
}

func init() {
	tipsCmd.Flags().IntVar(&tipsDays, "days", 7, "Number of days to analyze")
	tipsCmd.Flags().IntVar(&tipsTop, "top", 10, "Number of top sessions to analyze")
	tipsCmd.Flags().StringVar(&tipsModel, "model", "sonnet", "Model to use for analysis (haiku, sonnet, opus)")
	rootCmd.AddCommand(tipsCmd)
}

func runTips(cmd *cobra.Command, args []string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.AddDate(0, 0, -(tipsDays - 1))

	// Pull all sessions for historical baseline
	allSessions, err := claude.AllSessions()
	if err != nil {
		fmt.Println("No Claude Code session data found. Use Claude Code first, then try again.")
		return nil
	}
	debug("tips: loaded %d total sessions", len(allSessions))

	if len(allSessions) == 0 {
		fmt.Println("No Claude Code sessions found. Use Claude Code first, then try again.")
		return nil
	}

	// Filter to the requested window
	var sessions []*claude.Session
	for _, s := range allSessions {
		if s.StartTime.After(since) || s.StartTime.Equal(since) {
			sessions = append(sessions, s)
		}
	}
	debug("tips: %d sessions in %d-day window", len(sessions), tipsDays)

	if len(sessions) == 0 {
		fmt.Printf("No sessions in the last %d days.\n", tipsDays)
		return nil
	}

	// Compute historical baseline from all data
	var hist historicalStats
	hist.TotalSessions = len(allSessions)
	hist.MinSessionCost = allSessions[0].Cost
	var totalTurns, totalErrors int
	for _, s := range allSessions {
		hist.TotalCost += s.Cost
		totalTurns += s.Turns
		totalErrors += s.Errors
		if s.Cost < hist.MinSessionCost {
			hist.MinSessionCost = s.Cost
		}
		if s.Cost > hist.MaxSessionCost {
			hist.MaxSessionCost = s.Cost
		}
	}
	hist.AvgSessionCost = hist.TotalCost / float64(hist.TotalSessions)
	hist.AvgTurns = float64(totalTurns) / float64(hist.TotalSessions)
	hist.AvgErrors = float64(totalErrors) / float64(hist.TotalSessions)
	if len(allSessions) > 0 {
		first := allSessions[0].StartTime
		hist.DaysOfData = int(now.Sub(first).Hours()/24) + 1
	}

	// Build period summary
	var periodCost float64
	for _, s := range sessions {
		periodCost += s.Cost
	}

	summary := tipsSummary{
		PeriodDays:     tipsDays,
		PeriodTotal:    periodCost,
		PeriodSessions: len(sessions),
		AvgSessionCost: periodCost / float64(len(sessions)),
		Historical:     hist,
	}

	// Sort by cost descending, take top 10
	sorted := make([]*claude.Session, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cost > sorted[j].Cost
	})
	if len(sorted) > tipsTop {
		sorted = sorted[:tipsTop]
	}

	for i, s := range sorted {
		growthPct := 0
		if s.FirstInputSize > 0 {
			growthPct = ((s.LastInputSize - s.FirstInputSize) * 100) / s.FirstInputSize
		}
		debug("tips: session[%d] %s $%.2f %d turns %d prompts", i, s.Project, s.Cost, s.Turns, len(s.UserPrompts))

		summary.Sessions = append(summary.Sessions, sessionSummary{
			Index:            i,
			Time:             s.StartTime.Local().Format("Mon 3:04pm"),
			Project:          s.Project,
			Summary:          s.Summary,
			Model:            s.Model,
			Cost:             s.Cost,
			Turns:            s.Turns,
			Errors:           s.Errors,
			Interruptions:    s.Interruptions,
			ToolCounts:       s.ToolCounts,
			OutputTokens:     s.Usage.OutputTokens,
			ContextGrowthPct: growthPct,
		})
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling summary: %w", err)
	}
	debug("tips: pass1 payload size=%d bytes", len(data))

	prompt := fmt.Sprintf(`You are a cost advisor for AI coding sessions. Given session analytics from the past %d days plus a historical baseline, identify 2-3 sessions with actionable tips to reduce costs.

Use the historical baseline to judge what's normal vs abnormal for this user.

Patterns to look for:
- Sessions with high error counts (agent stuck in retry loops)
- Sessions with many interruptions (user had to repeatedly redirect)
- Sessions with heavy Read/Grep/Glob tool use and high context growth (unfocused exploration)
- Sessions with many turns but low output tokens (spinning wheels)
- Sessions that cost much more than this user's historical average

Return a JSON array of 2-3 objects. Each object must have these exact fields:
- "session_index": the index field from the session you're referencing
- "header": cost, day/time, and project on one line (e.g. "$16.53  Thu 9:14am  ~/Code/insights")
- "description": what they were doing, from the summary field
- "metrics": key metrics showing the problem, with specific numbers compared to baseline
- "tip": one concrete, actionable sentence

Return ONLY valid JSON. No markdown fences, no preamble, no other text.

<data>
%s
</data>`, tipsDays, string(data))

	// Compute summary box
	boxTitle := fmt.Sprintf("%d-Day Summary", tipsDays)
	costHL := lipgloss.NewStyle().Foreground(display.ColorAccent)
	boxLines := []string{
		fmt.Sprintf("Spend: %s (%d sessions)", costHL.Render(fmt.Sprintf("$%.2f", periodCost)), len(sessions)),
		fmt.Sprintf("Avg:   %s/session (baseline: %s)", costHL.Render(fmt.Sprintf("$%.2f", summary.AvgSessionCost)), costHL.Render(fmt.Sprintf("$%.2f", hist.AvgSessionCost))),
	}
	if len(sorted) >= 2 && periodCost > 0 {
		top2Cost := sorted[0].Cost + sorted[1].Cost
		pct := (top2Cost / periodCost) * 100
		if pct >= 15 {
			boxLines = append(boxLines, fmt.Sprintf("Top 2 sessions = %s (%.0f%% of total)", costHL.Render(fmt.Sprintf("$%.2f", top2Cost)), pct))
		}
	}

	// Show spinner while waiting
	fmt.Println()
	start := time.Now()
	stop := startSpinner("Analyzing sessions")
	c := exec.Command("claude", "-p", prompt, "--model", tipsModel)
	out, err := c.Output()
	elapsed := time.Since(start).Round(time.Millisecond)
	stop()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Println("  Claude CLI not found. Install it from https://docs.anthropic.com/en/docs/claude-code")
		} else {
			fmt.Printf("  Failed to get tips from Claude (%s model). Check your API key and try again.\n", tipsModel)
		}
		return nil
	}
	debug("tips: pass1 completed in %s, response size=%d bytes", elapsed, len(out))
	fmt.Printf("%s\n", elapsed)

	// Strip markdown fences if present
	raw := strings.TrimSpace(string(out))
	if strings.HasPrefix(raw, "```") {
		if i := strings.Index(raw, "\n"); i != -1 {
			raw = raw[i+1:]
		}
		if strings.HasSuffix(raw, "```") {
			raw = raw[:len(raw)-3]
		}
		raw = strings.TrimSpace(raw)
	}

	var tips []sessionTip
	if err := json.Unmarshal([]byte(raw), &tips); err != nil {
		debug("tips: pass1 JSON parse error: %v", err)
		debug("tips: pass1 raw output: %s", raw)
		fmt.Print(string(out))
		return nil
	}
	for i, t := range tips {
		debug("tips: pass1 tip[%d] session_index=%d header=%q", i, t.SessionIndex, t.Header)
	}

	// Pass 2: for each tip, find the exemplifying user prompt
	start = time.Now()
	stop = startSpinner("Finding example prompts")
	type indexedMatch struct {
		idx int
		pm  *promptMatch
	}
	matchCh := make(chan indexedMatch, len(tips))

	for i, tip := range tips {
		go func(idx int, t sessionTip) {
			pm := findExamplePrompt(t, sorted, tipsModel)
			matchCh <- indexedMatch{idx, pm}
		}(i, tip)
	}

	matches := make([]*promptMatch, len(tips))
	for range tips {
		im := <-matchCh
		matches[im.idx] = im.pm
	}
	elapsed = time.Since(start).Round(time.Millisecond)
	stop()
	fmt.Printf("%s\n\n", elapsed)

	fmt.Println(renderBox(boxTitle, boxLines))
	fmt.Println()

	for i, tip := range tips {
		fmt.Println(renderSessionBlock(tip, matches[i]))
		fmt.Println()
	}
	return nil
}

func truncateDebug(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func findExamplePrompt(tip sessionTip, sorted []*claude.Session, model string) *promptMatch {
	if tip.SessionIndex < 0 || tip.SessionIndex >= len(sorted) {
		debug("pass2: session_index %d out of range (have %d sessions)", tip.SessionIndex, len(sorted))
		return nil
	}
	sess := sorted[tip.SessionIndex]
	if len(sess.UserPrompts) == 0 {
		debug("pass2: session[%d] has no user prompts", tip.SessionIndex)
		return nil
	}
	debug("pass2: session[%d] sending %d prompts to model", tip.SessionIndex, len(sess.UserPrompts))

	promptsJSON, err := json.Marshal(sess.UserPrompts)
	if err != nil {
		return nil
	}
	debug("pass2: session[%d] prompt payload size=%d bytes", tip.SessionIndex, len(promptsJSON))

	p := fmt.Sprintf(`Given this tip about an AI coding session and the user's actual prompts from that session, pick the single prompt that best exemplifies the problem described in the tip. Then write a better version of that prompt that would have avoided the waste.

<tip>%s</tip>

<metrics>%s</metrics>

<user_prompts>
%s
</user_prompts>

Return a JSON object with exactly two fields:
- "quote": the exact text of the problematic prompt (copy it verbatim, but truncate to 200 chars if longer)
- "better": a rewritten version of that prompt that would have been more efficient

Return ONLY valid JSON. No markdown fences, no preamble.`, tip.Tip, tip.Metrics, string(promptsJSON))

	c := exec.Command("claude", "-p", p, "--model", model)
	var stderr strings.Builder
	c.Stderr = &stderr
	out, err := c.Output()
	if err != nil {
		debug("pass2: session[%d] claude error: %v stderr: %s", tip.SessionIndex, err, stderr.String())
		return nil
	}
	debug("pass2: session[%d] response size=%d bytes", tip.SessionIndex, len(out))

	raw := strings.TrimSpace(string(out))
	if strings.HasPrefix(raw, "```") {
		if i := strings.Index(raw, "\n"); i != -1 {
			raw = raw[i+1:]
		}
		if strings.HasSuffix(raw, "```") {
			raw = raw[:len(raw)-3]
		}
		raw = strings.TrimSpace(raw)
	}

	var pm promptMatch
	if err := json.Unmarshal([]byte(raw), &pm); err != nil {
		debug("pass2: session[%d] JSON parse error: %v raw: %s", tip.SessionIndex, err, raw)
		return nil
	}
	debug("pass2: session[%d] matched quote=%q", tip.SessionIndex, truncateDebug(pm.Quote, 80))
	return &pm
}
