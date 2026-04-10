package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhubert/tokencoach/internal/claude"
)

var debugMode bool
var configDir string

var rootCmd = &cobra.Command{
	Use:               "tokencoach",
	Short:             "AI-powered coaching for your Claude Code usage",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if configDir != "" {
			claude.ConfigDirOverride = configDir
		}
	},
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
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Claude config directory (overrides CLAUDE_CONFIG_DIR)")
	rootCmd.AddCommand(statsCmd)
}
