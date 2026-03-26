package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"promptlinter/internal/config"
	"promptlinter/internal/hook"
)

func main() {
	root := &cobra.Command{
		Use:   "plint",
		Short: "Intercepts Claude Code prompts, detects token waste, and suggests rewrites",
	}

	root.AddCommand(analyzeCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func analyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "analyze",
		Short:         "Analyze a prompt from the UserPromptSubmit hook",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[PromptLinter] config error: %v\n", err)
				return
			}
			hook.HandleAnalyze(cfg, os.Stdin, os.Stdout, os.Stderr)
		},
	}
}
