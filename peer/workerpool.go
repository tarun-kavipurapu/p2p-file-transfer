package peer

import (
	"sync"
)

type Job interface {
	Execute() error
}

type Result struct {
	Job Job
	Err error
}

type WorkerPool struct {
	maxWorkers int
	jobs       chan Job
	results    chan Result
	done       chan struct{}
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		jobs:       make(chan Job),
		results:    make(chan Result),
		done:       make(chan struct{}),
	}
}

func (wp *WorkerPool) Start() {
	var wg sync.WaitGroup
	for i := 0; i < wp.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range wp.jobs {
				result := Result{Job: job, Err: job.Execute()}
				wp.results <- result
			}
		}()
	}

	go func() {
		wg.Wait()
		close(wp.results)
		close(wp.done)
	}()
}

func (wp *WorkerPool) Submit(job Job) {
	wp.jobs <- job
}

func (wp *WorkerPool) Results() <-chan Result {
	return wp.results
}

func (wp *WorkerPool) Done() <-chan struct{} {
	return wp.done
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
}
