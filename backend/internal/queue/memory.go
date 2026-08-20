package queue

import (
	"context"
	"sync"

	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// MemoryQueue is an unbounded in-memory channel-backed queue. It is the default
// so the system runs without RabbitMQ. It supports graceful stop and restart:
// Stop cancels the run context so in-flight workers finish their current job,
// drain anything already buffered, and exit. The jobs channel is deliberately
// never closed — Go channels cannot be reopened, and closing it would make a
// post-stop Enqueue panic on send-to-closed. Instead Enqueue gates on the
// started flag and returns ErrNotRunning while stopped, so Start can be called
// again to resume processing new submissions on the same channel.
//
// It is intentionally not durable: a process restart drops pending jobs. For
// durability, switch QueueConfig.Provider to "redis" (the Redis implementation
// is the natural extension point — the interface stays the same).
type MemoryQueue struct {
	processor Processor
	jobs      chan Job
	conc      int
	wg        sync.WaitGroup
	cancel    context.CancelFunc
	mu        sync.Mutex
	started   bool
}

func NewMemory(processor Processor, concurrency int) *MemoryQueue {
	if concurrency < 1 {
		concurrency = 1
	}
	return &MemoryQueue{
		processor: processor,
		jobs:      make(chan Job, 1024),
		conc:      concurrency,
	}
}

func (q *MemoryQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	q.cancel = cancel
	q.started = true
	q.mu.Unlock()

	for i := 0; i < q.conc; i++ {
		q.wg.Add(1)
		go q.worker(ctx)
	}
	logger.L.Info("index queue started", zap.Int("workers", q.conc))
	return nil
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job Job) error {
	// The running check is the lifecycle gate. While stopped, Enqueue returns a
	// recognizable error instead of touching the channel — this is what makes
	// stop/restart safe: we never send on a closed channel, so no panic.
	q.mu.Lock()
	running := q.started
	q.mu.Unlock()
	if !running {
		return ErrNotRunning
	}
	select {
	case q.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer full is treated as a transient error; the caller (service)
		// records it on the document as a failed-to-queue state.
		return ErrFull
	}
}

func (q *MemoryQueue) Stop() error {
	q.mu.Lock()
	q.started = false
	cancel := q.cancel
	q.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	// Wait outside the lock: workers don't take q.mu, but holding it across a
	// potentially long drain would block Enqueue's started check needlessly.
	q.wg.Wait()
	return nil
}

// worker pulls jobs until the run context is cancelled. Each job is retried up
// to maxRetries times (per the non-functional requirement of 3 retries); after
// that the processor records the failure on the document. On cancellation the
// worker drains whatever is already buffered before exiting, so a graceful stop
// still processes queued work — matching the prior close-and-range behavior.
func (q *MemoryQueue) worker(ctx context.Context) {
	defer q.wg.Done()
	const maxRetries = 3
	for {
		select {
		case job, ok := <-q.jobs:
			if !ok {
				return
			}
			q.runJob(ctx, job, maxRetries)
		case <-ctx.Done():
			q.drain(ctx, maxRetries)
			return
		}
	}
}

// drain processes jobs already sitting in the buffer at stop time so they are
// not abandoned. It is non-blocking once the buffer empties; new enqueues
// concurrently see started==false and return ErrNotRunning, so it terminates.
func (q *MemoryQueue) drain(ctx context.Context, maxRetries int) {
	for {
		select {
		case job := <-q.jobs:
			q.runJob(ctx, job, maxRetries)
		default:
			return
		}
	}
}

// runJob processes one job with bounded retries, logging the permanent failure
// if all attempts are exhausted.
func (q *MemoryQueue) runJob(ctx context.Context, job Job, maxRetries int) {
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err = q.processor.Process(ctx, job); err == nil {
			return
		}
		logger.L.Warn("index attempt failed",
			zap.Stringer("doc", job.DocumentID),
			zap.Int("attempt", attempt),
			zap.Error(err))
		if attempt == maxRetries {
			break
		}
		// Retry inline; with a real queue we'd re-enqueue with a backoff.
	}
	logger.L.Error("index job permanently failed",
		zap.Stringer("doc", job.DocumentID), zap.Error(err))
}
