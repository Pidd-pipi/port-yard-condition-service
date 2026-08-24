package notify

import "sync/atomic"

// Stats holds cumulative dispatch counters. All fields are accessed
// atomically: writers use atomic.AddInt64 and Snapshot uses atomic loads,
// so the struct is safe for concurrent use by the dispatcher and its
// callers without external locking.
type Stats struct {
	Dispatched int64
	Delivered  int64
	Failed     int64
}

// Snapshot returns a consistent point-in-time copy of the counters.
func (s *Stats) Snapshot() Stats {
	return Stats{
		Dispatched: atomic.LoadInt64(&s.Dispatched),
		Delivered:  atomic.LoadInt64(&s.Delivered),
		Failed:     atomic.LoadInt64(&s.Failed),
	}
}
