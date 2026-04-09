package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhubert/insights/internal/claude"
	"github.com/zhubert/insights/internal/display"
)

var rootCmd = &cobra.Command{
	Use:   "insights",
	Short: "Cost analytics for Claude Code sessions",
	RunE:  runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(weekCmd)
}

func runRoot(cmd *cobra.Command, args []string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	todaySessions, err := claude.SessionsSince(today)
	if err != nil {
		return fmt.Errorf("reading sessions: %w", err)
	}

	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	weekSessions, err := claude.SessionsSince(weekStart)
	if err != nil {
		return fmt.Errorf("reading sessions: %w", err)
	}

	// Find most recent session with data
	all, err := claude.AllSessions()
	if err != nil {
		return fmt.Errorf("reading sessions: %w", err)
	}

	if len(all) == 0 {
		fmt.Println("No Claude Code sessions found.")
		return nil
	}

	last := all[len(all)-1]

	fmt.Printf("\n")
	fmt.Printf("  Last session:  %s  (%d turns, %s in / %s out)  %s\n",
		display.FormatCost(last.Cost),
		last.Turns,
		display.FormatTokens(last.Usage.InputTokens+last.Usage.CacheReadInputTokens+last.Usage.CacheCreationInputTokens),
		display.FormatTokens(last.Usage.OutputTokens),
		last.Project,
	)

	var todayCost float64
	for _, s := range todaySessions {
		todayCost += s.Cost
	}
	fmt.Printf("  Today:         %s  across %d sessions\n",
		display.FormatCost(todayCost),
		len(todaySessions),
	)

	var weekCost float64
	for _, s := range weekSessions {
		weekCost += s.Cost
	}
	fmt.Printf("  This week:     %s  across %d sessions\n",
		display.FormatCost(weekCost),
		len(weekSessions),
	)
	fmt.Printf("\n")

	return nil
}
