# PromptLinter

A real-time prompt efficiency analyzer and coach for Claude Code users. Intercepts prompts before they reach Claude, detects waste, suggests concise rewrites, and tracks your prompt quality over time with detailed reports.

## Installation

### 1. Download the binary

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

Verify the installation:
```bash
plint --help
```

### 2. Connect to Claude Code

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

### 3. Modes

PromptLinter runs in **suggest** mode by default — it shows tips on stderr without blocking your prompt. You can change the mode in `~/.prompt-optimizer/config.json`:

- `suggest` — inline tips shown before the prompt runs (default)
- `silent` — logs analysis results only, no visible output
- `auto` — blocks wasteful prompts and asks you to retype