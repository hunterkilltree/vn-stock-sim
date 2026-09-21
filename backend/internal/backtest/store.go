package backtest

import "sync"

// MemoryStore is an in-memory backtest result store. api-spec.md's
// "queues onto the worker pool; poll GET /backtests/:id" contract is
// preserved in the response shape, but V1 runs the rule synchronously and
// stores it already "completed" — no real worker pool yet (see RESUME.md).
type MemoryStore struct {
	mu     sync.Mutex
	byUser map[string][]Backtest
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byUser: make(map[string][]Backtest)}
}

func (s *MemoryStore) Append(userID string, b Backtest) Backtest {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	b.ID = idOf(s.nextID)
	s.byUser[userID] = append(s.byUser[userID], b)
	return b
}

func (s *MemoryStore) List(userID string) []Backtest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Backtest, len(s.byUser[userID]))
	copy(out, s.byUser[userID])
	return out
}

func (s *MemoryStore) ByID(userID, id string) (Backtest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.byUser[userID] {
		if b.ID == id {
			return b, true
		}
	}
	return Backtest{}, false
}

func idOf(n int) string {
	const digits = "0123456789abcdef"
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{digits[n%16]}, buf...)
		n /= 16
	}
	if len(buf) == 0 {
		buf = []byte{'0'}
	}
	return "bt_" + string(buf)
}
