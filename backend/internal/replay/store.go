package replay

import "sync"

// Store is the Replay session port: MemoryStore or PGStore
// (phase-persistence.md decision 3). Save persists a session the service
// changed; MemoryStore's is a no-op, since it hands out the stored pointer.
type Store interface {
	nextSessionID() string
	Append(userID string, sess *Session)
	Get(userID, id string) (*Session, bool)
	Save(sess *Session)
}

// MemoryStore is an in-memory, per-user Replay session store -- same
// pattern as every other feature (order, backtest). Used when
// DATABASE_URL is unset; PGStore otherwise (phase-persistence.md).
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

// Save is a no-op: Get returns the stored pointer itself.
func (s *MemoryStore) Save(*Session) {}

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
