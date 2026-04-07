package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"promptlinter/internal/config"
	"promptlinter/internal/decision"
)

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write failed")
}

func makeInput(prompt string) string {
	b, _ := json.Marshal(HookInput{
		SessionID:     "test-session",
		Prompt:        prompt,
		Cwd:           "/tmp",
		HookEventName: "UserPromptSubmit",
	})
	return string(b)
}

func TestHandleAnalyze_CleanPrompt(t *testing.T) {
	cfg := config.DefaultConfig()
	stdin := strings.NewReader(makeInput("Fix the nil pointer in src/auth.go line 42"))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestHandleAnalyze_FillerPrompt_Tip(t *testing.T) {
	cfg := config.DefaultConfig()
	stdin := strings.NewReader(makeInput("Can you please look at the code and fix it, thanks"))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty for suggest mode", stderr.String())
	}

	var tip struct {
		SystemMessage  string `json:"systemMessage"`
		SuppressOutput bool   `json:"suppressOutput"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &tip); err != nil {
		t.Fatalf("failed to parse tip JSON: %v (stdout=%q)", err, stdout.String())
	}
	if !strings.Contains(tip.SystemMessage, "PromptLinter") {
		t.Errorf("systemMessage = %q, want PromptLinter feedback", tip.SystemMessage)
	}
	if !tip.SuppressOutput {
		t.Error("suppressOutput = false, want true")
	}
}

func TestHandleAnalyze_AutoMode_Block(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Mode = config.ModeAuto
	cfg.EscalateOnIndirectFlags = true

	// "the file" triggers vague_reference flag → block in auto mode
	stdin := strings.NewReader(makeInput("Fix the bug in the file"))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() == 0 {
		t.Fatal("stdout empty, want block JSON")
	}

	var block decision.BlockOutput
	if err := json.Unmarshal(stdout.Bytes(), &block); err != nil {
		t.Fatalf("failed to parse block JSON: %v", err)
	}
	if block.Decision != "block" {
		t.Errorf("decision = %q, want \"block\"", block.Decision)
	}
	if block.Reason == "" {
		t.Error("reason is empty")
	}
}

func TestHandleAnalyze_OffMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Mode = config.ModeOff
	stdin := strings.NewReader(makeInput("Can you please look at the code and fix it, thanks"))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty for off mode", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty for off mode", stderr.String())
	}
}

func TestHandleAnalyze_SilentMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Mode = config.ModeSilent
	stdin := strings.NewReader(makeInput("Can you please look at the code and fix it, thanks"))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty for silent mode", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty for silent mode", stderr.String())
	}
}

func TestHandleAnalyze_IgnoredPattern(t *testing.T) {
	cfg := config.DefaultConfig()
	// Default ignored patterns include `^y$|^n$|^yes$|^no$`
	stdin := strings.NewReader(makeInput("yes"))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty for ignored prompt", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty for ignored prompt", stderr.String())
	}
}

func TestHandleAnalyze_EmptyPrompt(t *testing.T) {
	cfg := config.DefaultConfig()
	stdin := strings.NewReader(makeInput(""))
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestHandleAnalyze_MalformedJSON(t *testing.T) {
	cfg := config.DefaultConfig()
	stdin := strings.NewReader("not json at all")
	var stdout, stderr bytes.Buffer

	HandleAnalyze(cfg, stdin, &stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty on error", stdout.String())
	}
	if !strings.Contains(stderr.String(), "failed to parse") {
		t.Errorf("stderr = %q, want parse error message", stderr.String())
	}
}

func TestHandleAnalyze_FallbackLogsWhenStderrWriteFails(t *testing.T) {
	cfg := config.DefaultConfig()
	stdin := strings.NewReader("not json at all")
	var stdout bytes.Buffer

	home := t.TempDir()
	t.Setenv("HOME", home)

	HandleAnalyze(cfg, stdin, &stdout, errWriter{})

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty on error", stdout.String())
	}

	logPath := filepath.Join(home, ".promptlinter", "promptlinter.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read fallback log: %v", err)
	}
	if !strings.Contains(string(data), "failed to parse hook input") {
		t.Fatalf("log = %q, want parse error entry", string(data))
	}
}
