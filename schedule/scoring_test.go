package schedule

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleStopStopsTicks(t *testing.T) {
	var counter int32
	runner := NewRunner(Job{
		Name:     "tick",
		Interval: 10 * time.Millisecond,
		Run: func(ctx context.Context) error {
			atomic.AddInt32(&counter, 1)
			return nil
		},
	})
	runner.Start()
	time.Sleep(50 * time.Millisecond)
	runner.Stop()
	after := atomic.LoadInt32(&counter)
	time.Sleep(60 * time.Millisecond)
	if got := atomic.LoadInt32(&counter); got != after {
		t.Fatalf("ticks continued after Stop: before=%d after=%d", after, got)
	}
}
