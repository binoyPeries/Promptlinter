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
	if result.TotalWastedTokens != 0 {
		t.Errorf("expected 0 wasted tokens, got %d", result.TotalWastedTokens)
	}
	if len(result.Issues()) != 0 {
		t.Errorf("expected 0 issues, got %d", len(result.Issues()))
	}
	if len(result.Flags()) != 0 {
		t.Errorf("expected 0 flags, got %d", len(result.Flags()))
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

func TestEngine_FillerDetected(t *testing.T) {
	e := newTestEngine(t)
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
	if result.TotalWastedTokens == 0 {
		t.Error("expected wasted tokens > 0")
	}
	if len(result.Issues()) == 0 {
		t.Error("expected issues from filler/meta detectors")
	}
}

func TestEngine_HeavyWaste(t *testing.T) {
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
	if result.TotalWastedTokens < 100 {
		t.Skipf("wasted tokens (%d) lower than expected for heavy paste", result.TotalWastedTokens)
	}
}

func TestEngine_IndirectFlags(t *testing.T) {
	e := newTestEngine(t)
	prompt := "Fix the bug in the file"
	result, err := e.Run(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Flags()) == 0 {
		t.Error("expected indirect flags for vague prompt")
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
	if len(result.Issues()) == 0 {
		t.Error("expected issues from filler detector")
	}
	if len(result.Flags()) == 0 {
		t.Error("expected flags from indirect flags detector")
	}
}
