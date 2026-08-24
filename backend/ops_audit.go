package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var opsAuditSequence uint64

func newOpsAuditID() string { return fmt.Sprintf("evt-%06d", atomic.AddUint64(&opsAuditSequence, 1)) }

type OpsAudit struct {
	mu     sync.RWMutex
	events []OpsEvent
	cap    int
}

func newOpsAudit() *OpsAudit { return &OpsAudit{events: []OpsEvent{}, cap: 500} }
func (a *OpsAudit) Add(recordID, typ, actor string) OpsEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	event := OpsEvent{ID: newOpsAuditID(), RecordID: recordID, Type: typ, Actor: actor, At: time.Now().UTC().Format(time.RFC3339Nano)}
	a.events = append(a.events, event)
	// Bound the trail on every write so memory stays predictable between
	// sweeps; keep the newest events and drop the oldest overflow.
	if a.cap > 0 && len(a.events) > a.cap {
		overflow := len(a.events) - a.cap
		a.events = append([]OpsEvent(nil), a.events[overflow:]...)
	}
	return event
}
func (a *OpsAudit) For(recordID string) []OpsEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := []OpsEvent{}
	for _, event := range a.events {
		if event.RecordID == recordID {
			out = append(out, event)
		}
	}
	return out
}
func (a *OpsAudit) Since(start time.Time) []OpsEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := []OpsEvent{}
	for _, event := range a.events {
		parsed, err := time.Parse(time.RFC3339Nano, event.At)
		if err == nil && !parsed.Before(start) {
			out = append(out, event)
		}
	}
	return out
}
func (a *OpsAudit) Count() int { a.mu.RLock(); defer a.mu.RUnlock(); return len(a.events) }
func (a *OpsAudit) Latest() (OpsEvent, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.events) == 0 {
		return OpsEvent{}, false
	}
	return a.events[len(a.events)-1], true
}
func (a *OpsAudit) Recent(limit int) []OpsEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if limit <= 0 || limit > len(a.events) {
		limit = len(a.events)
	}
	// The newest events live at the tail; return the last `limit` in
	// chronological order so callers never receive stale events.
	start := len(a.events) - limit
	out := make([]OpsEvent, 0, limit)
	out = append(out, a.events[start:]...)
	return out
}

// Trim drops the oldest events until the audit trail fits within max,
// keeping the most recent max entries.
func (a *OpsAudit) Trim(max int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if max < 0 {
		return
	}
	if len(a.events) > max {
		overflow := len(a.events) - max
		a.events = append([]OpsEvent(nil), a.events[overflow:]...)
	}
}
func (a *OpsAudit) Clear() { a.mu.Lock(); defer a.mu.Unlock(); a.events = a.events[:0] }
