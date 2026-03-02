package rules

import (
	"regexp"
	"strings"

	"promptlinter/internal/tokenizer"
)

// fillerPattern defines a regex pattern and its suggestion.
type fillerPattern struct {
	re         *regexp.Regexp
	suggestion string
}

// FillerDetector detects filler language like politeness, hedging, and greetings.
type FillerDetector struct {
	patterns []fillerPattern
	counter  *tokenizer.Counter
}

// NewFillerDetector creates a FillerDetector with all filler patterns.
func NewFillerDetector(counter *tokenizer.Counter) *FillerDetector {
	return &FillerDetector{
		counter: counter,
		patterns: []fillerPattern{
			// Start-of-prompt politeness
			{regexp.MustCompile(`(?i)^(can you|could you)\s+`), "Start with the verb directly"},
			{regexp.MustCompile(`(?i)^please\s+`), "Drop \"please\" — start with the verb"},
			{regexp.MustCompile(`(?i)^(would you|i need you to|i want you to)\s+`), "Start with the verb directly"},
			{regexp.MustCompile(`(?i)^(i was wondering if|would it be possible to|it would be great if)\s+`), "State what you need directly"},

			// Greetings
			{regexp.MustCompile(`(?i)^(hey|hi|hello),?\s*`), "Drop the greeting"},

			// Trailing politeness
			{regexp.MustCompile(`(?i)\s*(thanks|thank you|cheers|appreciate it)\.?\s*$`), "Drop the sign-off"},

			// Hedging (anywhere in prompt)
			{regexp.MustCompile(`(?i)\bi think maybe\b`), "State it directly"},
			{regexp.MustCompile(`(?i)\bi'?m not sure but\b`), "State it directly"},
			{regexp.MustCompile(`(?i)\bit might be\b`), "Be specific about what it is"},
			{regexp.MustCompile(`(?i)\bpossibly\b`), "Be specific"},
			{regexp.MustCompile(`(?i)\bperhaps\b`), "Be specific"},

			// Filler words (anywhere)
			{regexp.MustCompile(`(?i)\bbasically\b`), "Remove \"basically\""},
			{regexp.MustCompile(`(?i)\bactually\b`), "Remove \"actually\""},
			{regexp.MustCompile(`(?i)\bjust\b`), "Remove \"just\""},
			{regexp.MustCompile(`(?i)\bsome kind of\b`), "Be specific about what kind"},
			{regexp.MustCompile(`(?i)\bsort of\b`), "Be specific"},
			{regexp.MustCompile(`(?i)\bor something\b`), "Be specific"},

			// Trailing filler
			{regexp.MustCompile(`(?i)\bif that makes sense\b`), "Remove — it makes sense"},
			{regexp.MustCompile(`(?i)\bif you know what i mean\b`), "Remove — be explicit instead"},
			{regexp.MustCompile(`(?i)\byou know\b`), "Remove \"you know\""},
		},
	}
}

var _ Detector = (*FillerDetector)(nil)

func (d *FillerDetector) Name() string {
	return "filler"
}

func (d *FillerDetector) Detect(prompt string) (*Result, error) {
	result := &Result{DetectorName: d.Name()}
	trimmed := strings.TrimSpace(prompt)

	for _, p := range d.patterns {
		matches := p.re.FindAllString(trimmed, -1)
		for _, match := range matches {
			tokens := d.counter.Count(match)
			result.Issues = append(result.Issues, Issue{
				Type:       d.Name(),
				Match:      match,
				Suggestion: p.suggestion,
				Tokens:     tokens,
			})
			result.WastedTokens += tokens
		}
	}

	return result, nil
}
