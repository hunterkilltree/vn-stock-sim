package order

import (
	"sort"
	"sync"
)

// Store is the order port: MemoryStore or PGStore
// (phase-persistence.md decision 3). Replay logs its fills through
// Append too.
type Store interface {
	Append(userID string, o Order) Order
	List(userID string) []Order
	ByID(userID, id string) (Order, bool)
	Queued() []queuedOrder
	Replace(userID string, updated Order) Order

	// Trade plans (phase-risk-rules.md slice 1). OpenPlan is the
	// portfolio's open plan for sym; UpdatePlan never changes InitialStop.
	AppendPlan(userID string, p TradePlan) TradePlan
	OpenPlan(portfolioID, sym string) (TradePlan, bool)
	UpdatePlan(p TradePlan)
	Plans(userID string) []TradePlan
}

// MemoryStore is an in-memory, per-user order history, used when
// DATABASE_URL is unset (PGStore otherwise -- phase-persistence.md).
type MemoryStore struct {
	mu     sync.Mutex
	byUser map[string][]Order
	nextID int
	plans  []planRow
}

type planRow struct {
	userID string
	plan   TradePlan
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byUser: make(map[string][]Order)}
}

func (s *MemoryStore) AppendPlan(userID string, p TradePlan) TradePlan {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = hexID("tp_", len(s.plans)+1)
	s.plans = append(s.plans, planRow{userID: userID, plan: p})
	return p
}

func (s *MemoryStore) OpenPlan(portfolioID, sym string) (TradePlan, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.plans {
		if r.plan.PortfolioID == portfolioID && r.plan.Symbol == sym && r.plan.ClosedAt == "" {
			return r.plan, true
		}
	}
	return TradePlan{}, false
}

func (s *MemoryStore) UpdatePlan(p TradePlan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.plans {
		if r.plan.ID == p.ID {
			p.InitialStop = r.plan.InitialStop
			s.plans[i].plan = p
			return
		}
	}
}

func (s *MemoryStore) Plans(userID string) []TradePlan {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []TradePlan{}
	for _, r := range s.plans {
		if r.userID == userID {
			out = append(out, r.plan)
		}
	}
	return out
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

func idOf(n int) string { return hexID("ord_", n) }

func hexID(prefix string, n int) string {
	const digits = "0123456789abcdef"
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{digits[n%16]}, buf...)
		n /= 16
	}
	if len(buf) == 0 {
		buf = []byte{'0'}
	}
	return prefix + string(buf)
}

type queuedOrder struct {
	userID string
	order  Order
}

// Queued lists every user's still-queued orders, oldest first -- the
// matcher's work list.
func (s *MemoryStore) Queued() []queuedOrder {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []queuedOrder
	for uid, list := range s.byUser {
		for _, o := range list {
			if o.Status == "queued" {
				out = append(out, queuedOrder{userID: uid, order: o})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].order.CreatedAt < out[j].order.CreatedAt })
	return out
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
