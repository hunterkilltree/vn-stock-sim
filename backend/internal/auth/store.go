package auth

import (
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var ErrEmailTaken = errors.New("email already registered")
var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrUserNotFound = errors.New("user not found")

// Store is the user store port: MemoryStore (no DATABASE_URL) or
// PGStore (phase-persistence.md decision 3).
type Store interface {
	Create(email, displayName, password, marketInterest string) (User, error)
	VerifyCredentials(email, password string) (User, error)
	ByID(id string) (User, error)
}

// record is a stored user plus its password hash. Not exported — the
// Service never hands raw records back across the package boundary.
type record struct {
	user         User
	passwordHash []byte
}

// MemoryStore is an in-memory, mutex-guarded user store, used when
// DATABASE_URL is unset (PGStore otherwise -- phase-persistence.md).
type MemoryStore struct {
	mu      sync.RWMutex
	byEmail map[string]*record
	byID    map[string]*record
	nextID  int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byEmail: make(map[string]*record),
		byID:    make(map[string]*record),
	}
}

func (s *MemoryStore) Create(email, displayName, password, marketInterest string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[email]; exists {
		return User{}, ErrEmailTaken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	s.nextID++
	user := User{ID: idOf(s.nextID), Email: email, DisplayName: displayName, MarketInterest: marketInterest}
	rec := &record{user: user, passwordHash: hash}
	s.byEmail[email] = rec
	s.byID[user.ID] = rec
	return user, nil
}

func (s *MemoryStore) VerifyCredentials(email, password string) (User, error) {
	s.mu.RLock()
	rec, ok := s.byEmail[email]
	s.mu.RUnlock()
	if !ok {
		return User{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword(rec.passwordHash, []byte(password)) != nil {
		return User{}, ErrInvalidCredentials
	}
	return rec.user, nil
}

func (s *MemoryStore) ByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.byID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return rec.user, nil
}

func idOf(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "u0"
	}
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{digits[n%10]}, buf...)
		n /= 10
	}
	return "u" + string(buf)
}
