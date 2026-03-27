package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"os/exec"
	"time"

	"promptlinter/internal/analyzer/rules"
	"promptlinter/internal/tokenizer"
)

// ClaudeAnalyzer implements Analyzer using the `claude -p` CLI.
// It reuses the user's existing Claude Code auth — no separate API key needed.
type ClaudeAnalyzer struct {
	model   string
	counter *tokenizer.Counter
}

// NewClaudeAnalyzer creates a ClaudeAnalyzer for the given model alias (e.g. "haiku").
func NewClaudeAnalyzer(model string, counter *tokenizer.Counter) *ClaudeAnalyzer {
	return &ClaudeAnalyzer{model: model, counter: counter}
}

// claudeOutput matches the JSON structure returned by `claude -p --output-format json`.
type claudeOutput struct {
	Result    string  `json:"result"`
	TotalCost float64 `json:"total_cost_usd"`
	IsError   bool    `json:"is_error"`
}

// Analyze runs the prompt through the Claude CLI for deeper analysis beyond Layer 1 rules.
// Returns an error if the claude binary is not found or times out — callers should treat
// errors as non-fatal and fall back to Layer 1 output only.
func (c *ClaudeAnalyzer) Analyze(prompt string, issues []rules.Issue, flags []rules.Flag) (*Result, error) {
	systemPrompt := buildSystemPrompt(issues, flags)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"-p",
		"--model", c.model,
		"--no-session-persistence",
		"--output-format", "json",
		"--max-budget-usd", "0.05",
		"--system-prompt", systemPrompt,
		fmt.Sprintf("The prompt to analyze:\n<prompt>\n%s\n</prompt>", prompt),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("llm analysis timed out after 8s")
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

	suggestion := strings.TrimSpace(out.Result)
	tokensSaved := c.counter.Count(prompt) - c.counter.Count(suggestion)

	return &Result{
		Suggestion:  suggestion,
		TokensSaved: tokensSaved,
		CostUSD:     out.TotalCost,
	}, nil
}

// buildSystemPrompt constructs the system prompt incorporating Layer 1 findings
// so the LLM can focus on deeper analysis rather than repeating what rules already caught.
func buildSystemPrompt(issues []rules.Issue, flags []rules.Flag) string {
	var sb strings.Builder

	sb.WriteString("You are a prompt quality analyzer. A rules-based system (Layer 1) has already scanned the following prompt and found these issues:\n\n")

	if len(issues) > 0 {
		sb.WriteString("Direct waste detected:\n")
		for _, issue := range issues {
			fmt.Fprintf(&sb, "  - %s: %q → %s\n", issue.Type, issue.Match, issue.Suggestion)
		}
	}

	if len(flags) > 0 {
		sb.WriteString("Structural warnings:\n")
		for _, flag := range flags {
			fmt.Fprintf(&sb, "  - %s\n", flag.Description)
		}
	}

	sb.WriteString("\nYour task: analyze the prompt for deeper issues that regex cannot catch — ")
	sb.WriteString("vagueness, ambiguity, missing context, or over-specification. ")
	sb.WriteString("Then provide a concise, improved version of the prompt. ")
	sb.WriteString("Format your response as:\n")
	sb.WriteString("ANALYSIS: <one or two sentences on what is wrong>\n")
	sb.WriteString("SUGGESTION: <the improved prompt, nothing else>")

	return sb.String()
}
