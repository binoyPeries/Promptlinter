package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Mode represents the operating mode for the linter.
type Mode string

const (
	ModeSuggest Mode = "suggest"
	ModeSilent  Mode = "silent"
	ModeAuto    Mode = "auto"
	ModeOff     Mode = "off"
)

// ValidModes returns all valid operating modes.
func ValidModes() []Mode {
	return []Mode{ModeAuto, ModeOff, ModeSilent, ModeSuggest}
}

// Config holds all user-configurable settings.
type Config struct {
	Mode                    Mode     `json:"mode"`
	TipThreshold            int      `json:"tip_threshold"`
	EscalationThreshold     int      `json:"escalation_threshold"`
	EscalateOnIndirectFlags bool     `json:"escalate_on_indirect_flags"`
	LLMEnabled              bool     `json:"llm_enabled"`
	LLMModel                string   `json:"llm_model"`
	DailyBudget             float64  `json:"haiku_daily_budget"`
	IgnoredPatterns         []string `json:"ignored_patterns"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Mode:                    ModeSuggest,
		TipThreshold:            20,
		EscalationThreshold:     100,
		EscalateOnIndirectFlags: true,
		LLMEnabled:              false,
		LLMModel:                "haiku",
		DailyBudget:             0.10,
		IgnoredPatterns:         []string{`^/`, `^y$|^n$|^yes$|^no$`},
	}
}

// ConfigDir returns the path to ~/.promptlinter, creating it if needed.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	dir := filepath.Join(home, ".promptlinter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return dir, nil
}

// Load reads config from ~/.promptlinter/config.json.
// If the file does not exist, it returns DefaultConfig with no error.
// If the file exists but is malformed, it returns an error.
func Load() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return DefaultConfig(), nil
	}

	cfgPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := DefaultConfig()
			if writeErr := writeDefaults(cfgPath, cfg); writeErr != nil {
				return cfg, nil
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return cfg, nil
}

// SetMode loads the current config, updates the mode, and saves it back.
func SetMode(m Mode) error {
	dir, err := ConfigDir()
	if err != nil {
		return fmt.Errorf("failed to resolve config directory: %w", err)
	}
	cfgPath := filepath.Join(dir, "config.json")
	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg.Mode = m
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// writeDefaults writes the given config to path as indented JSON.
func writeDefaults(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadFrom reads config from the specified path.
// If the file does not exist, it returns DefaultConfig with no error.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config from %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from %s: %w", path, err)
	}
	return cfg, nil
}
