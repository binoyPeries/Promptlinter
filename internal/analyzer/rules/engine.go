package rules

import (
	"fmt"
	"sync"

	"promptlinter/internal/tokenizer"
)

// EngineResult is the aggregate output from running all detectors.
type EngineResult struct {
	TotalWastedTokens int
	Results           []*Result
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

	return engineResult, nil
}
