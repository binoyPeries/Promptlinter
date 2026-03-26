package decision

import (
	"fmt"
	"strings"

	"promptlinter/internal/analyzer"
	"promptlinter/internal/config"
)

// Action represents what the hook should do.
type Action int

const (
	ActionPass  Action = iota // empty stdout + stderr, prompt goes through
	ActionTip                 // feedback to stderr, empty stdout, prompt goes through
	ActionBlock               // JSON to stdout with decision:"block", prompt killed
)

// BlockOutput is the JSON structure written to stdout when blocking a prompt.
type BlockOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// Result holds the decision output. For ActionTip the feedback goes to stderr.
// For ActionBlock the BlockOutput goes to stdout as JSON.
type Result struct {
	Action Action
	Tip    string       // stderr text (ActionTip only)
	Block  *BlockOutput // stdout JSON (ActionBlock only)
}

// shouldBlock returns true if the analysis warrants blocking in auto mode.
func shouldBlock(cfg *config.Config, ar *analyzer.AnalysisResult) bool {
	if ar.WastedTokens >= cfg.EscalationThreshold {
		return true
	}
	if cfg.EscalateOnIndirectFlags && len(ar.Flags) > 0 {
		return true
	}
	return false
}

// hasFeedback returns true if the analysis has something worth showing.
func hasFeedback(cfg *config.Config, ar *analyzer.AnalysisResult) bool {
	if ar.WastedTokens >= cfg.TipThreshold {
		return true
	}
	if len(ar.Flags) > 0 {
		return true
	}
	return false
}

// Decide takes analysis results and config mode, returns the decision.
func Decide(cfg *config.Config, ar *analyzer.AnalysisResult) *Result {
	switch cfg.Mode {
	case "silent":
		return &Result{Action: ActionPass}

	case "auto":
		if shouldBlock(cfg, ar) {
			return &Result{
				Action: ActionBlock,
				Block: &BlockOutput{
					Decision: "block",
					Reason:   formatFeedback(ar),
				},
			}
		}
		return &Result{Action: ActionPass}

	default: // "suggest"
		if hasFeedback(cfg, ar) {
			return &Result{Action: ActionTip, Tip: formatFeedback(ar)}
		}
		return &Result{Action: ActionPass}
	}
}

// formatFeedback produces a feedback string with separate sections for
// token waste (issues) and structural warnings (flags).
func formatFeedback(result *analyzer.AnalysisResult) string {
	var parts []string

	if len(result.Issues) > 0 {
		parts = append(parts, fmt.Sprintf("[PromptLinter] ~%d tokens wasted.", result.WastedTokens))
		for _, issue := range result.Issues {
			parts = append(parts, fmt.Sprintf("  - %s: %q → %s", issue.Type, issue.Match, issue.Suggestion))
		}
	}

	if len(result.Flags) > 0 {
		parts = append(parts, "[PromptLinter] Prompt quality warnings:")
		for _, flag := range result.Flags {
			parts = append(parts, fmt.Sprintf("  - %s", flag.Description))
		}
	}

	return strings.Join(parts, "\n")
}
