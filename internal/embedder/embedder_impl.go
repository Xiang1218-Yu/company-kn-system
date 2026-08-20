package embedder

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strconv"

	"kn-system/internal/config"
	"kn-system/pkg/openai"
)

// MockEmbedder produces a deterministic hash-based vector. It exists so the
// system runs end-to-end with no external API key, while still exercising the
// real retrieval pipeline (store vectors, run cosine similarity). Two equal
// texts always map to the same vector; similar texts are not guaranteed to be
// near in vector space — this is a functional, not quality, stand-in.
type MockEmbedder struct {
	dim int
}

func NewMock(dim int) *MockEmbedder {
	return &MockEmbedder{dim: dim}
}

func (m *MockEmbedder) Dim() int { return m.dim }

// Embed hashes the text and stretches the 64-bit digest across `dim` slots,
// then L2-normalizes the result so cosine similarity is well-defined.
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, m.dim)
	if m.dim == 0 {
		return nil, fmt.Errorf("embedder dim is zero")
	}
	for i := 0; i < m.dim; i++ {
		// Mix the slot index into the hash so adjacent slots differ rather
		// than repeating the same digest.
		h := fnv.New64a()
		h.Write([]byte(strconv.Itoa(i)))
		h.Write([]byte{textSeparator})
		h.Write([]byte(text))
		v := float64(h.Sum64()%1000) / 1000.0 // [0,1)
		vec[i] = float32(v*2 - 1) // map to [-1,1)
	}
	normalize(vec)
	return vec, nil
}

// textSeparator keeps the index and text from forming an ambiguous boundary
// in the hash input.
const textSeparator = '|'

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
}

// OpenAIEmbedder calls the OpenAI embeddings API via the shared HTTP client.
type OpenAIEmbedder struct {
	client *openai.Client
	model  string
	dim    int
}

func NewOpenAI(cfg config.EmbedderConfig) (*OpenAIEmbedder, error) {
	c, err := openai.New(cfg.APIKey, cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	return &OpenAIEmbedder{client: c, model: cfg.Model, dim: cfg.Dim}, nil
}

func (e *OpenAIEmbedder) Dim() int { return e.dim }

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := e.client.Embed(ctx, e.model, text)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	return resp, nil
}

// New selects the configured embedder, defaulting to the mock when no provider
// or key is set so the system is runnable without external credentials.
func New(cfg config.EmbedderConfig) (Embedder, error) {
	switch cfg.Provider {
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("embedder.provider=openai but EMBEDDER_API_KEY is empty")
		}
		return NewOpenAI(cfg)
	default:
		return NewMock(cfg.Dim), nil
	}
}
