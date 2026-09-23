package replay

import "sync"

// MemoryStore is an in-memory, per-user Replay session store -- same
// pattern as every other V1 feature (order, backtest). No database wired
// up for V1 yet (see RESUME.md).
type MemoryStore struct {
	mu     sync.Mutex
	byUser map[string][]*Session
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byUser: make(map[string][]*Session)}
}

func (s *MemoryStore) nextSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return "rp_" + itoa(s.nextID)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// Append stores a newly-started session under userID.
func (s *MemoryStore) Append(userID string, sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byUser[userID] = append(s.byUser[userID], sess)
}

// Get returns the session only if it belongs to userID, so callers can't
// act on another user's session by guessing an id.
func (s *MemoryStore) Get(userID, id string) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.byUser[userID] {
		if sess.ID == id {
			return sess, true
		}
	}
	return nil, false
}
