package rules

import (
	"testing"

	"promptlinter/internal/tokenizer"
)

func newTestFillerDetector(t *testing.T) *FillerDetector {
	t.Helper()
	counter, err := tokenizer.New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}
	return NewFillerDetector(counter)
}

func TestFillerDetector_Name(t *testing.T) {
	d := newTestFillerDetector(t)
	if d.Name() != "filler" {
		t.Errorf("expected name 'filler', got %q", d.Name())
	}
}

func TestFillerDetector_CleanPrompt(t *testing.T) {
	d := newTestFillerDetector(t)
	result, err := d.Detect("Fix the login bug in src/auth.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues for clean prompt, got %d: %v", len(result.Issues), result.Issues)
	}
	if result.WastedTokens != 0 {
		t.Errorf("expected 0 wasted tokens, got %d", result.WastedTokens)
	}
}

func TestFillerDetector_Politeness(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"can you", "Can you fix the login bug"},
		{"could you", "Could you look at this error"},
		{"please", "Please update the tests"},
		{"would you", "Would you refactor this function"},
		{"i need you to", "I need you to add error handling"},
		{"i want you to", "I want you to fix this"},
	}

	d := newTestFillerDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected filler detected in %q, got none", tt.prompt)
			}
			if result.WastedTokens == 0 {
				t.Errorf("expected wasted tokens > 0 for %q", tt.prompt)
			}
		})
	}
}

func TestFillerDetector_Greetings(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"hey", "Hey, fix the login bug"},
		{"hi", "Hi can you look at this"},
		{"hello", "Hello, update the tests"},
	}

	d := newTestFillerDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			hasGreeting := false
			for _, issue := range result.Issues {
				if issue.Suggestion == "Drop the greeting" {
					hasGreeting = true
					break
				}
			}
			if !hasGreeting {
				t.Errorf("expected greeting detected in %q", tt.prompt)
			}
		})
	}
}

func TestFillerDetector_TrailingPoliteness(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"thanks", "Fix the login bug thanks"},
		{"thank you", "Look at this error thank you"},
		{"cheers", "Update the config cheers"},
		{"appreciate it", "Fix the tests appreciate it"},
	}

	d := newTestFillerDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected trailing filler in %q, got none", tt.prompt)
			}
		})
	}
}

func TestFillerDetector_Hedging(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"i think maybe", "I think maybe the bug is in auth.go"},
		{"not sure but", "I'm not sure but the test is failing"},
		{"perhaps", "Perhaps we should refactor this"},
	}

	d := newTestFillerDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected hedging detected in %q, got none", tt.prompt)
			}
		})
	}
}

func TestFillerDetector_FillerWords(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"basically", "Basically the function is broken"},
		{"actually", "The test is actually failing"},
		{"just", "Just run the tests"},
		{"sort of", "It sort of works but not really"},
	}

	d := newTestFillerDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected filler word detected in %q, got none", tt.prompt)
			}
		})
	}
}

func TestFillerDetector_MultipleFiller(t *testing.T) {
	d := newTestFillerDetector(t)
	prompt := "Hey, can you please look at the code and fix it if possible, thanks"
	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) < 2 {
		t.Errorf("expected multiple issues in %q, got %d", prompt, len(result.Issues))
	}
}

func TestFillerDetector_NoFalsePositives(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"adjust not just", "Adjust the config values"},
		{"clean technical", "Refactor handleLogin to use middleware pattern"},
		{"code with keywords", "if err != nil { return err }"},
	}

	d := newTestFillerDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Issues) != 0 {
				t.Errorf("expected no filler in %q, got %v", tt.prompt, result.Issues)
			}
		})
	}
}
