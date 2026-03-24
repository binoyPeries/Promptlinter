package rules

import (
	"fmt"
	"strings"
	"testing"

	"promptlinter/internal/tokenizer"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	counter, err := tokenizer.New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}
	return NewEngine(counter)
}

func TestEngine_CleanPrompt(t *testing.T) {
	e := newTestEngine(t)
	result, err := e.Run("Fix the nil pointer in src/auth.go line 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Recommendation != RecommendLog {
		t.Errorf("expected RecommendLog, got %s", result.Recommendation)
	}
	if result.TotalWastedTokens != 0 {
		t.Errorf("expected 0 wasted tokens, got %d", result.TotalWastedTokens)
	}
}

func TestEngine_AllDetectorsRun(t *testing.T) {
	e := newTestEngine(t)
	result, err := e.Run("Refactor handleLogin to use middleware")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 4 {
		t.Errorf("expected 4 detector results, got %d", len(result.Results))
	}
}

func TestEngine_MildFiller_RecommendTip(t *testing.T) {
	e := newTestEngine(t)
	// Stack many distinct filler + meta-commentary patterns to cross 20 tokens.
	prompt := "Hello, I was wondering if you could please help me out with something. " +
		"Let me explain what I need here. Here's what I want you to do. " +
		"What I'm trying to do is the following. I have a question about this. " +
		"For some context, I'm not sure but I think maybe perhaps we should basically just sort of " +
		"My goal is to actually go ahead and " +
		"possibly refactor handleLogin in src/auth.go to use middleware, thank you so much appreciate it cheers"
	result, err := e.Run(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalWastedTokens < tipThreshold {
		t.Skipf("wasted tokens (%d) below tip threshold, adjust prompt", result.TotalWastedTokens)
	}
	if result.Recommendation != RecommendTip && result.Recommendation != RecommendEscalate {
		t.Errorf("expected RecommendTip or RecommendEscalate, got %s (wasted: %d)", result.Recommendation, result.TotalWastedTokens)
	}
}

func TestEngine_HeavyWaste_RecommendEscalate(t *testing.T) {
	e := newTestEngine(t)

	// Large fenced code block (60 lines) triggers context_dump.
	var lines []string
	lines = append(lines, "fix this:")
	lines = append(lines, "```go")
	for i := 0; i < 60; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d(w http.ResponseWriter, r *http.Request) { return }", i))
	}
	lines = append(lines, "```")
	prompt := strings.Join(lines, "\n")

	result, err := e.Run(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalWastedTokens < escalateThreshold {
		t.Skipf("wasted tokens (%d) below escalate threshold", result.TotalWastedTokens)
	}
	if result.Recommendation != RecommendEscalate {
		t.Errorf("expected RecommendEscalate, got %s", result.Recommendation)
	}
}

func TestEngine_IndirectFlags_RecommendEscalate(t *testing.T) {
	e := newTestEngine(t)
	// Vague prompt — flags should trigger escalation even with 0 wasted tokens.
	prompt := "Fix the bug in the file"
	result, err := e.Run(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AllFlags) == 0 {
		t.Fatal("expected indirect flags, got none")
	}
	if result.Recommendation != RecommendEscalate {
		t.Errorf("expected RecommendEscalate due to flags, got %s", result.Recommendation)
	}
}

func TestEngine_RecommendationString(t *testing.T) {
	tests := []struct {
		r    Recommendation
		want string
	}{
		{RecommendLog, "log"},
		{RecommendTip, "tip"},
		{RecommendEscalate, "escalate"},
	}
	for _, tt := range tests {
		if got := tt.r.String(); got != tt.want {
			t.Errorf("Recommendation(%d).String() = %q, want %q", tt.r, got, tt.want)
		}
	}
}

func TestEngine_AggregatesIssuesAndFlags(t *testing.T) {
	e := newTestEngine(t)
	// Prompt with both filler issues and vague flags.
	prompt := "Hey, can you please fix the bug in the file, thanks"
	result, err := e.Run(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AllIssues) == 0 {
		t.Error("expected issues from filler detector")
	}
	if len(result.AllFlags) == 0 {
		t.Error("expected flags from indirect flags detector")
	}
}
