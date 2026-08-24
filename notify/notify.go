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
	wg      sync.WaitGroup
	stats   *Stats
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDispatcher builds a dispatcher with the given senders and worker count.
func NewDispatcher(senders []Sender, queueSize, workers int) *Dispatcher {
	if queueSize < 1 {
		queueSize = 64
	}
	if workers < 1 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		senders: senders,
		queue:   make(chan Message, queueSize),
		workers: workers,
		done:    make(chan struct{}),
		stats:   &Stats{},
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start launches the worker pool.
func (d *Dispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker()
	}
}

// Dispatch enqueues a message, honoring cancellation.
func (d *Dispatcher) Dispatch(ctx context.Context, msg Message) error {
	atomic.AddInt64(&d.stats.Dispatched, 1)
	select {
	case d.queue <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) deliver(msg Message) {
	for _, sender := range d.senders {
		// Deliveries use the dispatcher's own context so that Stop cancels
		// any in-flight retry backoff instead of waiting it out.
		if err := sender.Send(d.ctx, msg); err != nil {
			atomic.AddInt64(&d.stats.Failed, 1)
		} else {
			atomic.AddInt64(&d.stats.Delivered, 1)
		}
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for {
		select {
		case msg, ok := <-d.queue:
			if !ok {
				return
			}
			d.deliver(msg)
		default:
			select {
			case msg, ok := <-d.queue:
				if !ok {
					return
				}
				d.deliver(msg)
			case <-d.done:
				return
			}
		}
	}
}

// Stop signals the workers to finish and waits for them to exit. Cancelling the
// dispatcher context interrupts in-flight retry backoff so workers exit
// promptly instead of draining their full retry schedule.
func (d *Dispatcher) Stop() {
	d.cancel()
	close(d.done)
	d.wg.Wait()
}
