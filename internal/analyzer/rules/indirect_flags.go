package rules

import (
	"regexp"
	"strings"

	"promptlinter/internal/tokenizer"
)

// overSpecStepThreshold is the minimum number of step markers to flag.
const overSpecStepThreshold = 3

// overSpecStepRatio is the minimum fraction of lines that must be step markers
// to flag as over-specification.
const overSpecStepRatio = 0.50

// indirectPattern pairs a regex with its flag metadata.
type indirectPattern struct {
	re          *regexp.Regexp
	flagType    string
	description string
}

var vaguePatterns = []indirectPattern{
	{regexp.MustCompile(`(?i)\bthe file\b`), "vague_reference", "Specify the file path"},
	{regexp.MustCompile(`(?i)\bthat (file|function|method|class|module|bug|error|issue|thing)\b`), "vague_reference", "Name it explicitly"},
	{regexp.MustCompile(`(?i)\bthe (bug|error|issue|problem)\b`), "vague_reference", "Describe the specific bug or error"},
	{regexp.MustCompile(`(?i)\bfix (it|this|that)\b`), "vague_reference", "Specify what to fix"},
	{regexp.MustCompile(`(?i)\b(it|this|that) (doesn't|does not|isn't|is not) (work|working)\b`), "vague_reference", "Describe what's broken and the expected behavior"},
	{regexp.MustCompile(`(?i)\bthe (code|thing) (I|we) (wrote|did|made)\b`), "vague_reference", "Reference the specific file or function"},
	{regexp.MustCompile(`(?i)\bsomewhere in\b`), "vague_reference", "Specify the exact location"},
	{regexp.MustCompile(`(?i)\bthe (same|usual|normal) (way|thing|approach)\b`), "vague_reference", "Describe the specific approach"},
}

var stepMarkerRe = regexp.MustCompile(`(?im)^\s*(step \d+|first,|second,|third,|then,|next,|finally,|\d+\.)`)

// IndirectFlagsDetector flags vague references and over-specification patterns.
// It produces Flag entries (not token counts) for Layer 2 estimation.
type IndirectFlagsDetector struct {
	counter *tokenizer.Counter
}

// NewIndirectFlagsDetector creates an IndirectFlagsDetector.
func NewIndirectFlagsDetector(counter *tokenizer.Counter) *IndirectFlagsDetector {
	return &IndirectFlagsDetector{counter: counter}
}

var _ Detector = (*IndirectFlagsDetector)(nil)

func (d *IndirectFlagsDetector) Name() string {
	return "indirect_flags"
}

func (d *IndirectFlagsDetector) Detect(prompt string) (*Result, error) {
	result := &Result{DetectorName: d.Name()}

	d.detectVagueReferences(prompt, result)
	d.detectOverSpecification(prompt, result)

	return result, nil
}

// detectVagueReferences flags phrases that rely on implicit context.
func (d *IndirectFlagsDetector) detectVagueReferences(prompt string, result *Result) {
	for _, p := range vaguePatterns {
		matches := p.re.FindAllString(prompt, -1)
		for _, m := range matches {
			result.Flags = append(result.Flags, Flag{
				Type:        p.flagType,
				Match:       strings.TrimSpace(m),
				Description: p.description,
			})
		}
	}
}

// detectOverSpecification flags prompts dominated by step markers.
func (d *IndirectFlagsDetector) detectOverSpecification(prompt string, result *Result) {
	matches := stepMarkerRe.FindAllString(prompt, -1)
	if len(matches) < overSpecStepThreshold {
		return
	}

	lines := strings.Split(strings.TrimSpace(prompt), "\n")
	if len(lines) == 0 {
		return
	}

	ratio := float64(len(matches)) / float64(len(lines))
	if ratio < overSpecStepRatio {
		return
	}

	result.Flags = append(result.Flags, Flag{
		Type:        "over_specification",
		Match:       strings.TrimSpace(matches[0]),
		Description: "Most of the prompt is step-by-step instructions — let the tool decide the order",
	})
}
