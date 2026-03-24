package rules

import (
	"fmt"
	"strings"
	"testing"

	"promptlinter/internal/tokenizer"
)

func newTestIndirectFlagsDetector(t *testing.T) *IndirectFlagsDetector {
	t.Helper()
	counter, err := tokenizer.New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}
	return NewIndirectFlagsDetector(counter)
}

func TestIndirectFlagsDetector_Name(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)
	if d.Name() != "indirect_flags" {
		t.Errorf("expected name 'indirect_flags', got %q", d.Name())
	}
}

func TestIndirectFlagsDetector_VagueReferences(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"the file", "Look at the file and fix the formatting"},
		{"that function", "That function is broken"},
		{"the bug", "Fix the bug"},
		{"fix it", "Just fix it"},
		{"it doesn't work", "It doesn't work anymore"},
		{"this does not work", "This does not work"},
		{"fix this", "Fix this please"},
		{"fix that", "Can you fix that"},
		{"find the bug", "Find the bug and fix it"},
		{"the code we wrote", "The code we wrote has issues"},
		{"somewhere in", "The error is somewhere in the backend"},
		{"the same way", "Do it the same way as before"},
	}

	d := newTestIndirectFlagsDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Flags) == 0 {
				t.Errorf("expected vague reference flag in %q, got none", tt.prompt)
			}
			hasVague := false
			for _, f := range result.Flags {
				if f.Type == "vague_reference" {
					hasVague = true
					break
				}
			}
			if !hasVague {
				t.Errorf("expected flag type 'vague_reference' in %q, got %v", tt.prompt, result.Flags)
			}
		})
	}
}

func TestIndirectFlagsDetector_NoVagueFalsePositives(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"specific file", "Fix the nil pointer in src/auth.go"},
		{"specific function", "Refactor handleLogin to use middleware"},
		{"specific error", "Getting 'connection refused' on port 8080"},
		{"file path reference", "Update the config in ~/.prompt-optimizer/config.json"},
	}

	d := newTestIndirectFlagsDetector(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.Detect(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, f := range result.Flags {
				if f.Type == "vague_reference" {
					t.Errorf("expected no vague flags for %q, got: %v", tt.prompt, f)
				}
			}
		})
	}
}

func TestIndirectFlagsDetector_OverSpecification(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)

	// Short prompt with excessive step markers.
	prompt := `Step 1: Open the config file.
Step 2: Change the port.
Step 3: Save the file.
Step 4: Restart the server.`

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasOverSpec := false
	for _, f := range result.Flags {
		if f.Type == "over_specification" {
			hasOverSpec = true
			break
		}
	}
	if !hasOverSpec {
		t.Error("expected over_specification flag for short step-heavy prompt")
	}
}

func TestIndirectFlagsDetector_OverSpec_NumberedSteps(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)

	prompt := `1. Open the file.
2. Find the function.
3. Add error handling.`

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasOverSpec := false
	for _, f := range result.Flags {
		if f.Type == "over_specification" {
			hasOverSpec = true
			break
		}
	}
	if !hasOverSpec {
		t.Error("expected over_specification flag for numbered steps in short prompt")
	}
}

func TestIndirectFlagsDetector_LongPromptWithStepsIsOK(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)

	// Steps are a minority of lines — ratio is low, should not flag.
	var lines []string
	lines = append(lines, "We need to migrate the database schema from integer IDs to UUIDs.")
	lines = append(lines, "The old schema uses auto-increment integers for the users table.")
	lines = append(lines, "The new schema requires UUIDs for cross-service compatibility.")
	lines = append(lines, "Three tables are affected: users, sessions, and tokens.")
	lines = append(lines, "Foreign keys between these tables also need updating.")
	lines = append(lines, "The rollback plan is to restore from the pre-migration backup.")
	lines = append(lines, "Here is the order of operations:")
	for i := 1; i <= 3; i++ {
		lines = append(lines, fmt.Sprintf("Step %d: Perform migration phase %d.", i, i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range result.Flags {
		if f.Type == "over_specification" {
			t.Errorf("long prompt with steps should not trigger over_specification, got: %v", f)
		}
	}
}

func TestIndirectFlagsDetector_FewStepsIsOK(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)

	// Only 2 steps — below threshold.
	prompt := `Step 1: Fix the auth bug.
Step 2: Run the tests.`

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range result.Flags {
		if f.Type == "over_specification" {
			t.Errorf("2 steps should not trigger over_specification, got: %v", f)
		}
	}
}

func TestIndirectFlagsDetector_NoWastedTokens(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)

	// Even with flags, WastedTokens should be 0 — Layer 2 estimates the cost.
	result, err := d.Detect("Fix the bug in the file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WastedTokens != 0 {
		t.Errorf("expected 0 wasted tokens for indirect flags, got %d", result.WastedTokens)
	}
}

func TestIndirectFlagsDetector_MultipleFlags(t *testing.T) {
	d := newTestIndirectFlagsDetector(t)

	prompt := "Fix the bug in the file, it doesn't work"
	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Flags) < 2 {
		t.Errorf("expected multiple flags in %q, got %d: %v", prompt, len(result.Flags), result.Flags)
	}
}
