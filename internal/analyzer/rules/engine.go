package rules

import (
	"fmt"
	"sync"

	"promptlinter/internal/tokenizer"
)

// EngineConfig holds configurable thresholds for the engine.
type EngineConfig struct {
	TipThreshold      int  // wasted tokens >= this → RecommendTip
	EscalateThreshold int  // wasted tokens >= this → RecommendEscalate
	EscalateOnFlags   bool // whether indirect flags trigger escalation
}

// DefaultEngineConfig returns the standard thresholds.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		TipThreshold:      20,
		EscalateThreshold: 100,
		EscalateOnFlags:   true,
	}
}

// Recommendation indicates what action to take based on analysis results.
type Recommendation int

const (
	RecommendLog      Recommendation = iota // < 20 wasted tokens, no flags
	RecommendTip                            // 20-100 wasted tokens, no flags
	RecommendEscalate                       // > 100 tokens OR has indirect flags
)

func (r Recommendation) String() string {
	switch r {
	case RecommendLog:
		return "log"
	case RecommendTip:
		return "tip"
	case RecommendEscalate:
		return "escalate"
	default:
		return "unknown"
	}
}

// EngineResult is the aggregate output from running all detectors.
type EngineResult struct {
	TotalWastedTokens int
	Results           []*Result
	Recommendation    Recommendation
}

// Issues flattens all issues from all detector results.
func (er *EngineResult) Issues() []Issue {
	var out []Issue
	for _, r := range er.Results {
		out = append(out, r.Issues...)
	}
	return out
}

// Flags flattens all flags from all detector results.
func (er *EngineResult) Flags() []Flag {
	var out []Flag
	for _, r := range er.Results {
		out = append(out, r.Flags...)
	}
	return out
}

// Engine runs all detectors in parallel and aggregates results.
type Engine struct {
	detectors []Detector
	cfg       EngineConfig
}

// NewEngine creates an Engine with the standard set of detectors.
func NewEngine(counter *tokenizer.Counter, cfg EngineConfig) *Engine {
	return &Engine{
		cfg: cfg,
		detectors: []Detector{
			NewFillerDetector(counter),
			NewMetaCommentaryDetector(counter),
			NewContextDumpDetector(counter),
			NewIndirectFlagsDetector(counter),
		},
	}
}

type detectorOutput struct {
	name   string
	result *Result
	err    error
}

// Run executes all detectors in parallel and returns the aggregate result.
func (e *Engine) Run(prompt string) (*EngineResult, error) {
	ch := make(chan detectorOutput, len(e.detectors))
	var wg sync.WaitGroup

	for _, d := range e.detectors {
		wg.Add(1)
		go func(det Detector) {
			defer wg.Done()
			r, err := det.Detect(prompt)
			ch <- detectorOutput{name: det.Name(), result: r, err: err}
		}(d)
	}

	wg.Wait()
	close(ch)

	engineResult := &EngineResult{}

	for out := range ch {
		if out.err != nil {
			return nil, fmt.Errorf("detector %q failed: %w", out.name, out.err)
		}
		engineResult.Results = append(engineResult.Results, out.result)
		engineResult.TotalWastedTokens += out.result.WastedTokens
	}

	engineResult.Recommendation = e.recommend(engineResult.TotalWastedTokens, engineResult.Flags())

	return engineResult, nil
}

func (e *Engine) recommend(wastedTokens int, flags []Flag) Recommendation {
	if wastedTokens >= e.cfg.EscalateThreshold {
		return RecommendEscalate
	}
	if e.cfg.EscalateOnFlags && len(flags) > 0 {
		return RecommendEscalate
	}
	if wastedTokens >= e.cfg.TipThreshold {
		return RecommendTip
	}
	return RecommendLog
}
