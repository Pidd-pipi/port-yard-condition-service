package schedule

import (
	"context"
	"sync"
	"time"
)

// Job is a periodic background task.
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// Runner executes jobs on fixed intervals until Stop.
type Runner struct {
	mu       sync.Mutex
	jobs     []Job
	stop     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
	started  bool
	lastErr  map[string]string
}

// NewRunner builds a runner for the given jobs.
func NewRunner(jobs ...Job) *Runner {
	return &Runner{jobs: jobs, stop: make(chan struct{}), lastErr: map[string]string{}}
}

// Start launches one goroutine per job.
func (r *Runner) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	r.started = true
	for _, job := range r.jobs {
		r.wg.Add(1)
		go r.loop(job)
	}
}

func (r *Runner) loop(job Job) {
	defer r.wg.Done()
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := job.Run(ctx)
			cancel()
			r.mu.Lock()
			if err != nil {
				r.lastErr[job.Name] = err.Error()
			} else {
				delete(r.lastErr, job.Name)
			}
			r.mu.Unlock()
		case <-r.stop:
			return
		}
	}
}

// Stop signals all jobs to finish and waits for them to exit.
// It is idempotent: the first call closes the stop channel and drains the
// running goroutines; subsequent calls return immediately.
func (r *Runner) Stop() {
	r.stopOnce.Do(func() {
		close(r.stop)
	})
	r.wg.Wait()
}

// LastError returns the most recent error recorded for a job, if any.
func (r *Runner) LastError(name string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	err, ok := r.lastErr[name]
	return err, ok
}
