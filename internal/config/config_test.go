package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Mode != ModeSuggest {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "suggest")
	}
	if cfg.TipThreshold != 20 {
		t.Errorf("TipThreshold = %d, want 20", cfg.TipThreshold)
	}
	if cfg.EscalationThreshold != 100 {
		t.Errorf("EscalationThreshold = %d, want 100", cfg.EscalationThreshold)
	}
	if !cfg.EscalateOnIndirectFlags {
		t.Error("EscalateOnIndirectFlags = false, want true")
	}
	if cfg.LLMEnabled {
		t.Error("LLMEnabled = true, want false")
	}
	if cfg.LLMModel != "haiku" {
		t.Errorf("LLMModel = %q, want %q", cfg.LLMModel, "haiku")
	}
	if cfg.DailyBudget != 0.10 {
		t.Errorf("DailyBudget = %f, want 0.10", cfg.DailyBudget)
	}
	if len(cfg.IgnoredPatterns) != 2 {
		t.Errorf("IgnoredPatterns len = %d, want 2", len(cfg.IgnoredPatterns))
	}
}

func TestLoadFrom_MissingFile(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != ModeSuggest {
		t.Errorf("Mode = %q, want default %q", cfg.Mode, "suggest")
	}
}

func TestLoadFrom_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{
		"mode": "auto",
		"tip_threshold": 30,
		"escalation_threshold": 80,
		"escalate_on_indirect_flags": false,
		"llm_enabled": true,
		"llm_model": "sonnet",
		"haiku_daily_budget": 0.05,
		"ignored_patterns": ["^/test"]
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Mode != ModeAuto {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "auto")
	}
	if cfg.TipThreshold != 30 {
		t.Errorf("TipThreshold = %d, want 30", cfg.TipThreshold)
	}
	if cfg.EscalationThreshold != 80 {
		t.Errorf("EscalationThreshold = %d, want 80", cfg.EscalationThreshold)
	}
	if cfg.EscalateOnIndirectFlags {
		t.Error("EscalateOnIndirectFlags = true, want false")
	}
	if !cfg.LLMEnabled {
		t.Error("LLMEnabled = false, want true")
	}
	if cfg.LLMModel != "sonnet" {
		t.Errorf("LLMModel = %q, want %q", cfg.LLMModel, "sonnet")
	}
	if cfg.DailyBudget != 0.05 {
		t.Errorf("DailyBudget = %f, want 0.05", cfg.DailyBudget)
	}
	if len(cfg.IgnoredPatterns) != 1 || cfg.IgnoredPatterns[0] != "^/test" {
		t.Errorf("IgnoredPatterns = %v, want [^/test]", cfg.IgnoredPatterns)
	}
}

func TestLoadFrom_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{bad json`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestLoadFrom_PartialJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"mode": "silent"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Mode != ModeSilent {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "silent")
	}
	// Unset fields should retain defaults.
	if cfg.TipThreshold != 20 {
		t.Errorf("TipThreshold = %d, want default 20", cfg.TipThreshold)
	}
	if cfg.EscalationThreshold != 100 {
		t.Errorf("EscalationThreshold = %d, want default 100", cfg.EscalationThreshold)
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("ConfigDir returned relative path: %s", dir)
	}
	if filepath.Base(dir) != ".promptlinter" {
		t.Errorf("ConfigDir base = %q, want %q", filepath.Base(dir), ".promptlinter")
	}
}
