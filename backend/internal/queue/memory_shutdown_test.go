package queue

import (
	"context"
	"testing"
	"time"

	"kn-system/pkg/logger"
)

type shutdownProcessor struct{}

func (shutdownProcessor) Process(context.Context, Job) error {
	return nil
}

func TestConcurrentStopAndEnqueueNeverSendsToClosedQueue(t *testing.T) {
	logger.Init("test")
	q := NewMemory(shutdownProcessor{}, 1)
	if err := q.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	closed := make(chan struct{})
	release := make(chan struct{})
	q.afterClose = func() {
		close(closed)
		<-release
	}

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		_ = q.Stop()
	}()
	<-closed

	panicSeen := make(chan any, 1)
	go func() {
		defer func() {
			panicSeen <- recover()
		}()
		_ = q.Enqueue(context.Background(), Job{})
	}()
	select {
	case p := <-panicSeen:
		close(release)
		<-stopDone
		if p != nil {
			t.Fatalf("enqueue panicked after the jobs channel was closed: %v", p)
		}
	case <-time.After(time.Second):
		close(release)
		<-stopDone
		t.Fatal("enqueue did not complete")
	}
}
