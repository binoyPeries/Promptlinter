package analyzer

import (
	"promptlinter/internal/config"
	"testing"
)

func TestAnalyze_CleanPrompt(t *testing.T) {
	a, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	result, err := a.Analyze("Fix the nil pointer in src/auth.go line 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WastedTokens != 0 {
		t.Errorf("WastedTokens = %d, want 0", result.WastedTokens)
	}
	if result.TotalTokens == 0 {
		t.Error("TotalTokens = 0, want > 0")
	}
	if len(result.Issues) != 0 {
		t.Errorf("Issues = %d, want 0", len(result.Issues))
	}
}

func TestAnalyze_FillerPrompt(t *testing.T) {
	a, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	result, err := a.Analyze("Can you please look at the code and fix it, thanks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WastedTokens == 0 {
		t.Error("WastedTokens = 0, want > 0 for filler prompt")
	}
	if len(result.Issues) == 0 {
		t.Error("Issues empty, want filler issues")
	}
}

func TestAnalyze_VaguePrompt_HasFlags(t *testing.T) {
	a, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	result, err := a.Analyze("Fix the bug in the file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Flags) == 0 {
		t.Error("expected flags for vague prompt")
	}
}

func TestAnalyze_TotalTokensCounted(t *testing.T) {
	a, err := New(config.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	result, err := a.Analyze("Add rate limiting to the login endpoint in src/auth/login.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalTokens == 0 {
		t.Error("TotalTokens = 0, want > 0")
	}
}
