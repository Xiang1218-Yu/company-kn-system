package llm

import (
	"context"
	"fmt"
	"strings"

	"kn-system/internal/config"
	"kn-system/pkg/openai"
)

// MockLLM produces a canned answer that echoes the retrieved context. It lets
// the full QA pipeline run — including citation assembly — without an API key.
// The answer makes the grounding explicit so a reviewer can see retrieval worked.
type MockLLM struct{}

func NewMock() *MockLLM { return &MockLLM{} }

func (m *MockLLM) Name() string { return "mock" }

// Complete synthesizes a deterministic answer from the last user message plus
// any context already baked into the prompt. Tokens are streamed word by word
// by delegating to Stream to keep output shaping consistent.
func (m *MockLLM) Complete(ctx context.Context, messages []Message) (string, error) {
	var sb strings.Builder
	err := m.Stream(ctx, messages, func(t string) { sb.WriteString(t) })
	return sb.String(), err
}

func (m *MockLLM) Stream(ctx context.Context, messages []Message, onToken func(string)) error {
	if len(messages) == 0 {
		return fmt.Errorf("llm: no messages")
	}
	last := messages[len(messages)-1].Content
	out := fmt.Sprintf(
		"[mock-llm] 基于知识库检索到的内容，针对「%s」的回答如下：\n\n"+
			"系统当前使用 mock 模型生成示例回答。配置 LLM_PROVIDER=openai 并设置 LLM_API_KEY 后即可切换为真实大模型。\n\n"+
			"已检索到相关上下文将作为来源引用展示，便于核对。", last,
	)
	// Emit word by word so the SSE path and the non-streaming path exercise the
	// same tokenization, and so the frontend's typing animation has content.
	for _, w := range strings.Fields(out) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		onToken(w + " ")
	}
	return nil
}

// OpenAILLM calls the chat completions endpoint via the shared client.
type OpenAILLM struct {
	client *openai.Client
	model  string
}

func NewOpenAI(cfg config.LLMConfig) (*OpenAILLM, error) {
	c, err := openai.New(cfg.APIKey, cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	return &OpenAILLM{client: c, model: cfg.Model}, nil
}

func (l *OpenAILLM) Name() string { return "openai:" + l.model }

func (l *OpenAILLM) Complete(ctx context.Context, messages []Message) (string, error) {
	om := make([]openai.ChatMessage, len(messages))
	for i, m := range messages {
		om[i] = openai.ChatMessage{Role: m.Role, Content: m.Content}
	}
	choice, err := l.client.Chat(ctx, l.model, om)
	if err != nil {
		return "", err
	}
	return choice.Message.Content, nil
}

func (l *OpenAILLM) Stream(ctx context.Context, messages []Message, onToken func(string)) error {
	om := make([]openai.ChatMessage, len(messages))
	for i, m := range messages {
		om[i] = openai.ChatMessage{Role: m.Role, Content: m.Content}
	}
	return l.client.ChatStream(ctx, l.model, om, onToken)
}

// New selects the configured LLM. Defaulting to mock keeps the system runnable
// without credentials, matching the embedder's behavior.
func New(cfg config.LLMConfig) (LLM, error) {
	switch cfg.Provider {
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("llm.provider=openai but LLM_API_KEY is empty")
		}
		return NewOpenAI(cfg)
	default:
		return NewMock(), nil
	}
}
