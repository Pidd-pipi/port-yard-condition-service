package notify

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Message is a single notification payload.
type Message struct {
	Channel string
	Body    string
	At      time.Time
}

// Sender delivers a message to one channel.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Dispatcher fans messages out to senders through a bounded queue.
type Dispatcher struct {
	senders []Sender
	queue   chan Message
	workers int
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup
	stats   *Stats
}

// NewDispatcher builds a dispatcher with the given senders and worker count.
func NewDispatcher(senders []Sender, queueSize, workers int) *Dispatcher {
	if queueSize < 1 {
		queueSize = 64
	}
	if workers < 1 {
		workers = 1
	}
	return &Dispatcher{
		senders: senders,
		queue:   make(chan Message, queueSize),
		workers: workers,
		done:    make(chan struct{}),
		stats:   &Stats{},
	}
}

// Start launches the worker pool.
func (d *Dispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker()
	}
}

// errDispatcherStopped is returned when a message is dispatched after Stop.
var errDispatcherStopped = errStopped{}

type errStopped struct{}

func (errStopped) Error() string { return "notify: dispatcher stopped" }

// Dispatch enqueues a message, honoring cancellation. It is safe to call
// after Stop: in that case the message is dropped and an error is returned
// rather than panicking on a send to a closed channel.
func (d *Dispatcher) Dispatch(ctx context.Context, msg Message) error {
	for {
		// Reject new work as soon as Stop has been signaled.
		select {
		case <-d.done:
			return errDispatcherStopped
		default:
		}
		select {
		case d.queue <- msg:
			atomic.AddInt64(&d.stats.Dispatched, 1)
			return nil
		case <-d.done:
			return errDispatcherStopped
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *Dispatcher) deliver(msg Message) {
	for _, sender := range d.senders {
		if err := sender.Send(context.Background(), msg); err != nil {
			atomic.AddInt64(&d.stats.Failed, 1)
		} else {
			atomic.AddInt64(&d.stats.Delivered, 1)
		}
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for msg := range d.queue {
		d.deliver(msg)
	}
}

// Snapshot returns a consistent point-in-time copy of the dispatcher's
// cumulative counters. It is safe to call concurrently with Dispatch.
func (d *Dispatcher) Snapshot() Stats {
	return d.stats.Snapshot()
}

// Stop signals the workers to finish and waits for them to exit. It is
// idempotent: the first call drains the queue and joins the workers;
// subsequent calls return immediately. Dispatch calls made concurrently
// with or after Stop are dropped safely.
func (d *Dispatcher) Stop() {
	d.once.Do(func() {
		close(d.done)
		close(d.queue)
	})
	d.wg.Wait()
}
