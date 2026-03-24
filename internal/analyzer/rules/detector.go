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
	Flags        []Flag
	WastedTokens int
}

// Issue is a single problem found in the prompt.
type Issue struct {
	Type       string // "filler", "meta_commentary", "context_dumping"
	Match      string // the matched text
	Suggestion string // what to do instead
	Tokens     int    // tokens wasted by this match
}

// Flag is a qualitative signal that indicates the prompt may benefit from
// Layer 2 analysis. Flags do not carry token counts — Layer 2 estimates them.
type Flag struct {
	Type        string // "vague_reference", "over_specification"
	Match       string // the matched text
	Description string // human-readable explanation
}
