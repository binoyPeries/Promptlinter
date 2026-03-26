package hook

// HookInput is the JSON payload from Claude Code's UserPromptSubmit hook.
type HookInput struct {
	SessionID      string `json:"session_id"`
	Prompt         string `json:"prompt"`
	Cwd            string `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`
	HookEventName  string `json:"hook_event_name"`
}
