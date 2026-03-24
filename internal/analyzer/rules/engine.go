package rules

import (
	"sync"

	"promptlinter/internal/tokenizer"
)

// Thresholds for engine recommendations.
const (
	tipThreshold      = 20  // wasted tokens >= this → RecommendTip
	escalateThreshold = 100 // wasted tokens >= this → RecommendEscalate
)

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
	AllIssues         []Issue
	AllFlags          []Flag
	Results           []*Result
	Recommendation    Recommendation
}

// Engine runs all detectors in parallel and aggregates results.
type Engine struct {
	detectors []Detector
}

// NewEngine creates an Engine with the standard set of detectors.
func NewEngine(counter *tokenizer.Counter) *Engine {
	return &Engine{
		detectors: []Detector{
			NewFillerDetector(counter),
			NewMetaCommentaryDetector(counter),
			NewContextDumpDetector(counter),
			NewIndirectFlagsDetector(counter),
		},
	}
}

type detectorOutput struct {
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
			ch <- detectorOutput{result: r, err: err}
		}(d)
	}

	wg.Wait()
	close(ch)

	engineResult := &EngineResult{}

	for out := range ch {
		if out.err != nil {
			return nil, out.err
		}
		engineResult.Results = append(engineResult.Results, out.result)
		engineResult.TotalWastedTokens += out.result.WastedTokens
		engineResult.AllIssues = append(engineResult.AllIssues, out.result.Issues...)
		engineResult.AllFlags = append(engineResult.AllFlags, out.result.Flags...)
	}

	engineResult.Recommendation = recommend(engineResult.TotalWastedTokens, engineResult.AllFlags)

	return engineResult, nil
}

func recommend(wastedTokens int, flags []Flag) Recommendation {
	if len(flags) > 0 || wastedTokens >= escalateThreshold {
		return RecommendEscalate
	}
	if wastedTokens >= tipThreshold {
		return RecommendTip
	}
	return RecommendLog
}
