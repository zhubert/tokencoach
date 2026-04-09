package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhubert/insights/internal/claude"
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

var (
	tipsDays  int
	tipsModel string
	tipsTop   int
)

var tipsCmd = &cobra.Command{
	Use:   "tips",
	Short: "AI-generated tips to reduce your Claude Code costs",
	RunE:  runTips,
}

type sessionSummary struct {
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
		return fmt.Errorf("reading sessions: %w", err)
	}

	// Filter to the requested window
	var sessions []*claude.Session
	for _, s := range allSessions {
		if s.StartTime.After(since) || s.StartTime.Equal(since) {
			sessions = append(sessions, s)
		}
	}

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

	for _, s := range sorted {
		growthPct := 0
		if s.FirstInputSize > 0 {
			growthPct = ((s.LastInputSize - s.FirstInputSize) * 100) / s.FirstInputSize
		}

		summary.Sessions = append(summary.Sessions, sessionSummary{
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

	prompt := fmt.Sprintf(`You are a cost advisor for AI coding sessions. Given session analytics from the past %d days plus a historical baseline, identify 2-3 actionable tips to reduce costs.

Use the historical baseline to judge what's normal vs abnormal for this user.

Patterns to look for:
- Sessions with high error counts (agent stuck in retry loops)
- Sessions with many interruptions (user had to repeatedly redirect)
- Sessions with heavy Read/Grep/Glob tool use and high context growth (unfocused exploration)
- Sessions with many turns but low output tokens (spinning wheels)
- Sessions that cost much more than this user's historical average

You MUST use this exact format. No other format is acceptable:

[one line comparing this period vs historical baseline]

---

[Day] [Time] — [Project] — $[Cost]
[What they were doing, from the summary field]
[The problem pattern you identified, with specific numbers]
> Tip: [One concrete, actionable sentence]

---

[Day] [Time] — [Project] — $[Cost]
[What they were doing, from the summary field]
[The problem pattern you identified, with specific numbers]
> Tip: [One concrete, actionable sentence]

---

[repeat for each tip, 2-3 total]

<data>
%s
</data>`, tipsDays, string(data))

	// Show spinner while waiting for Haiku
	fmt.Println()
	start := time.Now()
	stop := startSpinner("Analyzing sessions")
	c := exec.Command("claude", "-p", prompt, "--model", tipsModel)
	out, err := c.Output()
	elapsed := time.Since(start).Round(time.Millisecond)
	stop()
	fmt.Printf("  Analyzed sessions in %s\n\n", elapsed)
	if err != nil {
		return fmt.Errorf("claude: %w", err)
	}
	fmt.Print(string(out))
	return nil
}
