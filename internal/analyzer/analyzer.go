package analyzer

import (
	"fmt"

	"promptlinter/internal/analyzer/rules"
	"promptlinter/internal/config"
	"promptlinter/internal/tokenizer"
)

// AnalysisResult is the full output from analyzing a prompt.
type AnalysisResult struct {
	TotalTokens  int
	WastedTokens int
	Issues       []rules.Issue
	Flags        []rules.Flag
}

// Analyzer orchestrates Layer 1 (and eventually Layer 2) analysis.
type Analyzer struct {
	engine  *rules.Engine
	counter *tokenizer.Counter
	cfg     *config.Config
}

// New creates an Analyzer from the given config.
func New(cfg *config.Config) (*Analyzer, error) {
	counter, err := tokenizer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to init tokenizer: %w", err)
	}

	return &Analyzer{
		engine:  rules.NewEngine(counter),
		counter: counter,
		cfg:     cfg,
	}, nil
}

// Analyze runs the rules engine on the prompt and returns the result.
func (a *Analyzer) Analyze(prompt string) (*AnalysisResult, error) {
	er, err := a.engine.Run(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze prompt: %w", err)
	}

	return &AnalysisResult{
		TotalTokens:  a.counter.Count(prompt),
		WastedTokens: er.TotalWastedTokens,
		Issues:       er.Issues(),
		Flags:        er.Flags(),
	}, nil
}
