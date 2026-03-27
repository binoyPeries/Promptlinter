package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

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
		reportInternalError(stderr, "failed to parse hook input: %v", err)
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
		reportInternalError(stderr, "failed to init analyzer: %v", err)
		return
	}

	result, err := a.Analyze(input.Prompt)
	if err != nil {
		reportInternalError(stderr, "analysis failed: %v", err)
		return
	}

	d := decision.Decide(cfg, result)

	switch d.Action {
	case decision.ActionTip:
		if _, err := fmt.Fprintln(stderr, d.Tip); err != nil {
			reportInternalError(stderr, "failed to write tip output: %v", err)
			return
		}
	case decision.ActionBlock:
		if err := json.NewEncoder(stdout).Encode(d.Block); err != nil {
			reportInternalError(stderr, "failed to write block output: %v", err)
		}
	}
}

// reportInternalError attempts to write the error message to stderr,
// but if that fails (e.g. due to a broken pipe),
// it falls back to logging the error to a file in the config directory.
// This ensures that we capture internal errors without risking blocking the prompt submission process.
func reportInternalError(stderr io.Writer, format string, args ...any) {
	message := fmt.Sprintf("[PromptLinter] "+format, args...)
	if _, err := fmt.Fprintln(stderr, message); err == nil {
		return
	}

	dir, err := config.ConfigDir()
	if err != nil {
		return
	}

	logPath := filepath.Join(dir, "promptlinter.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()

	timestamp := time.Now().Format(time.RFC3339)
	_, _ = fmt.Fprintf(f, "%s %s\n", timestamp, message)
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
