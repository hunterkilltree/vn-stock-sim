package order

import "sync"

// MemoryStore is an in-memory, per-user order history. No database wired
// up for V1 yet (see RESUME.md).
type MemoryStore struct {
	mu     sync.Mutex
	byUser map[string][]Order
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byUser: make(map[string][]Order)}
}

func (s *MemoryStore) Append(userID string, o Order) Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	o.ID = idOf(s.nextID)
	s.byUser[userID] = append(s.byUser[userID], o)
	return o
}

func (s *MemoryStore) List(userID string) []Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Order, len(s.byUser[userID]))
	copy(out, s.byUser[userID])
	return out
}

func (s *MemoryStore) ByID(userID, id string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, o := range s.byUser[userID] {
		if o.ID == id {
			return o, true
		}
	}
	return Order{}, false
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
	return "ord_" + string(buf)
}

// Replace overwrites the stored order with the same ID (used by Cancel to
// persist a status transition).
func (s *MemoryStore) Replace(userID string, updated Order) Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.byUser[userID]
	for i, o := range list {
		if o.ID == updated.ID {
			list[i] = updated
			return updated
		}
	}
	return updated
}
