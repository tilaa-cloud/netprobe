package ping

import (
	"context"
	"sync"

	"netprobe/internal/target"
)

// Job represents a single ping job to be executed
type Job struct {
	Target target.Target
}

// WorkerPool manages concurrent ping workers
type WorkerPool struct {
	workers       int
	jobs          chan Job
	results       chan PingResult
	wg            sync.WaitGroup
	executor      *Executor
	ctx           context.Context
	cancel        context.CancelFunc
	closeJobsOnce sync.Once
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(numWorkers int, executor *Executor) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		workers:  numWorkers,
		jobs:     make(chan Job, numWorkers*2),
		results:  make(chan PingResult, numWorkers*2),
		executor: executor,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

// worker processes jobs from the jobs channel
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for {
		select {
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}

			// Execute pings for this target
			results, err := wp.executor.Ping(wp.ctx, job.Target)
			if err == nil {
				for _, result := range results {
					select {
					case wp.results <- result:
					case <-wp.ctx.Done():
						return
					}
				}
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// SubmitBatch submits a batch of targets to be pinged
func (wp *WorkerPool) SubmitBatch(targets []target.Target) {
	for _, t := range targets {
		wp.jobs <- Job{Target: t}
	}
}

// CollectResults returns the results channel
func (wp *WorkerPool) CollectResults() chan PingResult {
	return wp.results
}

// Stop stops all workers and waits for them to finish
func (wp *WorkerPool) Stop() {
	wp.closeJobsOnce.Do(func() {
		close(wp.jobs)
	})
	wp.wg.Wait()
	wp.CloseResults()
}

// CloseJobs closes the jobs channel to signal no more jobs are coming
func (wp *WorkerPool) CloseJobs() {
	wp.closeJobsOnce.Do(func() {
		close(wp.jobs)
	})
}

// WaitForWorkers waits for all workers to finish processing
func (wp *WorkerPool) WaitForWorkers() {
	wp.wg.Wait()
}

// CloseResults closes the results channel
func (wp *WorkerPool) CloseResults() {
	close(wp.results)
}
