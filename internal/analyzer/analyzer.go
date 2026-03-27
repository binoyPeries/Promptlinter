package analyzer

import (
	"fmt"

	"promptlinter/internal/analyzer/llm"
	"promptlinter/internal/analyzer/rules"
	"promptlinter/internal/config"
	"promptlinter/internal/tokenizer"
)

// AnalysisResult is the full output from analyzing a prompt.
type AnalysisResult struct {
	TotalTokens   int
	WastedTokens  int
	Issues        []rules.Issue
	Flags         []rules.Flag
	LLMSuggestion string // empty if LLM analysis was not run or failed
	TokensSaved   int    // original_tokens - suggestion_tokens; 0 if no LLM result
	LLMError      string // non-empty if LLM was attempted but failed
}

// Analyzer orchestrates Layer 1 (rules engine) and Layer 2 (LLM) analysis.
type Analyzer struct {
	engine  *rules.Engine
	counter *tokenizer.Counter
	cfg     *config.Config
	llm     llm.Analyzer // nil if LLM analysis is disabled
}

// New creates an Analyzer from the given config.
func New(cfg *config.Config) (*Analyzer, error) {
	counter, err := tokenizer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to init tokenizer: %w", err)
	}

	a := &Analyzer{
		engine:  rules.NewEngine(counter),
		counter: counter,
		cfg:     cfg,
	}

	if cfg.LLMEnabled {
		a.llm = llm.NewClaudeAnalyzer(cfg.LLMModel, counter)
	}

	return a, nil
}

// Analyze runs Layer 1 rules analysis, and Layer 2 LLM analysis if enabled and warranted.
func (a *Analyzer) Analyze(prompt string) (*AnalysisResult, error) {
	er, err := a.engine.Run(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze prompt: %w", err)
	}

	result := &AnalysisResult{
		TotalTokens:  a.counter.Count(prompt),
		WastedTokens: er.TotalWastedTokens,
		Issues:       er.Issues(),
		Flags:        er.Flags(),
	}

	if a.llm != nil && shouldEscalate(a.cfg, result) {
		lr, err := a.llm.Analyze(prompt, result.Issues, result.Flags)
		if err != nil {
			result.LLMError = err.Error()
			return result, nil // return Layer 1 results even if LLM fails
		}
		result.LLMSuggestion = lr.Suggestion
		result.TokensSaved = lr.TokensSaved
	}

	return result, nil
}

// shouldEscalate returns true if the prompt warrants LLM analysis.
func shouldEscalate(cfg *config.Config, ar *AnalysisResult) bool {
	if ar.WastedTokens >= cfg.EscalationThreshold {
		return true
	}
	return cfg.EscalateOnIndirectFlags && len(ar.Flags) > 0
}
