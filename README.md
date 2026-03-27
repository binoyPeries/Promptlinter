# PromptLinter

PromptLinter is a lightweight CLI hook for Claude Code that catches prompt waste before it gets sent.

It helps you write clearer prompts with fewer tokens, fewer clarification turns, and less back-and-forth.

Think of it as a small guardrail for your prompts: fast feedback before the expensive part happens.

## Why it exists

- Dev prompts often include filler, vague references, and oversized context dumps.
- That burns tokens and adds extra turns you did not need.
- PromptLinter gives immediate feedback so you can tighten the prompt before it reaches Claude.

## What you get

- `⚡` Fast: runs on each prompt as a small Go binary.
- `🛠️` Practical: feedback is short, direct, and useful.
- `🔒` Local-first: rules engine works without external API calls.
- `🎛️` Flexible: `suggest`, `silent`, and `auto` modes.

## Install

Download the latest release for your platform from the [Releases page](https://github.com/binoyPeries/Promptlinter/releases).

Set the version you want to install (see [Releases](https://github.com/binoyPeries/Promptlinter/releases)):
```bash
VERSION=vX.Y.Z  # e.g. v0.1.0
```

**macOS (Apple Silicon):**
```bash
curl -Lo plint.tar.gz "https://github.com/binoyPeries/Promptlinter/releases/download/${VERSION}/plint-darwin-arm64.tar.gz"
tar xzf plint.tar.gz
sudo mkdir -p /usr/local/bin && sudo mv plint /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -Lo plint.tar.gz "https://github.com/binoyPeries/Promptlinter/releases/download/${VERSION}/plint-darwin-amd64.tar.gz"
tar xzf plint.tar.gz
sudo mkdir -p /usr/local/bin && sudo mv plint /usr/local/bin/
```

**Linux (x86_64):**
```bash
curl -Lo plint.tar.gz "https://github.com/binoyPeries/Promptlinter/releases/download/${VERSION}/plint-linux-amd64.tar.gz"
tar xzf plint.tar.gz
sudo mkdir -p /usr/local/bin && sudo mv plint /usr/local/bin/
```

**Windows (PowerShell):**
```powershell
$VERSION = "vX.Y.Z"  # e.g. v0.1.0
Invoke-WebRequest -OutFile plint.zip "https://github.com/binoyPeries/Promptlinter/releases/download/$VERSION/plint-windows-amd64.zip"
Expand-Archive plint.zip -DestinationPath .
New-Item -ItemType Directory -Force "$HOME\\bin" | Out-Null
Move-Item .\\plint.exe "$HOME\\bin\\plint.exe" -Force
```

Add `$HOME\bin` to your `PATH` if it is not already there.

Verify the installation:
```bash
plint --help
```

## Connect to Claude Code

Add the following hook to your Claude Code settings file (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "plint analyze"
          }
        ]
      }
    ]
  }
}
```

This tells Claude Code to run `plint analyze` on every prompt you submit. PromptLinter reads the prompt from stdin, analyzes it, and returns feedback.

## Modes

PromptLinter runs in `suggest` mode by default. Configure it in `~/.promptlinter/config.json`:

- `suggest` — tips on stderr; prompt still goes through (default)
- `silent` — logs analysis results only, no visible output
- `auto` — blocks wasteful prompts and asks you to retype
