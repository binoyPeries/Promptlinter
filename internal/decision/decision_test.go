package decision

import (
	"strings"
	"testing"

	"promptlinter/internal/analyzer"
	"promptlinter/internal/analyzer/rules"
	"promptlinter/internal/config"
)

func modeCfg(mode config.Mode) *config.Config {
	c := config.DefaultConfig()
	c.Mode = mode
	return c
}

func makeResult(wasted int, issues []rules.Issue, flags []rules.Flag) *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		TotalTokens:  100,
		WastedTokens: wasted,
		Issues:       issues,
		Flags:        flags,
	}
}

var sampleIssues = []rules.Issue{
	{Type: "filler", Match: "please", Suggestion: "Drop \"please\"", Tokens: 1},
}

var sampleFlags = []rules.Flag{
	{Type: "vague_reference", Match: "the file", Description: "No file path specified"},
}

// --- Off mode: always pass ---

func TestDecide_OffMode(t *testing.T) {
	for _, wasted := range []int{0, 30, 150} {
		r := Decide(modeCfg(config.ModeOff), makeResult(wasted, sampleIssues, sampleFlags))
		if r.Action != ActionPass {
			t.Errorf("off/wasted=%d: action = %d, want ActionPass", wasted, r.Action)
		}
	}
}

// --- Silent mode: always pass ---

func TestDecide_SilentMode(t *testing.T) {
	for _, wasted := range []int{0, 30, 150} {
		r := Decide(modeCfg(config.ModeSilent), makeResult(wasted, sampleIssues, sampleFlags))
		if r.Action != ActionPass {
			t.Errorf("silent/wasted=%d: action = %d, want ActionPass", wasted, r.Action)
		}
	}
}

// --- Suggest mode: feedback to stderr based on thresholds ---

func TestDecide_SuggestMode_LowWaste(t *testing.T) {
	r := Decide(modeCfg(config.ModeSuggest), makeResult(5, nil, nil))
	if r.Action != ActionPass {
		t.Errorf("action = %d, want ActionPass", r.Action)
	}
}

func TestDecide_SuggestMode_MildWaste(t *testing.T) {
	r := Decide(modeCfg(config.ModeSuggest), makeResult(30, sampleIssues, nil))
	if r.Action != ActionTip {
		t.Errorf("action = %d, want ActionTip", r.Action)
	}
	if r.Tip == "" {
		t.Error("tip is empty")
	}
	if r.Block != nil {
		t.Error("block should be nil for suggest mode")
	}
}

func TestDecide_SuggestMode_FlagsOnly(t *testing.T) {
	// Low waste but flags present — should still show feedback.
	r := Decide(modeCfg(config.ModeSuggest), makeResult(5, nil, sampleFlags))
	if r.Action != ActionTip {
		t.Errorf("action = %d, want ActionTip for flags", r.Action)
	}
}

func TestDecide_SuggestMode_HighWaste(t *testing.T) {
	r := Decide(modeCfg(config.ModeSuggest), makeResult(120, sampleIssues, sampleFlags))
	if r.Action != ActionTip {
		t.Errorf("action = %d, want ActionTip", r.Action)
	}
	if r.Tip == "" {
		t.Error("tip is empty")
	}
}

// --- Auto mode: block on high waste/flags, pass otherwise ---

func TestDecide_AutoMode_LowWaste(t *testing.T) {
	r := Decide(modeCfg(config.ModeAuto), makeResult(5, nil, nil))
	if r.Action != ActionPass {
		t.Errorf("action = %d, want ActionPass", r.Action)
	}
}

func TestDecide_AutoMode_MildWaste(t *testing.T) {
	// Below escalation threshold, no flags — pass in auto.
	r := Decide(modeCfg(config.ModeAuto), makeResult(30, sampleIssues, nil))
	if r.Action != ActionPass {
		t.Errorf("action = %d, want ActionPass (below escalation threshold)", r.Action)
	}
}

func TestDecide_AutoMode_HighWaste(t *testing.T) {
	r := Decide(modeCfg(config.ModeAuto), makeResult(120, sampleIssues, nil))
	if r.Action != ActionBlock {
		t.Errorf("action = %d, want ActionBlock", r.Action)
	}
	if r.Block == nil {
		t.Fatal("block is nil")
	}
	if r.Block.Decision != "block" {
		t.Errorf("decision = %q, want \"block\"", r.Block.Decision)
	}
	if r.Block.Reason == "" {
		t.Error("reason is empty")
	}
}

func TestDecide_AutoMode_FlagsBlock(t *testing.T) {
	// Low waste but flags present — should block in auto with EscalateOnFlags.
	r := Decide(modeCfg(config.ModeAuto), makeResult(5, nil, sampleFlags))
	if r.Action != ActionBlock {
		t.Errorf("action = %d, want ActionBlock due to flags", r.Action)
	}
}

func TestDecide_AutoMode_FlagsDisabled(t *testing.T) {
	cfg := modeCfg(config.ModeAuto)
	cfg.EscalateOnIndirectFlags = false
	// Low waste + flags, but EscalateOnFlags is off — should pass.
	r := Decide(cfg, makeResult(5, nil, sampleFlags))
	if r.Action != ActionPass {
		t.Errorf("action = %d, want ActionPass with flags disabled", r.Action)
	}
}

// --- Format tests ---

func TestFormatFeedback_CompactHeader(t *testing.T) {
	result := &analyzer.AnalysisResult{
		WastedTokens: 5,
		Issues:       sampleIssues,
		Flags:        sampleFlags,
	}
	out := formatFeedback(result)

	if !strings.HasPrefix(out, "⚡ PromptLinter") {
		t.Error("missing header prefix")
	}
	if strings.Contains(out, "tokens saved") {
		t.Error("should not show token count when <= threshold")
	}
	if !strings.Contains(out, "• Drop \"please\" (filler)") {
		t.Errorf("missing issue bullet, got:\n%s", out)
	}
	if !strings.Contains(out, "• No file path specified (vague_reference)") {
		t.Errorf("missing flag bullet, got:\n%s", out)
	}
}

func TestFormatFeedback_ShowsTokensWhenSignificant(t *testing.T) {
	result := &analyzer.AnalysisResult{
		WastedTokens: 25,
		Issues:       sampleIssues,
	}
	out := formatFeedback(result)

	if !strings.Contains(out, "~25 tokens saved") {
		t.Errorf("should show tokens when > threshold, got:\n%s", out)
	}
}

func TestFormatFeedback_LLMSuggestion(t *testing.T) {
	result := &analyzer.AnalysisResult{
		WastedTokens:  5,
		Issues:        sampleIssues,
		LLMSuggestion: "Fix the auth bug in login.go",
		TokensSaved:   15,
	}
	out := formatFeedback(result)

	if !strings.Contains(out, "💡 Try: Fix the auth bug in login.go") {
		t.Errorf("missing LLM suggestion, got:\n%s", out)
	}
	if !strings.Contains(out, "~15 tokens saved") {
		t.Errorf("should show LLM token count, got:\n%s", out)
	}
}

func TestFormatFeedback_LLMError(t *testing.T) {
	result := &analyzer.AnalysisResult{
		WastedTokens: 5,
		Issues:       sampleIssues,
		LLMError:     "timeout",
	}
	out := formatFeedback(result)

	if !strings.Contains(out, "⚠ AI analysis failed: timeout") {
		t.Errorf("missing LLM error, got:\n%s", out)
	}
}
