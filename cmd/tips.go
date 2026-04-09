package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhubert/insights/internal/claude"
)

var tipsCmd = &cobra.Command{
	Use:   "tips",
	Short: "AI-generated tips to reduce your Claude Code costs",
	RunE:  runTips,
}

type sessionSummary struct {
	Time             string         `json:"time"`
	Project          string         `json:"project"`
	Model            string         `json:"model"`
	Cost             float64        `json:"cost"`
	Turns            int            `json:"turns"`
	Errors           int            `json:"errors"`
	Interruptions    int            `json:"interruptions"`
	ToolCounts       map[string]int `json:"tool_counts"`
	OutputTokens     int            `json:"output_tokens"`
	ContextGrowthPct int            `json:"context_growth_pct"`
}

type weekSummary struct {
	WeekTotal      float64          `json:"week_total"`
	SessionCount   int              `json:"session_count"`
	AvgSessionCost float64          `json:"avg_session_cost"`
	Sessions       []sessionSummary `json:"sessions"`
}

func init() {
	rootCmd.AddCommand(tipsCmd)
}

func runTips(cmd *cobra.Command, args []string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	sessions, err := claude.SessionsSince(weekStart)
	if err != nil {
		return fmt.Errorf("reading sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions this week.")
		return nil
	}

	// Build summary
	var totalCost float64
	for _, s := range sessions {
		totalCost += s.Cost
	}

	summary := weekSummary{
		WeekTotal:      totalCost,
		SessionCount:   len(sessions),
		AvgSessionCost: totalCost / float64(len(sessions)),
	}

	// Sort by cost descending, take top 10
	sorted := make([]*claude.Session, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cost > sorted[j].Cost
	})
	if len(sorted) > 10 {
		sorted = sorted[:10]
	}

	for _, s := range sorted {
		growthPct := 0
		if s.FirstInputSize > 0 {
			growthPct = ((s.LastInputSize - s.FirstInputSize) * 100) / s.FirstInputSize
		}

		summary.Sessions = append(summary.Sessions, sessionSummary{
			Time:             s.StartTime.Local().Format("Mon 3:04pm"),
			Project:          s.Project,
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

	prompt := fmt.Sprintf(`You are a cost advisor for AI coding sessions. Given these session analytics from the past week, identify 2-3 actionable tips to reduce costs. Be specific — reference the session time, project, and the pattern you noticed.

Patterns to look for:
- Sessions with high error counts (agent stuck in retry loops)
- Sessions with many interruptions (user had to repeatedly redirect)
- Sessions with heavy Read/Grep/Glob tool use and high context growth (unfocused exploration)
- Sessions with many turns but low output tokens (spinning wheels)
- Sessions that cost much more than the average

Keep your response under 10 lines. Be direct and practical. No preamble.

<sessions>
%s
</sessions>`, string(data))

	c := exec.Command("claude", "-p", prompt, "--model", "haiku")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Println()
	return c.Run()
}
