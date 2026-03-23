package rules

import (
	"fmt"
	"regexp"
	"strings"

	"promptlinter/internal/tokenizer"
)

// Thresholds for context dumping detection.
const (
	// codeBlockLineThreshold is the minimum lines in a single code block to flag.
	codeBlockLineThreshold = 50

	// stackTraceLineThreshold is the minimum lines of stack trace to flag.
	stackTraceLineThreshold = 10

	// logDumpLineThreshold is the minimum lines of log output to flag.
	logDumpLineThreshold = 20

	// pasteRatioThreshold flags when pasted content exceeds this fraction of total tokens.
	pasteRatioThreshold = 0.70

	// unfencedCodeLineThreshold is the minimum consecutive code-like lines to flag.
	unfencedCodeLineThreshold = 30
)

var (
	// codeFenceRe matches fenced code blocks (``` with optional language tag).
	codeFenceRe = regexp.MustCompile("(?m)^```[a-zA-Z]*\\s*$")

	// stackTracePatterns match common stack trace lines across languages.
	stackTracePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s+at .+\(.+:\d+`),                    // JS/Java/TS: "  at foo (file.js:10)"
		regexp.MustCompile(`(?m)^goroutine \d+`),                       // Go: "goroutine 1 [running]:"
		regexp.MustCompile(`(?m)^\s+.+\.go:\d+`),                       // Go stack frame
		regexp.MustCompile(`(?m)^Traceback \(most recent call last\)`), // Python
		regexp.MustCompile(`(?m)^\s+File ".+", line \d+`),              // Python stack frame
		regexp.MustCompile(`(?m)^\s+from .+:\d+:\d+`),                  // C++/Rust: "  from src/main.rs:12:5"
		regexp.MustCompile(`(?m)^\s+at .+\[0x[0-9a-f]+\]`),             // C/C++ native: "  at 0x7fff [0x00123]"
		regexp.MustCompile(`(?m)^\s+\d+: .+::\w+`),                     // Rust backtrace: "  4: mymod::func"
		regexp.MustCompile(`(?m)^\s+.+\.rb:\d+:in\s+`),                 // Ruby: "  app.rb:10:in `method'"
		regexp.MustCompile(`(?m)^\s+.+\.cs:\d+`),                       // C#: "  at Foo.Bar() in file.cs:10"
		regexp.MustCompile(`(?m)^\s+.+\.java:\d+\)`),                   // Java: "  at com.Foo.bar(Foo.java:10)"
		regexp.MustCompile(`(?m)^\s+.+\.swift:\d+`),                    // Swift
		regexp.MustCompile(`(?m)^\s+.+\.kt:\d+`),                       // Kotlin
		regexp.MustCompile(`(?m)^\s+.+\.php:\d+`),                      // PHP
		regexp.MustCompile(`(?m)^\s+.+\.scala:\d+\)`),                  // Scala
		regexp.MustCompile(`(?m)^#\d+\s+.+\.dart:\d+`),                 // Dart/Flutter: "#0 Class.method (file.dart:10)"
		regexp.MustCompile(`(?m)^\s+.+\.ex:\d+:`),                      // Elixir
		regexp.MustCompile(`(?m)^\s+.+\.lua:\d+:`),                     // Lua
	}

	// codeLineRe matches lines that look like source code.
	codeLineRe = regexp.MustCompile(
		`(?m)^(` +
			`\s*(func |def |class |import |from |package |module |` +
			`const |let |var |if |for |while |return |switch |case |` +
			`#include |using |namespace )` +
			`|.*[{};]\s*$` +
			`|^\s*(//|#|/\*|\*|--|%%))`,
	)

	// logLinePatterns match common log line formats.
	logLinePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\d{2,4}[-/]\d{2}[-/]\d{2}[T ]?\d{2}:\d{2}`),      // 2024-01-15 10:30 or 2024/01/15T10:30
		regexp.MustCompile(`(?m)^\[?\d{2}:\d{2}:\d{2}[\].]`),                      // [10:30:00] or 10:30:00.123
		regexp.MustCompile(`(?m)^\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}`),            // syslog: "Jan 15 10:30:00"
		regexp.MustCompile(`(?m)^(DEBUG|INFO|WARN|ERROR|FATAL|TRACE)\s*[\[|\|:]`), // Level-prefixed: "ERROR: ..." or "INFO | ..."
		regexp.MustCompile(`(?m)^\[?(DEBUG|INFO|WARN|ERROR|FATAL|TRACE)\]?\s`),    // [ERROR] message
	}
)

// ContextDumpDetector detects prompts with excessive pasted content such as
// large code blocks, full stack traces, and log dumps.
type ContextDumpDetector struct {
	counter *tokenizer.Counter
}

// NewContextDumpDetector creates a ContextDumpDetector.
func NewContextDumpDetector(counter *tokenizer.Counter) *ContextDumpDetector {
	return &ContextDumpDetector{counter: counter}
}

var _ Detector = (*ContextDumpDetector)(nil)

func (d *ContextDumpDetector) Name() string {
	return "context_dumping"
}

func (d *ContextDumpDetector) Detect(prompt string) (*Result, error) {
	result := &Result{DetectorName: d.Name()}

	d.detectStackTraces(prompt, result)
	d.detectLogDumps(prompt, result)
	codeBlockTokens := d.detectLargeCodeBlocks(prompt, result)
	unfencedTokens := d.detectUnfencedCode(prompt, result)
	d.detectPasteHeavy(prompt, codeBlockTokens+unfencedTokens, result)

	return result, nil
}

// detectLargeCodeBlocks flags code fences with more than codeBlockLineThreshold lines.
// Returns total tokens in all code blocks (for paste-ratio calculation).
func (d *ContextDumpDetector) detectLargeCodeBlocks(prompt string, result *Result) int {
	locs := codeFenceRe.FindAllStringIndex(prompt, -1)
	totalCodeTokens := 0

	// Pair up fences: [0]=open, [1]=close, [2]=open, [3]=close, ...
	for i := 0; i+1 < len(locs); i += 2 {
		openEnd := locs[i][1]
		closeStart := locs[i+1][0]
		block := prompt[openEnd:closeStart]
		lines := strings.Count(block, "\n")
		tokens := d.counter.Count(block)
		totalCodeTokens += tokens

		if lines >= codeBlockLineThreshold {
			result.Issues = append(result.Issues, Issue{
				Type:       d.Name(),
				Match:      truncate(strings.TrimSpace(block), 80),
				Suggestion: "Trim to relevant lines or let the tool read the file directly",
				Tokens:     tokens,
			})
			result.WastedTokens += tokens
		}
	}

	return totalCodeTokens
}

// detectUnfencedCode flags large blocks of code pasted without markdown fences.
// Returns total tokens in the unfenced code block (for paste-ratio calculation).
func (d *ContextDumpDetector) detectUnfencedCode(prompt string, result *Result) int {
	if codeFenceRe.MatchString(prompt) {
		return 0
	}

	lines := strings.Split(prompt, "\n")

	maxRun, currentRun := 0, 0
	runStart := 0
	bestStart := 0

	for i, line := range lines {
		if !codeLineRe.MatchString(line) {
			currentRun = 0
			continue
		}
		if currentRun == 0 {
			runStart = i
		}
		currentRun++
		if currentRun > maxRun {
			maxRun = currentRun
			bestStart = runStart
		}
	}

	if maxRun < unfencedCodeLineThreshold {
		return 0
	}

	codeBlock := strings.Join(lines[bestStart:bestStart+maxRun], "\n")
	tokens := d.counter.Count(codeBlock)
	result.Issues = append(result.Issues, Issue{
		Type:       d.Name(),
		Match:      truncate(fmt.Sprintf("~%d consecutive lines of unfenced code", maxRun), 80),
		Suggestion: "Trim to relevant lines or let the tool read the file directly",
		Tokens:     tokens,
	})
	result.WastedTokens += tokens
	return tokens
}

// detectStackTraces flags large stack traces.
func (d *ContextDumpDetector) detectStackTraces(prompt string, result *Result) {
	for _, re := range stackTracePatterns {
		matches := re.FindAllString(prompt, -1)
		if len(matches) >= stackTraceLineThreshold {
			// Estimate tokens for all matched stack trace lines.
			combined := strings.Join(matches, "\n")
			tokens := d.counter.Count(combined)
			result.Issues = append(result.Issues, Issue{
				Type:       d.Name(),
				Match:      truncate(matches[0], 80),
				Suggestion: "Include only the error message and first few frames",
				Tokens:     tokens,
			})
			result.WastedTokens += tokens
			return // One stack trace issue is enough.
		}
	}
}

// detectLogDumps flags large log pastes.
func (d *ContextDumpDetector) detectLogDumps(prompt string, result *Result) {
	lines := strings.Split(prompt, "\n")
	var logLines []string
	for _, line := range lines {
		for _, re := range logLinePatterns {
			if re.MatchString(line) {
				logLines = append(logLines, line)
				break
			}
		}
	}
	if len(logLines) >= logDumpLineThreshold {
		combined := strings.Join(logLines, "\n")
		tokens := d.counter.Count(combined)
		result.Issues = append(result.Issues, Issue{
			Type:       d.Name(),
			Match:      truncate(logLines[0], 80),
			Suggestion: "Include only the relevant log lines around the error",
			Tokens:     tokens,
		})
		result.WastedTokens += tokens
	}
}

// detectPasteHeavy flags prompts where pasted content dominates.
func (d *ContextDumpDetector) detectPasteHeavy(prompt string, codeBlockTokens int, result *Result) {
	if codeBlockTokens == 0 {
		return
	}
	totalTokens := d.counter.Count(prompt)
	if totalTokens == 0 {
		return
	}
	ratio := float64(codeBlockTokens) / float64(totalTokens)
	if ratio >= pasteRatioThreshold {
		wastedTokens := codeBlockTokens - (totalTokens - codeBlockTokens) // rough excess
		if wastedTokens <= 0 {
			return
		}
		result.Issues = append(result.Issues, Issue{
			Type:       d.Name(),
			Match:      "Code blocks are >70% of prompt",
			Suggestion: "Add more context about what to look for, or trim the pasted code",
			Tokens:     wastedTokens,
		})
		result.WastedTokens += wastedTokens
	}
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
