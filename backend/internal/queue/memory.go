package queue

import (
	"context"
	"sync"

	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// MemoryQueue is an unbounded in-memory channel-backed queue. It is the default
// so the system runs without RabbitMQ. It supports graceful stop: Stop cancels
// the context so in-flight workers finish their current job and exit.
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
	q.jobs = make(chan Job, 1024)
	q.cancel = cancel
	q.started = true
	jobs := q.jobs
	q.mu.Unlock()

	for i := 0; i < q.conc; i++ {
		q.wg.Add(1)
		go q.worker(ctx, jobs)
	}
	logger.L.Info("index queue started", zap.Int("workers", q.conc))
	return nil
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.started || q.jobs == nil {
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
	if !q.started {
		q.mu.Unlock()
		return nil
	}
	cancel := q.cancel
	jobs := q.jobs
	q.cancel = nil
	q.jobs = nil
	q.started = false
	if cancel != nil {
		cancel()
	}
	close(jobs)
	q.mu.Unlock()
	q.wg.Wait()
	return nil
}

// worker drains jobs until the context is cancelled. Each job is retried up to
// maxRetries times (per the non-functional requirement of 3 retries); after
// that the processor records the failure on the document.
func (q *MemoryQueue) worker(ctx context.Context, jobs <-chan Job) {
	defer q.wg.Done()
	const maxRetries = 3
	for job := range jobs {
		var err error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err = q.processor.Process(ctx, job); err == nil {
				break
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
		if err != nil {
			logger.L.Error("index job permanently failed",
				zap.Stringer("doc", job.DocumentID), zap.Error(err))
		}
	}
}
