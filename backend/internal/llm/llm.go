package llm

import "context"

// Message is one turn in a conversation. Role follows the OpenAI convention
// (system/user/assistant) so it maps cleanly to most providers.
type Message struct {
	Role    string
	Content string
}

// LLM generates an answer from a prompt. The streaming variant is what powers
// SSE question answering; non-streaming callers use Complete.
type LLM interface {
	Complete(ctx context.Context, messages []Message) (string, error)
	Stream(ctx context.Context, messages []Message, onToken func(string)) error
	Name() string
}
