package assistant

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	ErrJobQueueFull   = fmt.Errorf("job queue is full")
	ErrJobQueueClosed = fmt.Errorf("job queue is closed")
)

// Task is interface for Job
type Task interface {
	Execute(ctx context.Context) (interface{}, error)
}

// Job represents a task
type Job struct {
	ID   int
	Task Task
}

// Result represents the execution result of a task
type Result struct {
	JobID     int
	Value     interface{}
	Err       error
	StartTime time.Time
	EndTime   time.Time
}

// WorkerPool represents a worker pool
type WorkerPool struct {
	workerCount int
	jobQueue    chan Job
	resultChan  chan Result
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, workerSize int, jobQueueSize int) (*WorkerPool, error) {
	if workerSize <= 0 {
		return nil, fmt.Errorf("workerCount must be greater than 0")
	}

	ctx, cancel := context.WithCancel(ctx)
	return &WorkerPool{
		workerCount: workerSize,
		jobQueue:    make(chan Job, jobQueueSize),
		resultChan:  make(chan Result, jobQueueSize),
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Go(func() { wp.worker() })
	}
}

// worker is the goroutine that actually executes tasks
func (wp *WorkerPool) worker() {
	for {
		select {
		case job, ok := <-wp.jobQueue:
			if !ok {
				return
			}
			var err error
			var value interface{}
			startTime := time.Now()
			if job.Task != nil {
				value, err = job.Task.Execute(wp.ctx)
			} else {
				err = fmt.Errorf("task is nil")
			}
			endTime := time.Now()
			wp.resultChan <- Result{
				JobID:     job.ID,
				Value:     value,
				Err:       err,
				StartTime: startTime,
				EndTime:   endTime,
			}
		case <-wp.ctx.Done():
			return
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(job Job, timeout time.Duration) error {
	if wp.closed {
		return ErrJobQueueClosed
	}
	select {
	case <-wp.ctx.Done():
		return ErrJobQueueClosed
	case wp.jobQueue <- job:
		return nil
	case <-time.After(timeout):
		return ErrJobQueueFull
	}
}

// Wait waits for all workers to complete and closes the result channel
func (wp *WorkerPool) Wait() {
	close(wp.jobQueue)
	wp.wg.Wait()
	close(wp.resultChan)
}

// Results returns the result channel
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultChan
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.closed = true
	wp.cancel()
}
