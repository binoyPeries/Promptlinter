package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"promptlinter/internal/analyzer"
	"promptlinter/internal/config"
	"promptlinter/internal/decision"
)

// HandleAnalyze is the handler for the UserPromptSubmit hook.
// It reads the hook JSON from stdin, runs the analysis pipeline,
// and writes output to stdout/stderr based on the decision.
// Errors are written to stderr; it never returns a non-nil error
// to avoid blocking prompts due to tool failures.
func HandleAnalyze(cfg *config.Config, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
	var input HookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		fmt.Fprintf(stderr, "[PromptLinter] failed to parse hook input: %v\n", err)
		return
	}

	if input.Prompt == "" {
		return
	}

	if shouldIgnore(cfg, input.Prompt) {
		return
	}

	a, err := analyzer.New(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "[PromptLinter] failed to init analyzer: %v\n", err)
		return
	}

	result, err := a.Analyze(input.Prompt)
	if err != nil {
		fmt.Fprintf(stderr, "[PromptLinter] analysis failed: %v\n", err)
		return
	}

	d := decision.Decide(cfg, result)

	switch d.Action {
	case decision.ActionTip:
		fmt.Fprintln(stderr, d.Tip)
	case decision.ActionBlock:
		if err := json.NewEncoder(stdout).Encode(d.Block); err != nil {
			fmt.Fprintf(stderr, "[PromptLinter] failed to write block output: %v\n", err)
		}
	}
}

// shouldIgnore returns true if the prompt matches any of the configured
// ignored patterns.
func shouldIgnore(cfg *config.Config, prompt string) bool {
	for _, pattern := range cfg.IgnoredPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(prompt) {
			return true
		}
	}
	return false
}
