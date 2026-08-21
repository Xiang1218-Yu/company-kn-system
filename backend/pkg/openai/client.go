// Package openai is a thin HTTP client for the OpenAI-compatible REST API.
// It is shared by the embedder and llm packages so neither duplicates transport
// or auth logic. It depends only on the standard library plus go-openai types
// for request/response shaping, keeping the boundary small.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client posts JSON to an OpenAI-compatible endpoint. BaseURL and APIKey are
// configurable so the same client talks to OpenAI, Azure, or a local gateway.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New returns a Client bound to the given key and base URL.
func New(apiKey, baseURL string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai: api key required")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// EmbedRequest is the body for /embeddings.
type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// EmbedResponse carries the returned vector(s).
type EmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed posts one text and returns its embedding vector.
func (c *Client) Embed(ctx context.Context, model, input string) ([]float32, error) {
	body, err := json.Marshal(EmbedRequest{Model: model, Input: input})
	if err != nil {
		return nil, err
	}
	resp, err := c.post(ctx, "/embeddings", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embeddings: %s: %s", resp.Status, string(b))
	}
	var out EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("openai embeddings: empty data")
	}
	return out.Data[0].Embedding, nil
}

// ChatMessage is one role/content pair in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the body for /chat/completions. Stream toggles SSE.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatChoice is one completion in the non-streaming response.
type ChatChoice struct {
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatResponse is the non-streaming completion payload.
type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
}

// Chat does a single (non-streaming) completion and returns the first choice.
func (c *Client) Chat(ctx context.Context, model string, messages []ChatMessage) (*ChatChoice, error) {
	body, err := json.Marshal(ChatRequest{Model: model, Messages: messages, Stream: false})
	if err != nil {
		return nil, err
	}
	resp, err := c.post(ctx, "/chat/completions", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai chat: %s: %s", resp.Status, string(b))
	}
	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("openai chat: no choices")
	}
	return &out.Choices[0], nil
}

// ChatStream does a streaming completion. It calls onToken for each text delta
// and returns once the stream closes. Used for SSE question answering.
func (c *Client) ChatStream(ctx context.Context, model string, messages []ChatMessage, onToken func(string)) error {
	body, err := json.Marshal(ChatRequest{Model: model, Messages: messages, Stream: true})
	if err != nil {
		return err
	}
	resp, err := c.post(ctx, "/chat/completions", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai chat stream: %s: %s", resp.Status, string(b))
	}
	dec := json.NewDecoder(resp.Body)
	for {
		// Honor cancellation mid-stream: decoding the SSE body blocks on the
		// provider, so without this check a canceled request keeps draining
		// tokens until the upstream closes the connection. The request itself
		// is already bound to ctx, so once ctx is canceled the underlying
		// transport aborts; this select lets us stop promptly even when bytes
		// are still arriving.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var ev struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		for _, ch := range ev.Choices {
			if ch.Delta.Content != "" {
				onToken(ch.Delta.Content)
			}
		}
	}
}

func (c *Client) post(ctx context.Context, path string, body []byte) (*http.Response, error) {
	// Bail out before any HTTP work when the caller already canceled. The
	// request is bound to ctx below, but net/http only aborts an in-flight
	// transport call asynchronously — a synchronously completing transport (or a
	// provider that responds within the same tick) can return before the
	// cancellation watcher fires. Checking ctx up front makes cancellation
	// prompt and observable: no outbound request is made for an already-canceled
	// call, and the returned error is the canceled context so callers can
	// recognize it.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return c.http.Do(req)
}
