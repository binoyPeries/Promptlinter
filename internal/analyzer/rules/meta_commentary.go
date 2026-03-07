package rules

import (
	"regexp"
	"strings"

	"promptlinter/internal/tokenizer"
)

// metaPattern defines a regex pattern and its suggestion.
type metaPattern struct {
	re         *regexp.Regexp
	suggestion string
}

// MetaCommentaryDetector detects preamble and meta-commentary that adds no
// information — the user describing what they're about to ask instead of asking it.
type MetaCommentaryDetector struct {
	patterns []metaPattern
	counter  *tokenizer.Counter
}

// NewMetaCommentaryDetector creates a MetaCommentaryDetector with all meta-commentary patterns.
func NewMetaCommentaryDetector(counter *tokenizer.Counter) *MetaCommentaryDetector {
	return &MetaCommentaryDetector{
		counter: counter,
		patterns: []metaPattern{
			// Preamble — describing the ask
			{regexp.MustCompile(`(?i)^(let me (explain|describe|tell you|walk you through))\s+`), "Skip the preamble — state it directly"},
			{regexp.MustCompile(`(?i)^(here'?s what i\s+\w+)\b`), "Skip the preamble — state it directly"},
			{regexp.MustCompile(`(?i)^(here'?s my (question|request|ask|problem|issue))\b`), "Skip the preamble — state it directly"},
			{regexp.MustCompile(`(?i)^(what i('m| am) (trying|looking|wanting|hoping) to do is)\s+`), "State what you need directly"},
			{regexp.MustCompile(`(?i)^(so basically what i need is)\s+`), "State what you need directly"},
			{regexp.MustCompile(`(?i)^(i have a (question|request|task|problem|issue))\b`), "Skip — just ask the question"},

			// Context narration
			{regexp.MustCompile(`(?i)^(for (some |a bit of )?context,?\s+)`), "Remove — provide the context directly"},
			{regexp.MustCompile(`(?i)^(to give you (some |a bit of )?(context|background),?\s+)`), "Remove — provide the context directly"},
			{regexp.MustCompile(`(?i)^(some (context|background):?\s+)`), "Remove — provide the context directly"},
			{regexp.MustCompile(`(?i)^(just to (give you |provide )?(some )?(context|background),?\s+)`), "Remove — provide the context directly"},

			// Self-referential meta
			{regexp.MustCompile(`(?i)\b(i('m| am) going to (ask|tell|describe|explain|show))\b`), "Skip — just do it"},
			{regexp.MustCompile(`(?i)\b(before i (ask|start|begin|get into it),?\s+)`), "Remove — get to the point"},
			{regexp.MustCompile(`(?i)\b(first,? let me (explain|describe|tell you|give you some context))\b`), "Skip — state it directly"},
			{regexp.MustCompile(`(?i)\b(let me (start|begin) by (explaining|describing|saying))\b`), "Skip — state it directly"},

			// Task framing
			{regexp.MustCompile(`(?i)^(i('d| would) like you to)\s+`), "Start with the verb directly"},
			{regexp.MustCompile(`(?i)^(what i('d| would) like is)\s+`), "State it directly"},
			{regexp.MustCompile(`(?i)^(the (task|thing|goal|idea) (is|here is))\s+`), "State the task directly"},
			{regexp.MustCompile(`(?i)^(my (goal|objective|aim|idea) is to)\s+`), "Start with the verb directly"},
		},
	}
}

var _ Detector = (*MetaCommentaryDetector)(nil)

func (d *MetaCommentaryDetector) Name() string {
	return "meta_commentary"
}

func (d *MetaCommentaryDetector) Detect(prompt string) (*Result, error) {
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
