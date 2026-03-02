package tokenizer

import "github.com/pkoukk/tiktoken-go"

// Counter counts tokens in text.
type Counter struct {
	enc *tiktoken.Tiktoken
}

// New creates a Counter using the cl100k_base encoding.
// NOTE: even though this is the same encoding used by OpenAI, it works well for Claude too.
func New() (*Counter, error) {
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}
	return &Counter{enc: enc}, nil
}

// Count returns the number of tokens in the given text.
func (c *Counter) Count(text string) int {
	return len(c.enc.Encode(text, nil, nil))
}
