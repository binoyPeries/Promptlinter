package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"promptlinter/internal/config"
	"promptlinter/internal/hook"
)

// Set by GoReleaser ldflags.
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "plint",
		Short: "Intercepts Claude Code prompts, detects token waste, and suggests rewrites",
	}

	root.AddCommand(analyzeCmd())
	root.AddCommand(versionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the installed version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}

func analyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "analyze",
		Short:         "Analyze a prompt from the UserPromptSubmit hook",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Guard against recursive invocation: when plint escalates to the LLM layer it
			// shells out to `claude -p`, which fires the UserPromptSubmit hook again creating a loop.
			if os.Getenv("PLINT_INTERNAL") != "" {
				return
			}
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[PromptLinter] config error: %v\n", err)
				return
			}
			hook.HandleAnalyze(cfg, os.Stdin, os.Stdout, os.Stderr)
		},
	}
}
