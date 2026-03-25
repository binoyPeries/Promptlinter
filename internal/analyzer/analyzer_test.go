package analyzer

import (
	"promptlinter/internal/analyzer/rules"
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
	if result.Recommendation != rules.RecommendLog {
		t.Errorf("Recommendation = %s, want log", result.Recommendation)
	}
	if result.WastedTokens != 0 {
		t.Errorf("WastedTokens = %d, want 0", result.WastedTokens)
	}
	if result.TotalTokens == 0 {
		t.Error("TotalTokens = 0, want > 0")
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

func TestAnalyze_CustomThresholds(t *testing.T) {
	cfg := config.DefaultConfig()
	// Set very high tip threshold so mild filler stays at Log.
	cfg.TipThreshold = 500
	cfg.EscalationThreshold = 1000
	cfg.EscalateOnIndirectFlags = false

	a, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	result, err := a.Analyze("Can you please look at the code and fix it, thanks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WastedTokens == 0 {
		t.Skip("no waste detected, cannot test threshold override")
	}
	// With thresholds at 500/1000, mild filler should be Log.
	if result.Recommendation != rules.RecommendLog {
		t.Errorf("Recommendation = %s, want log with high thresholds", result.Recommendation)
	}
}

func TestAnalyze_EscalateOnFlagsDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.EscalateOnIndirectFlags = false

	a, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	// Vague prompt that would normally escalate due to flags.
	result, err := a.Analyze("Fix the bug in the file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Flags) == 0 {
		t.Fatal("expected flags, got none")
	}
	// With EscalateOnFlags disabled and low waste, should not escalate.
	if result.Recommendation == rules.RecommendEscalate {
		t.Errorf("Recommendation = escalate, want log or tip with EscalateOnFlags=false")
	}
}
