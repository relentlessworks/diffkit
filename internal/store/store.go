package store

import (
	"sync/atomic"
)

// Stats tracks in-memory usage counters.
type Stats struct {
	DiffsGenerated int64
	PatchesApplied int64
	WordDiffs      int64
}

// Store is an in-memory usage tracker. No persistence needed for a stateless service.
type Store struct {
	diffs  int64
	patches int64
	words  int64
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{}
}

func (s *Store) IncrDiff()   { atomic.AddInt64(&s.diffs, 1) }
func (s *Store) IncrPatch()  { atomic.AddInt64(&s.patches, 1) }
func (s *Store) IncrWord()   { atomic.AddInt64(&s.words, 1) }

// GetStats returns a snapshot of current usage counters.
func (s *Store) GetStats() Stats {
	return Stats{
		DiffsGenerated: atomic.LoadInt64(&s.diffs),
		PatchesApplied: atomic.LoadInt64(&s.patches),
		WordDiffs:      atomic.LoadInt64(&s.words),
	}
}
