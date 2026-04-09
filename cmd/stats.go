package cmd

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zhubert/tokencoach/internal/claude"
	"github.com/zhubert/tokencoach/internal/display"
)

var statsDays int

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Daily cost breakdown with historical comparison",
	RunE:  runStats,
}

func init() {
	statsCmd.Flags().IntVar(&statsDays, "days", 0, "Number of days to show (default: current week)")
}

func runStats(cmd *cobra.Command, args []string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var since time.Time
	var daysInPeriod int
	if statsDays > 0 {
		since = today.AddDate(0, 0, -(statsDays - 1))
		daysInPeriod = statsDays
	} else {
		since = today.AddDate(0, 0, -int(today.Weekday()))
		daysInPeriod = int(today.Weekday()) + 1
	}
	debug("stats: period=%s to %s (%d days)", since.Format("2006-01-02"), today.Format("2006-01-02"), daysInPeriod)

	// Pull all sessions for historical baseline
	allSessions, err := claude.AllSessions()
	if err != nil {
		fmt.Println("No Claude Code session data found. Use Claude Code first, then try again.")
		return nil
	}
	debug("stats: loaded %d total sessions", len(allSessions))

	if len(allSessions) == 0 {
		fmt.Println("No Claude Code sessions found. Use Claude Code first, then try again.")
		return nil
	}

	// Filter to period
	var sessions []*claude.Session
	for _, s := range allSessions {
		if s.StartTime.After(since) || s.StartTime.Equal(since) {
			sessions = append(sessions, s)
		}
	}
	debug("stats: %d sessions in period", len(sessions))

	if len(sessions) == 0 {
		if statsDays > 0 {
			fmt.Printf("\n  No sessions in the last %d days.\n\n", statsDays)
		} else {
			fmt.Print("\n  No sessions this week.\n\n")
		}
		return nil
	}

	// Compute historical baseline
	var histTotal float64
	var histDays int
	for _, s := range allSessions {
		histTotal += s.Cost
	}
	if len(allSessions) > 0 {
		first := allSessions[0].StartTime
		histDays = int(now.Sub(first).Hours()/24) + 1
	}
	histAvgDaily := histTotal / float64(histDays)
	histAvgSession := histTotal / float64(len(allSessions))

	type dayStats struct {
		cost     float64
		sessions int
		tokens   int
	}

	days := make(map[string]*dayStats)
	for _, s := range sessions {
		key := s.StartTime.Local().Format("2006-01-02")
		d, ok := days[key]
		if !ok {
			d = &dayStats{}
			days[key] = d
		}
		d.cost += s.Cost
		d.sessions++
		d.tokens += s.Usage.InputTokens + s.Usage.OutputTokens +
			s.Usage.CacheReadInputTokens + s.Usage.CacheCreationInputTokens
	}

	// Find max cost for bar scaling
	var maxCost float64
	for _, d := range days {
		if d.cost > maxCost {
			maxCost = d.cost
		}
	}

	var totalCost float64
	var totalSessions int

	labelStyle := lipgloss.NewStyle().Bold(true)
	costStyle := lipgloss.NewStyle().Foreground(display.ColorAccent)
	dimStyle := lipgloss.NewStyle().Foreground(display.ColorDim)

	fmt.Printf("\n")
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for i := 0; i < daysInPeriod; i++ {
		date := since.AddDate(0, 0, i)
		key := date.Format("2006-01-02")
		d := days[key]

		var label string
		if statsDays > 14 {
			label = date.Format("Jan 02")
		} else {
			label = dayNames[int(date.Weekday())]
		}

		styledLabel := labelStyle.Render(fmt.Sprintf("%-6s", label))

		if d != nil {
			bar := display.Bar(d.cost, maxCost, 20)
			sessLabel := "sessions"
			if d.sessions == 1 {
				sessLabel = "session "
			}
			styledCost := costStyle.Render(fmt.Sprintf("%7s", display.FormatCost(d.cost)))
			meta := dimStyle.Render(fmt.Sprintf("(%3d %s, %6s tokens)", d.sessions, sessLabel, display.FormatTokens(d.tokens)))
			fmt.Printf("  %s %s  %s  %s\n", styledLabel, bar, styledCost, meta)
			totalCost += d.cost
			totalSessions += d.sessions
		} else {
			bar := display.Bar(0, maxCost, 20)
			styledCost := dimStyle.Render(fmt.Sprintf("%7s", "$0.00"))
			fmt.Printf("  %s %s  %s\n", styledLabel, bar, styledCost)
		}
	}

	fmt.Println()
	if totalSessions > 0 {
		periodAvgSession := totalCost / float64(totalSessions)
		periodAvgDaily := totalCost / float64(daysInPeriod)

		w := display.GetTermWidth()
		boxWidth := w - 4
		if boxWidth > 50 {
			boxWidth = 50
		}

		lines := []string{
			fmt.Sprintf("Total:       %8s across %d sessions", display.FormatCost(totalCost), totalSessions),
			fmt.Sprintf("Avg/session: %8s %s", display.FormatCost(periodAvgSession),
				dimStyle.Render(fmt.Sprintf("(historical: %8s)", display.FormatCost(histAvgSession)))),
			fmt.Sprintf("Avg/day:     %8s %s", display.FormatCost(periodAvgDaily),
				dimStyle.Render(fmt.Sprintf("(historical: %8s)", display.FormatCost(histAvgDaily)))),
		}
		fmt.Println(display.RoundedBox(strings.Join(lines, "\n"), boxWidth))
	}
	fmt.Println()

	return nil
}
