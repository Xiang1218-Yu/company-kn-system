package queue

import (
	"context"

	"github.com/google/uuid"
)

// Job is a unit of indexing work: turn one document's bytes into stored chunks
// with embeddings. Carrying only IDs keeps the queue payload small and lets the
// worker re-fetch the bytes from storage on pickup.
type Job struct {
	DocumentID uuid.UUID
	KbID       uuid.UUID
}

// Processor handles a Job. The queue depends on this interface so the worker
// loop is decoupled from the indexing implementation (which lives in service).
type Processor interface {
	Process(ctx context.Context, job Job) error
}

// Queue delivers Jobs to a Processor. Implementations may be in-memory (for
// single-instance deployments) or backed by Redis Stream (for horizontal
// scaling). Start begins delivery; Stop drains and shuts down.
type Queue interface {
	Start(ctx context.Context) error
	Enqueue(ctx context.Context, job Job) error
	Stop() error
}
