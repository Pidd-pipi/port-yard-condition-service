package notify

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type failingSender struct {
	attempts *int32
}

func (f failingSender) Send(_ context.Context, _ Message) error {
	atomic.AddInt32(f.attempts, 1)
	return errors.New("boom")
}

func TestRetrySenderHonorsCancel(t *testing.T) {
	var attempts int32
	sender := NewRetrySender(failingSender{attempts: &attempts}, 3)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sender.Send(ctx, Message{Channel: "c", Body: "b", At: time.Now()})
	if err == nil {
		t.Fatal("expected a context cancellation error")
	}
	if got := atomic.LoadInt32(&attempts); got > 1 {
		t.Fatalf("inner send attempts = %d, want at most 1 (must stop after cancel)", got)
	}
}
