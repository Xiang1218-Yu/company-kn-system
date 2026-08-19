package embedder

import "context"

// Embedder turns text into dense vectors for semantic retrieval. The interface
// isolates the rest of the system from any specific provider so swapping
// OpenAI embeddings for a self-hosted BGE model is a one-file change.
type Embedder interface {
	// Embed returns a single vector for the given text. Implementations should
	// truncate or error on inputs exceeding their model's context length.
	Embed(ctx context.Context, text string) ([]float32, error)
	// Dim reports the dimensionality of vectors this embedder produces. The
	// chunks.vector column is sized to match at migration time.
	Dim() int
}
