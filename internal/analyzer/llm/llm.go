package llm

import (
	"promptlinter/internal/analyzer/rules"
)

// Result holds the output from the LLM analysis layer.
type Result struct {
	Suggestion  string  // LLM's suggested improvement
	TokensSaved int     // original_tokens - suggestion_tokens
	CostUSD     float64 // reported by the underlying LLM provider
}

// Analyzer is the interface for LLM-based prompt analysis.
// Implementations are provider-specific (e.g. ClaudeAnalyzer).
type Analyzer interface {
	Analyze(prompt string, issues []rules.Issue, flags []rules.Flag) (*Result, error)
}
