package watchlist

import (
	"sync"
	"time"
)

// entry is what's actually persisted per user; quote fields are resolved
// live from the symbol provider on read, not stored (store.go only owns
// "which symbols did this user add, and when").
type entry struct {
	symbol  string
	addedAt time.Time
}

// MemoryStore is an in-memory per-user watchlist. Same rationale as
// auth.MemoryStore: no database wired up for V1 yet.
type MemoryStore struct {
	mu    sync.RWMutex
	byUse map[string][]entry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byUse: make(map[string][]entry)}
}

func (s *MemoryStore) Add(userID, symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.byUse[userID] {
		if e.symbol == symbol {
			return
		}
	}
	s.byUse[userID] = append(s.byUse[userID], entry{symbol: symbol, addedAt: time.Now()})
}

func (s *MemoryStore) Remove(userID, symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.byUse[userID]
	for i, e := range list {
		if e.symbol == symbol {
			s.byUse[userID] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

func (s *MemoryStore) List(userID string) []entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]entry, len(s.byUse[userID]))
	copy(out, s.byUse[userID])
	return out
}
