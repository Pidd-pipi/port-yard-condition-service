package notify

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNotifyStopThenDispatchNoPanic(t *testing.T) {
	d := NewDispatcher([]Sender{NewLogSender(100)}, 4, 1)
	d.Start()
	if err := d.Dispatch(context.Background(), Message{Channel: "c", Body: "b", At: time.Now()}); err != nil {
		t.Fatalf("dispatch before stop: %v", err)
	}
	d.Stop()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("dispatch after stop panicked: %v", recovered)
		}
	}()
	_ = d.Dispatch(context.Background(), Message{Channel: "c", Body: "b", At: time.Now()})
}

func TestNotifyConcurrentStatsRace(t *testing.T) {
	d := NewDispatcher([]Sender{NewLogSender(5000)}, 64, 2)
	d.Start()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 40; j++ {
				_ = d.Dispatch(context.Background(), Message{Channel: "c", Body: "b", At: time.Now()})
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 40; j++ {
				_ = d.stats.Snapshot()
			}
		}()
	}
	close(start)
	wg.Wait()
	d.Stop()
}
