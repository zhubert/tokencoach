package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var debugMode bool

var rootCmd = &cobra.Command{
	Use:               "tokencoach",
	Short:             "AI-powered coaching for your Claude Code usage",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func debug(format string, args ...any) {
	if debugMode {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging to stderr")
	rootCmd.AddCommand(statsCmd)
}
