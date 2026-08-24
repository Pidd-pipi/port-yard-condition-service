package notify

import (
	"context"
	"sync"
	"time"
)

// LogSender records delivered messages in a bounded ring buffer and never fails.
type LogSender struct {
	mu    sync.RWMutex
	items []Message
	limit int
}

// NewLogSender builds a sender that retains at most limit messages.
func NewLogSender(limit int) *LogSender {
	if limit < 1 {
		limit = 500
	}
	return &LogSender{items: []Message{}, limit: limit}
}

// Send appends the message to the ring buffer.
func (s *LogSender) Send(_ context.Context, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, msg)
	if len(s.items) > s.limit {
		s.items = s.items[len(s.items)-s.limit:]
	}
	return nil
}

// Recent returns the most recent delivered messages.
func (s *LogSender) Recent(limit int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.items) {
		limit = len(s.items)
	}
	out := make([]Message, 0, limit)
	for i := len(s.items) - limit; i < len(s.items); i++ {
		out = append(out, s.items[i])
	}
	return out
}

// Count returns how many messages are retained.
func (s *LogSender) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// RetrySender wraps another sender and retries transient failures with backoff.
type RetrySender struct {
	inner   Sender
	max     int
	backoff func(attempt int) time.Duration
}

// NewRetrySender builds a sender that retries up to maxAttempts times.
func NewRetrySender(inner Sender, maxAttempts int) *RetrySender {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &RetrySender{inner: inner, max: maxAttempts, backoff: func(attempt int) time.Duration {
		if attempt > 5 {
			attempt = 5
		}
		return time.Duration(1<<uint(attempt)) * 10 * time.Millisecond
	}}
}

// Send attempts delivery, honoring context cancellation between retries.
func (s *RetrySender) Send(ctx context.Context, msg Message) error {
	var err error
	for attempt := 0; attempt < s.max; attempt++ {
		if err = s.inner.Send(ctx, msg); err == nil {
			return nil
		}
		timer := time.NewTimer(s.backoff(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
