package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"promptlinter/internal/analyzer/rules"
)

// ClaudeAnalyzer implements Analyzer using the `claude -p` CLI.
// It reuses the user's existing Claude Code auth — no separate API key needed.
type ClaudeAnalyzer struct {
	model string
}

// NewClaudeAnalyzer creates a ClaudeAnalyzer for the given model alias (e.g. "haiku").
func NewClaudeAnalyzer(model string) *ClaudeAnalyzer {
	return &ClaudeAnalyzer{model: model}
}

// claudeOutput matches the JSON structure returned by `claude -p --output-format json`.
type claudeOutput struct {
	TotalCost        float64   `json:"total_cost_usd"`
	IsError          bool      `json:"is_error"`
	StructuredOutput *llmResult `json:"structured_output"`
}

// llmResult is the structured output from Haiku via --json-schema.
type llmResult struct {
	Analysis    string `json:"analysis"`
	Suggestion  string `json:"suggestion"`
	TokensSaved int    `json:"tokens_saved"`
}

// jsonSchema is passed to --json-schema to enforce structured output from Haiku.
const jsonSchema = `{
  "type": "object",
  "properties": {
    "analysis":     {"type": "string"},
    "suggestion":   {"type": "string"},
    "tokens_saved": {"type": "integer"}
  },
  "required": ["analysis", "suggestion", "tokens_saved"]
}`

// Analyze runs the prompt through the Claude CLI for deeper analysis beyond Layer 1 rules.
// Returns an error if the claude binary is not found or times out — callers should treat
// errors as non-fatal and fall back to Layer 1 output only.
func (c *ClaudeAnalyzer) Analyze(prompt string, issues []rules.Issue, flags []rules.Flag) (*Result, error) {
	systemPrompt := buildSystemPrompt(issues, flags)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"-p",
		"--model", c.model,
		"--no-session-persistence",
		"--output-format", "json",
		"--json-schema", jsonSchema,
		"--max-budget-usd", "0.05",
		"--system-prompt", systemPrompt,
		fmt.Sprintf("The prompt to analyze:\n<prompt>\n%s\n</prompt>", prompt),
	)
	cmd.Dir = os.TempDir() // avoid loading any project CLAUDE.md or hooks
	cmd.Env = append(os.Environ(), "PLINT_INTERNAL=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("llm analysis timed out after 30s")
		}
		return nil, fmt.Errorf("claude -p failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	var out claudeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}
	if out.IsError {
		return nil, fmt.Errorf("claude returned an error result")
	}

	if out.StructuredOutput == nil {
		return nil, fmt.Errorf("claude returned no structured output")
	}

	return &Result{
		Suggestion:  strings.TrimSpace(out.StructuredOutput.Suggestion),
		TokensSaved: out.StructuredOutput.TokensSaved,
		CostUSD:     out.TotalCost,
	}, nil
}

// buildSystemPrompt constructs the system prompt incorporating Layer 1 findings
// so the LLM can focus on deeper analysis rather than repeating what rules already caught.
func buildSystemPrompt(issues []rules.Issue, flags []rules.Flag) string {
	var sb strings.Builder

	sb.WriteString("You are a prompt quality reviewer. A rules-based system has flagged the following prompt. Your job has three parts:\n\n")

	sb.WriteString("PART 1 — Validate Layer 1 findings (rules can produce false positives):\n")
	if len(issues) > 0 {
		sb.WriteString("Flagged waste:\n")
		for _, issue := range issues {
			fmt.Fprintf(&sb, "  - %s: %q → %s\n", issue.Type, issue.Match, issue.Suggestion)
		}
	} else {
		sb.WriteString("  (none)\n")
	}
	if len(flags) > 0 {
		sb.WriteString("Structural warnings:\n")
		for _, flag := range flags {
			fmt.Fprintf(&sb, "  - %s\n", flag.Description)
		}
	} else {
		sb.WriteString("  (none)\n")
	}

	sb.WriteString("\nPART 2 — Your own deeper analysis:\n")
	sb.WriteString("Identify issues the rules missed: vagueness, ambiguity, missing context, over-specification.\n")
	sb.WriteString("Discard any Layer 1 findings that are false positives.\n")

	sb.WriteString("\nPART 3 — Output fields:\n")
	sb.WriteString("- 'analysis': 1-2 sentences summarising the real issues (confirmed Layer 1 + your findings).\n")
	sb.WriteString("- 'tokens_saved': integer estimate of wasted tokens in the original prompt (filler, hedging, redundancy). If the prompt is actually fine, return 0.\n")
	sb.WriteString("- 'suggestion': the improved prompt the user can submit directly.\n")
	sb.WriteString("  If a full rewrite is possible: provide it as plain text, no markdown, no labels.\n")
	sb.WriteString("  If critical context is missing: provide a concise template or guiding question that shows the user what to fill in (e.g. 'Explain [topic] in the context of [your use case].').\n")
	sb.WriteString("  Keep it short. One option only.\n")

	return sb.String()
}
