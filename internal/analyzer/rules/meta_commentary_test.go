package rules

import (
	"testing"

	"promptlinter/internal/tokenizer"
)

func newTestMetaCommentaryDetector(t *testing.T) *MetaCommentaryDetector {
	t.Helper()
	counter, err := tokenizer.New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}
	return NewMetaCommentaryDetector(counter)
}

func TestMetaCommentaryDetector_Name(t *testing.T) {
	d := newTestMetaCommentaryDetector(t)
	if d.Name() != "meta_commentary" {
		t.Errorf("expected name 'meta_commentary', got %q", d.Name())
	}
}

func TestMetaCommentaryDetector_CleanPrompt(t *testing.T) {
	d := newTestMetaCommentaryDetector(t)
	result, err := d.Detect("Refactor handleLogin to use middleware pattern")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues for clean prompt, got %d: %v", len(result.Issues), result.Issues)
	}
}

func TestMetaCommentaryDetector_Preamble(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"let me explain", "Let me explain what I need — the auth module is broken"},
		{"let me walk you through", "Let me walk you through the issue"},
		{"heres what i want", "Here's what I want: fix the login flow"},
		{"heres what i need", "Heres what I need: update the tests"},
		{"heres what i think", "Here's what I think about the auth flow"},
		{"heres my problem", "Here's my problem with the auth module"},
		{"what im trying to do", "What I'm trying to do is fix the auth module"},
		{"what im hoping to do", "What I'm hoping to do is improve coverage"},
		{"i have a question", "I have a question about the auth module"},
		{"i have a problem", "I have a problem with the failing tests"},
	}

	d := newTestMetaCommentaryDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected meta-commentary in %q, got none", tt.prompt)
			}
			if result.WastedTokens == 0 {
				t.Errorf("expected wasted tokens > 0 for %q", tt.prompt)
			}
		})
	}
}

func TestMetaCommentaryDetector_ContextNarration(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"for context", "For context, we use Go 1.22 and cobra for CLI"},
		{"for some context", "For some context, the auth module uses JWT"},
		{"for a bit of context", "For a bit of context, we just migrated to Go"},
		{"to give you context", "To give you context, this is a monorepo"},
		{"to give you background", "To give you some background, we migrated from Python"},
		{"some context", "Some context: the tests are flaky on CI"},
		{"just to give context", "Just to give you some context, the service is new"},
	}

	d := newTestMetaCommentaryDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected context narration in %q, got none", tt.prompt)
			}
		})
	}
}

func TestMetaCommentaryDetector_SelfReferential(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"im going to ask", "I'm going to ask you to refactor the auth module"},
		{"im going to show", "I'm going to show you the error"},
		{"before i ask", "Before I ask, the project uses Go modules"},
		{"before i get into it", "Before I get into it, some background"},
		{"first let me explain", "First let me explain the project structure"},
		{"first, let me describe", "First, let me describe what the code does"},
		{"let me start by explaining", "Let me start by explaining the architecture"},
	}

	d := newTestMetaCommentaryDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected self-referential meta in %q, got none", tt.prompt)
			}
		})
	}
}

func TestMetaCommentaryDetector_TaskFraming(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"id like you to", "I'd like you to fix the failing tests"},
		{"what id like is", "What I'd like is a refactored auth module"},
		{"the task is", "The task is to update the CI pipeline"},
		{"my goal is to", "My goal is to improve test coverage"},
	}

	d := newTestMetaCommentaryDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected task framing in %q, got none", tt.prompt)
			}
		})
	}
}

func TestMetaCommentaryDetector_NoFalsePositives(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"direct command", "Fix the login bug in src/auth.go"},
		{"technical question", "Why does handleLogin return 401 for valid tokens?"},
		{"code snippet", "Add error handling to the database query"},
		{"context word in code", "Update the context.WithTimeout call to 30s"},
	}

	d := newTestMetaCommentaryDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) != 0 {
				t.Errorf("expected no issues for %q, got %v", tt.prompt, result.Issues)
			}
		})
	}
}
