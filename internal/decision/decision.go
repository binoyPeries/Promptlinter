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
	case config.ModeOff, config.ModeSilent:
		return &Result{Action: ActionPass}

	case config.ModeAuto:
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

// tokenDisplayThreshold is the minimum token count to show in the header.
const tokenDisplayThreshold = 10

// formatFeedback produces a compact feedback string with a single header,
// bullet-pointed issues/flags, and an LLM suggestion.
func formatFeedback(result *analyzer.AnalysisResult) string {
	var lines []string

	// Header: issue count + token savings (only if significant).
	issueCount := len(result.Issues) + len(result.Flags)
	tokensSaved := result.TokensSaved
	if tokensSaved == 0 {
		tokensSaved = result.WastedTokens
	}
	if tokensSaved > tokenDisplayThreshold {
		lines = append(lines, fmt.Sprintf("⚡ PromptLinter — %d issues, ~%d tokens saved", issueCount, tokensSaved))
	} else {
		lines = append(lines, fmt.Sprintf("⚡ PromptLinter — %d issues", issueCount))
	}

	// Issues (from rules engine).
	for _, issue := range result.Issues {
		lines = append(lines, fmt.Sprintf("  • %s (%s)", issue.Suggestion, issue.Type))
	}

	// Flags (quality warnings).
	for _, flag := range result.Flags {
		lines = append(lines, fmt.Sprintf("  • %s (%s)", flag.Description, flag.Type))
	}

	// LLM suggestion.
	if result.LLMSuggestion != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  💡 Try: %s", result.LLMSuggestion))
	}

	// LLM error.
	if result.LLMError != "" {
		lines = append(lines, fmt.Sprintf("  ⚠ AI analysis failed: %s", result.LLMError))
	}

	return strings.Join(lines, "\n")
}
