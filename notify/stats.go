package notify

import "sync/atomic"

// Stats holds cumulative dispatch counters.
type Stats struct {
	Dispatched int64
	Delivered  int64
	Failed     int64
}

// Snapshot returns a consistent copy of the counters.
func (s *Stats) Snapshot() Stats {
	return Stats{
		Dispatched: atomic.LoadInt64(&s.Dispatched),
		Delivered:  atomic.LoadInt64(&s.Delivered),
		Failed:     atomic.LoadInt64(&s.Failed),
	}
}
