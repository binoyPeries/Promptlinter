package rules

// Detector is the interface every rules engine detector must implement.
type Detector interface {
	Name() string
	Detect(prompt string) (*Result, error)
}

// Result is what a detector returns after analyzing a prompt.
type Result struct {
	DetectorName string
	Issues       []Issue
	WastedTokens int
}

// Issue is a single problem found in the prompt.
type Issue struct {
	Type       string // "filler", "redundancy", "meta_commentary", "context_dumping"
	Match      string // the matched text
	Suggestion string // what to do instead
	Tokens     int    // tokens wasted by this match
}
