package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"kn-system/pkg/logger"
)

type lifecycleProcessor struct {
	mu   sync.Mutex
	jobs []Job
}

func (p *lifecycleProcessor) Process(_ context.Context, job Job) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobs = append(p.jobs, job)
	return nil
}

func (p *lifecycleProcessor) contains(id uuid.UUID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, job := range p.jobs {
		if job.DocumentID == id {
			return true
		}
	}
	return false
}

func TestMemoryQueueRejectsStoppedEnqueueAndRestarts(t *testing.T) {
	logger.Init("test")
	processor := &lifecycleProcessor{}
	q := NewMemory(processor, 2)
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("start queue: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := q.Enqueue(context.Background(), Job{DocumentID: uuid.New()}); err != nil {
				t.Errorf("enqueue while running: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if err := q.Stop(); err != nil {
		t.Fatalf("stop queue: %v", err)
	}

	var stoppedErr error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("enqueue after stop panicked: %v", recovered)
			}
		}()
		stoppedErr = q.Enqueue(context.Background(), Job{DocumentID: uuid.New()})
	}()
	if stoppedErr == nil || stoppedErr.Error() != "queue: not running" {
		t.Fatalf("enqueue after stop error = %v, want queue: not running", stoppedErr)
	}

	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("restart queue: %v", err)
	}
	defer q.Stop()

	restartedID := uuid.New()
	if err := q.Enqueue(context.Background(), Job{DocumentID: restartedID}); err != nil {
		t.Fatalf("enqueue after restart: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for !processor.contains(restartedID) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !processor.contains(restartedID) {
		t.Fatal("restarted queue did not process the new job")
	}
}
