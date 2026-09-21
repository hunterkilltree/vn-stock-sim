package portfolio

import (
	"errors"
	"sync"
)

var ErrInsufficientCash = errors.New("insufficient virtual cash")
var ErrInsufficientShares = errors.New("insufficient shares")

type position struct {
	quantity int64
	avgCost  float64
}

type account struct {
	cash      float64
	positions map[string]*position
}

// MemoryStore is the in-memory paper-trading ledger: one account per user,
// lazily opened with StartingCash on first touch. No database wired up for
// V1 yet (see RESUME.md).
type MemoryStore struct {
	mu       sync.Mutex
	accounts map[string]*account
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{accounts: make(map[string]*account)}
}

func (s *MemoryStore) ensure(userID string) *account {
	acc, ok := s.accounts[userID]
	if !ok {
		acc = &account{cash: StartingCash, positions: make(map[string]*position)}
		s.accounts[userID] = acc
	}
	return acc
}

// ApplyFill books a filled market order against the user's paper account:
// buy debits cash and grows the position at a blended average cost, sell
// credits cash and shrinks the position. Returns the fill error (if any)
// without mutating state, so a rejected order never partially applies.
func (s *MemoryStore) ApplyFill(userID, sym, side string, quantity int64, price float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := s.ensure(userID)
	cost := price * float64(quantity)

	switch side {
	case "buy":
		if acc.cash < cost {
			return ErrInsufficientCash
		}
		pos, ok := acc.positions[sym]
		if !ok {
			pos = &position{}
			acc.positions[sym] = pos
		}
		totalCost := pos.avgCost*float64(pos.quantity) + cost
		pos.quantity += quantity
		pos.avgCost = totalCost / float64(pos.quantity)
		acc.cash -= cost
	case "sell":
		pos, ok := acc.positions[sym]
		if !ok || pos.quantity < quantity {
			return ErrInsufficientShares
		}
		pos.quantity -= quantity
		if pos.quantity == 0 {
			delete(acc.positions, sym)
		}
		acc.cash += cost
	}
	return nil
}

func (s *MemoryStore) Cash(userID string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensure(userID).cash
}

func (s *MemoryStore) Positions(userID string) map[string]position {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := s.ensure(userID)
	out := make(map[string]position, len(acc.positions))
	for sym, p := range acc.positions {
		out[sym] = *p
	}
	return out
}
