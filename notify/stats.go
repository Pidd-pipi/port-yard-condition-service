package notify

// Stats holds cumulative dispatch counters.
type Stats struct {
	Dispatched int64
	Delivered  int64
	Failed     int64
}

// Snapshot returns a consistent copy of the counters.
func (s *Stats) Snapshot() Stats {
	return Stats{
		Dispatched: s.Dispatched,
		Delivered:  s.Delivered,
		Failed:     s.Failed,
	}
}
