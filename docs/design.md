## Problem Statement

LLM users waste **15-30% of their tokens** on inefficient prompts. Common patterns include:

| Pattern | Example | Cost |
|---------|---------|------|
| **Filler language** | "Can you please", "I was wondering if" | 5-15 tokens per prompt, compounding over hundreds of prompts |
| **Vague references** | "fix the bug", "look at the file" | Claude spends 3-7 extra turns searching, costing thousands of tokens |
| **Redundant phrasing** | Saying the same thing three different ways | 30-50% of prompt tokens wasted |
| **Context re-explanation** | Repeating project info already in CLAUDE.md | 100-200 tokens per session wasted |
| **Unbatched requests** | Sending 3 related edits as 3 separate prompts | 3x round-trip overhead |

Existing tools in this space either **expand** prompts (making them longer/more detailed) or focus on generic text compression. **Nobody is helping coding-assistant users write shorter, more effective prompts and tracking their improvement over time.**

## Solution

### Overview

PromptLinter is a **CLI tool** that integrates with Claude Code via hooks. It operates as a two-layer hybrid system:

1. **Layer 1 — Rules Engine**: Fast, local, zero-cost regex/heuristic analysis (~5ms)
2. **Layer 2 — Small Model (Haiku)**: Smart, context-aware rewriting when waste is significant (~0.01¢ per call)

It runs on **every prompt submission**, analyzes the prompt, and depending on the user's chosen mode:
- **Suggests** a rewrite (default)
- **Silently logs** for later reporting
- **Auto-blocks** wasteful prompts and shows the rewrite

All data is stored locally in SQLite. Users can generate reports showing their prompt efficiency trends over time.

### Deep Dive

### Layer 1 — Rules Engine (Direct Waste Detection)

Layer 1 runs on every prompt locally with zero cost and ~5ms latency. It detects **direct waste** — tokens physically present in the prompt that add no information — and reports the exact count of wasted tokens.

It operates on the prompt text alone, requiring no external context.

**Detectors:**

| Detector | What it catches | How it measures |
|----------|----------------|-----------------|
| **Filler** | Politeness, hedging — "Can you please", "I was wondering if", "I think maybe" | Count tokens in matched phrases |
| **Redundancy** | Same term or phrase repeated within one prompt | Count tokens in duplicate occurrences |
| **Meta-Commentary** | Preamble — "Let me explain what I need", "Here's what I want" | Count tokens in matched preamble patterns |
| **Context Dumping** | Full stack traces, entire file pastes when a few lines would suffice | Count tokens in unnecessary pasted content |

Layer 1 also **flags indirect risk patterns** it can detect but cannot estimate the cost of — vague references ("the file", "fix the bug") and over-specification (sequential step markers for basic operations). These flags are passed to Layer 2.

**Thresholds:**
- `< 20 tokens wasted` → pass through, log only
- `20-100 tokens wasted` → inline tip (show what to remove)
- `> 100 tokens wasted` OR indirect risk flags present → escalate to Layer 2

### Layer 2 — Small Model (Indirect Waste Estimation)

Layer 2 handles what Layer 1 cannot: **context-dependent analysis** that requires understanding the project, session history, and semantic meaning. It is invoked only when Layer 1 finds significant direct waste or flags indirect risks.

| Analysis | Why Layer 1 can't do it |
|----------|------------------------|
| **Context Overlap** — prompt re-explains what's in CLAUDE.md | Requires semantic comparison, not just keyword matching |
| **Indirect waste estimation** — how many extra turns a vague prompt will cause | Depends on codebase size, session state, task complexity |
| **Specificity assessment** — whether missing details will actually cause extra work | "fix the bug in auth" is fine in a 5-file project, costly in a monorepo |
| **Batching detection** — whether sequential prompts could have been combined | Requires understanding intent across turns |
| **Rewriting** — producing a concise rewrite that preserves all intent | Requires understanding meaning, not just removing patterns |

Layer 2 returns both a rewritten prompt and an **indirect waste estimate** in tokens. These estimates calibrate over time using actual turn and tool-call data collected across sessions.

When Layer 2 is disabled, Layer 1 still flags indirect risks as qualitative warnings without a token estimate.

> **Example:** A small, fast model like Claude Haiku works well here — ~$0.0001 per call, fast enough for real-time use, smart enough for semantic analysis and rewriting.


### User Experience

**Three modes** control how PromptLinter responds to each prompt:

| Mode | Behavior |
|------|----------|
| **Suggest** (default) | Shows waste feedback inline. The prompt still goes through — the user learns for next time. |
| **Silent** | Logs everything to the database. Zero output, zero interruption. User reviews waste later via reports. |
| **Auto** | Blocks prompts above a waste threshold. Shows a rewrite and lets the user accept it (`y`), edit it before submitting (`e`), or dismiss and submit the original (`n`). |

**Real-time feedback examples:**

- **Low waste (<20 tokens):** Prompt passes through silently. `✓ Prompt looks efficient — no waste detected`
- **Moderate waste (20-100 tokens):** One-line inline tip. `~4 tokens wasted. Quick tip: Drop "Can you please" — saves 4 tokens`
- **High waste (>100 tokens or indirect risks):** Full suggestion with rewrite, showing direct waste, indirect waste estimate, and a concise rewrite.
- **Auto mode block:** Prompt is blocked with a breakdown of why and a suggested rewrite. User chooses: accept rewrite (`y`), edit it (`e`), or dismiss and send original (`n`).

**Reports** are generated on demand from stored data:

- Daily, weekly, or monthly periods
- Show direct vs indirect waste breakdown in tokens
- Rank issues by tokens wasted (not by count — a single vague prompt costing ~3k tokens matters more than 10 filler phrases costing 40 tokens)
- Compare against previous period to show improvement trends
- Highlight best and worst prompts from the user's own history

**What's configurable:**

| Setting | What it controls | Default |
|---------|-----------------|---------|
| **Mode** | suggest / silent / auto | suggest |
| **Tip threshold** | Direct waste above which an inline tip is shown | 20 tokens |
| **Escalation threshold** | Direct waste above which Layer 2 is invoked | 100 tokens |
| **Escalate on indirect flags** | Whether indirect risk flags also trigger Layer 2 | true |
| **Layer 2 enabled** | Whether the small model is used at all (rules-only mode if off) | true |
| **Daily budget** | Max spend on Layer 2 calls per day | $0.10 |
| **Ignored patterns** | Prompts to skip (e.g., slash commands, "yes"/"no" confirmations) | `^/`, `^y$\|^n$` |
| **Privacy** | Whether to store full prompt text, rewrites, or redact file paths | store all |