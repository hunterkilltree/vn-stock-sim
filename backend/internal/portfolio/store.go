package portfolio

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrInsufficientCash = errors.New("insufficient virtual cash")
var ErrInsufficientShares = errors.New("insufficient shares")
var ErrNotFound = errors.New("portfolio not found")

type position struct {
	quantity int64
	avgCost  float64
}

type account struct {
	cash      float64
	positions map[string]*position
}

// MemoryStore is the in-memory paper-trading ledger. Reworked in Phase B
// (see phase-b.md decision 1) from one account per user to one account
// per Portfolio: a user may own several portfolios, each with its own
// cash balance and position set, keyed by portfolio ID. byUser indexes a
// user's portfolio IDs for listing; byUserDefault remembers each user's
// lazily-created default portfolio so the pre-Phase-B single-portfolio
// callers (GET /portfolio, /portfolio/positions, orders with no
// portfolioId) keep working unchanged. No database wired up for V1 yet
// (see RESUME.md).
type MemoryStore struct {
	mu            sync.Mutex
	portfolios    map[string]*Portfolio
	accounts      map[string]*account
	byUser        map[string][]string
	byUserDefault map[string]string
	// equity indexes each portfolio's EquityPoint history -- see
	// AppendEquitySnapshot and phase-e.md item 1.
	equity map[string][]EquityPoint
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		portfolios:    make(map[string]*Portfolio),
		accounts:      make(map[string]*account),
		byUser:        make(map[string][]string),
		byUserDefault: make(map[string]string),
		equity:        make(map[string][]EquityPoint),
	}
}

func (s *MemoryStore) nextPortfolioID() string {
	s.nextID++
	return fmt.Sprintf("pf_%d", s.nextID)
}

// Create opens a new named portfolio for userID with its own cash
// balance seeded from startingCapital. The first KindTrading portfolio
// a user gets becomes their default (phase-g.md decision 5) -- that is
// how Signup's capital picker sets the main portfolio's capital.
func (s *MemoryStore) Create(userID, name, market string, startingCapital float64, currency, kind string) Portfolio {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createLocked(userID, name, market, startingCapital, currency, kind)
}

func (s *MemoryStore) createLocked(userID, name, market string, startingCapital float64, currency, kind string) Portfolio {
	p := Portfolio{
		ID:              s.nextPortfolioID(),
		UserID:          userID,
		Name:            name,
		Market:          market,
		Kind:            kind,
		StartingCapital: startingCapital,
		Currency:        currency,
		CreatedAt:       time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	s.portfolios[p.ID] = &p
	s.accounts[p.ID] = &account{cash: startingCapital, positions: make(map[string]*position)}
	s.byUser[userID] = append(s.byUser[userID], p.ID)
	if _, ok := s.byUserDefault[userID]; !ok && kind == KindTrading {
		s.byUserDefault[userID] = p.ID
	}
	// Seed one equity point at creation so a brand-new portfolio's
	// history is never empty (its NAV is just its starting capital).
	s.equity[p.ID] = []EquityPoint{{Timestamp: p.CreatedAt, NAV: startingCapital}}
	return p
}

// AppendEquitySnapshot records a portfolio's current NAV at "now" --
// called by Service.ApplyFill after every successful fill (see
// service.go), not on a schedule, per phase-e.md item 1's simplest-for-V1
// approach.
func (s *MemoryStore) AppendEquitySnapshot(portfolioID string, nav float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.equity[portfolioID] = append(s.equity[portfolioID], EquityPoint{
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		NAV:       nav,
	})
}

func (s *MemoryStore) EquityHistory(portfolioID string) []EquityPoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]EquityPoint, len(s.equity[portfolioID]))
	copy(out, s.equity[portfolioID])
	return out
}

// OwnerOf returns the userID that owns portfolioID, without requiring
// the caller to already know it -- Service.Stats/Positions need this to
// call OrdersPort.FilledOrders(userID, ...), since order.MemoryStore is
// keyed by userID, not portfolioID. Callers only reach here after a
// handler has already authorized the request via GetPortfolio(userID,
// id), so this doesn't itself enforce ownership.
func (s *MemoryStore) OwnerOf(portfolioID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.portfolios[portfolioID]
	if !ok {
		return "", false
	}
	return p.UserID, true
}

// DefaultFor returns the lazily-created default portfolio ID for userID,
// opening a "Danh mục chính" (Main portfolio) stock portfolio seeded with
// StartingCash the first time it's asked for a given user -- the same
// lazy-open behavior the old single-portfolio ensure() had, now sitting
// on top of the multi-portfolio store.
func (s *MemoryStore) DefaultFor(userID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.byUserDefault[userID]; ok {
		return id
	}
	return s.createLocked(userID, "Danh mục chính", "stock", StartingCash, "VND", KindTrading).ID
}

func (s *MemoryStore) List(userID string) []Portfolio {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.byUser[userID]
	out := make([]Portfolio, 0, len(ids))
	for _, id := range ids {
		out = append(out, *s.portfolios[id])
	}
	return out
}

// Get returns the portfolio only if it belongs to userID, so callers
// can't read/act on another user's portfolio by guessing an ID.
func (s *MemoryStore) Get(userID, id string) (Portfolio, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.portfolios[id]
	if !ok || p.UserID != userID {
		return Portfolio{}, false
	}
	return *p, true
}

func (s *MemoryStore) accountFor(portfolioID string) *account {
	acc, ok := s.accounts[portfolioID]
	if !ok {
		acc = &account{positions: make(map[string]*position)}
		s.accounts[portfolioID] = acc
	}
	return acc
}

// ApplyFill books a filled order against a portfolio's ledger: buy debits
// cash and grows the position at a blended average cost, sell credits
// cash and shrinks the position. Returns the fill error (if any) without
// mutating state, so a rejected order never partially applies.
func (s *MemoryStore) ApplyFill(portfolioID, sym, side string, quantity int64, price float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := s.accountFor(portfolioID)
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

func (s *MemoryStore) Cash(portfolioID string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accountFor(portfolioID).cash
}

func (s *MemoryStore) Positions(portfolioID string) map[string]position {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := s.accountFor(portfolioID)
	out := make(map[string]position, len(acc.positions))
	for sym, p := range acc.positions {
		out[sym] = *p
	}
	return out
}
