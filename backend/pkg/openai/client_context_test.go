package openai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type recordingTransport struct {
	called atomic.Bool
}

func (t *recordingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.called.Store(true)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}]}`)),
		Header:     make(http.Header),
	}, nil
}

func TestOpenAIClientPropagatesCanceledContext(t *testing.T) {
	transport := &recordingTransport{}
	client := &Client{
		baseURL: "http://provider.invalid",
		apiKey:  "test-key",
		http:    &http.Client{Transport: transport},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Chat(ctx, "test-model", []ChatMessage{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected canceled context to stop the provider request")
	}
	if transport.called.Load() {
		t.Fatal("canceled request reached the provider transport")
	}
}
