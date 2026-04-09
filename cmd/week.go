package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhubert/insights/internal/claude"
	"github.com/zhubert/insights/internal/display"
)

var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show daily cost breakdown for the current week",
	RunE:  runWeek,
}

func runWeek(cmd *cobra.Command, args []string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	sessions, err := claude.SessionsSince(weekStart)
	if err != nil {
		return fmt.Errorf("reading sessions: %w", err)
	}

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

	fmt.Printf("\n")
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for i := 0; i <= int(today.Weekday()); i++ {
		date := weekStart.AddDate(0, 0, i)
		key := date.Format("2006-01-02")
		d := days[key]

		dayName := dayNames[int(date.Weekday())]

		if d != nil {
			bar := display.Bar(d.cost, maxCost, 20)
			fmt.Printf("  %s  %s  %7s  (%d sessions, %s tokens)\n",
				dayName, bar, display.FormatCost(d.cost), d.sessions, display.FormatTokens(d.tokens))
			totalCost += d.cost
			totalSessions += d.sessions
		} else {
			bar := display.Bar(0, maxCost, 20)
			fmt.Printf("  %s  %s  %7s\n", dayName, bar, "$0.00")
		}
	}

	fmt.Printf("\n")
	if totalSessions > 0 {
		fmt.Printf("  Total:   %s across %d sessions\n", display.FormatCost(totalCost), totalSessions)
		fmt.Printf("  Avg/session: %s\n", display.FormatCost(totalCost/float64(totalSessions)))
	}
	fmt.Printf("\n")

	return nil
}
