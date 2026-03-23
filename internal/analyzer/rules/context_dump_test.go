package rules

import (
	"fmt"
	"strings"
	"testing"

	"promptlinter/internal/tokenizer"
)

func newTestContextDumpDetector(t *testing.T) *ContextDumpDetector {
	t.Helper()
	counter, err := tokenizer.New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}
	return NewContextDumpDetector(counter)
}

func TestContextDumpDetector_Name(t *testing.T) {
	d := newTestContextDumpDetector(t)
	if d.Name() != "context_dumping" {
		t.Errorf("expected name 'context_dumping', got %q", d.Name())
	}
}

func TestContextDumpDetector_CleanPrompt(t *testing.T) {
	d := newTestContextDumpDetector(t)
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

// --- Large code block tests ---

func TestContextDumpDetector_LargeCodeBlock(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Build a fenced code block with 60 lines (above 50-line threshold).
	var lines []string
	lines = append(lines, "```go")
	for i := 0; i < 60; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d() {}", i))
	}
	lines = append(lines, "```")
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Error("expected large code block to be flagged")
	}
	if result.WastedTokens == 0 {
		t.Error("expected wasted tokens > 0")
	}
}

func TestContextDumpDetector_SmallCodeBlock(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// 10 lines — well below threshold.
	var lines []string
	lines = append(lines, "```go")
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d() {}", i))
	}
	lines = append(lines, "```")
	prompt := "Fix this code:\n" + strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not flag for large code block.
	for _, issue := range result.Issues {
		if issue.Suggestion == "Trim to relevant lines or let the tool read the file directly" {
			t.Errorf("small code block should not be flagged, got: %v", issue)
		}
	}
}

// --- Stack trace tests ---

func TestContextDumpDetector_StackTrace_JS(t *testing.T) {
	d := newTestContextDumpDetector(t)

	var lines []string
	lines = append(lines, "Error: connection refused")
	for i := 0; i < 12; i++ {
		lines = append(lines, fmt.Sprintf("  at handler%d (src/server.js:%d)", i, 10+i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Error("expected JS stack trace to be flagged")
	}
}

func TestContextDumpDetector_StackTrace_Python(t *testing.T) {
	d := newTestContextDumpDetector(t)

	var lines []string
	lines = append(lines, "Traceback (most recent call last)")
	for i := 0; i < 12; i++ {
		lines = append(lines, fmt.Sprintf("  File \"app.py\", line %d", 10+i))
		lines = append(lines, fmt.Sprintf("    do_thing_%d()", i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Error("expected Python stack trace to be flagged")
	}
}

func TestContextDumpDetector_StackTrace_Go(t *testing.T) {
	d := newTestContextDumpDetector(t)

	var lines []string
	lines = append(lines, "goroutine 1 [running]:")
	for i := 0; i < 12; i++ {
		lines = append(lines, fmt.Sprintf("  main.handler%d()", i))
		lines = append(lines, fmt.Sprintf("  /app/main.go:%d", 10+i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Error("expected Go stack trace to be flagged")
	}
}

func TestContextDumpDetector_ShortStackTrace(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Only 3 frames — below threshold.
	prompt := "Error: failed\n  at handler (src/app.js:10)\n  at main (src/index.js:5)\n  at run (src/boot.js:1)"

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range result.Issues {
		if issue.Suggestion == "Include only the error message and first few frames" {
			t.Errorf("short stack trace should not be flagged, got: %v", issue)
		}
	}
}

// --- Log dump tests ---

func TestContextDumpDetector_LogDump_Timestamps(t *testing.T) {
	d := newTestContextDumpDetector(t)

	var lines []string
	for i := 0; i < 25; i++ {
		lines = append(lines, fmt.Sprintf("2024-01-15 10:%02d:00 INFO request handled", i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Error("expected log dump to be flagged")
	}
}

func TestContextDumpDetector_LogDump_LevelPrefixed(t *testing.T) {
	d := newTestContextDumpDetector(t)

	var lines []string
	for i := 0; i < 25; i++ {
		lines = append(lines, fmt.Sprintf("[ERROR] failed to process request %d", i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Error("expected level-prefixed log dump to be flagged")
	}
}

func TestContextDumpDetector_ShortLogOutput(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Only 5 lines — below threshold.
	var lines []string
	for i := 0; i < 5; i++ {
		lines = append(lines, fmt.Sprintf("2024-01-15 10:%02d:00 INFO ok", i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range result.Issues {
		if issue.Suggestion == "Include only the relevant log lines around the error" {
			t.Errorf("short log output should not be flagged, got: %v", issue)
		}
	}
}

// --- Unfenced code tests ---

func TestContextDumpDetector_UnfencedCode(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// 35 consecutive lines of Go code without backticks.
	var lines []string
	lines = append(lines, "fix this code:")
	for i := 0; i < 35; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d() {", i))
		lines = append(lines, "  return nil")
		lines = append(lines, "}")
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, "consecutive lines of unfenced code") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected unfenced code to be flagged")
	}
	if result.WastedTokens == 0 {
		t.Error("expected wasted tokens > 0")
	}
}

func TestContextDumpDetector_UnfencedCode_BelowThreshold(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Only 10 consecutive code lines — below threshold.
	var lines []string
	lines = append(lines, "look at this:")
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d() {}", i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, "unfenced code") {
			t.Errorf("short unfenced code should not be flagged, got: %v", issue)
		}
	}
}

func TestContextDumpDetector_UnfencedCode_SkipsWhenFenced(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Code is fenced — unfenced detector should skip entirely.
	var lines []string
	lines = append(lines, "```go")
	for i := 0; i < 35; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d() {}", i))
	}
	lines = append(lines, "```")
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, "unfenced code") {
			t.Errorf("fenced code should not trigger unfenced detector, got: %v", issue)
		}
	}
}

func TestContextDumpDetector_UnfencedCode_ScatteredKeywords(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Prose with scattered code keywords — should NOT be flagged
	// because they aren't consecutive.
	var lines []string
	for i := 0; i < 50; i++ {
		if i%3 == 0 {
			lines = append(lines, "import something here")
		} else {
			lines = append(lines, "This is a regular prose line about the architecture.")
		}
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, "unfenced code") {
			t.Errorf("scattered keywords in prose should not trigger unfenced detector, got: %v", issue)
		}
	}
}

// --- Paste-heavy tests ---

func TestContextDumpDetector_PasteHeavy(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Prompt that is >70% code by tokens.
	var codeLines []string
	for i := 0; i < 20; i++ {
		codeLines = append(codeLines, fmt.Sprintf("func handler%d(w http.ResponseWriter, r *http.Request) { return }", i))
	}
	prompt := "fix:\n```go\n" + strings.Join(codeLines, "\n") + "\n```"

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, ">70%") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected paste-heavy prompt to be flagged")
	}
}

func TestContextDumpDetector_PasteHeavy_UnfencedCode(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Prompt that is >70% unfenced code — should trigger paste-heavy via unfenced tokens.
	var lines []string
	lines = append(lines, "fix:")
	for i := 0; i < 35; i++ {
		lines = append(lines, fmt.Sprintf("func handler%d(w http.ResponseWriter, r *http.Request) { return }", i))
	}
	prompt := strings.Join(lines, "\n")

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, ">70%") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected paste-heavy to trigger for unfenced code")
	}
}

func TestContextDumpDetector_BalancedPrompt(t *testing.T) {
	d := newTestContextDumpDetector(t)

	// Short code block with plenty of surrounding context — should not trigger paste-heavy.
	prompt := `I'm seeing an error in the auth module when JWT tokens expire.
The relevant function is:
` + "```go" + `
func validateToken(token string) error {
    claims, err := jwt.Parse(token)
    if err != nil {
        return err
    }
    return claims.Valid()
}
` + "```" + `
The error is "token expired" but the token was just issued 5 minutes ago.
I think the clock skew setting might be wrong. Can you check the JWT config?`

	result, err := d.Detect(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range result.Issues {
		if strings.Contains(issue.Match, ">70%") {
			t.Errorf("balanced prompt should not trigger paste-heavy, got: %v", issue)
		}
	}
}
