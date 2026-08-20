package queue

import "errors"

// ErrFull is returned when the in-memory queue buffer is saturated. Callers
// map it to a transient failure the user can retry from the document list.
var ErrFull = errors.New("queue: full")

// ErrNotRunning is returned when a job is enqueued while the queue is stopped.
// Submitting after Stop must surface a recognizable error rather than panic on
// a closed channel, and it lets the caller distinguish "queue down" from a
// transient full buffer.
var ErrNotRunning = errors.New("queue: not running")

// New selects a queue backend. Only "memory" is implemented here; "redis" is
// reserved for when horizontal scaling is required and the interface lets that
// be added without touching the service layer.
func New(provider string, processor Processor, concurrency int) (Queue, error) {
	switch provider {
	case "redis":
		// Placeholder: a Redis-Stream implementation would live in redis.go
		// and be selected here. Left unimplemented to keep the stack runnable
		// without RabbitMQ/Redis Streams in the default profile.
		return nil, errors.New("queue: redis provider not yet implemented; use memory")
	default:
		return NewMemory(processor, concurrency), nil
	}
}
