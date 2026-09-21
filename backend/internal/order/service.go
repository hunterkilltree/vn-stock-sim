package order

import (
	"errors"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var ErrSymbolNotFound = errors.New("symbol not found")
var ErrInsufficientFunds = portfolio.ErrInsufficientCash
var ErrInsufficientShares = portfolio.ErrInsufficientShares
var ErrNotFound = errors.New("order not found")

type Ledger interface {
	ApplyFill(userID, sym, side string, quantity int64, price float64) error
}

type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type Service struct {
	store   *MemoryStore
	ledger  Ledger
	quotes  QuotePort
}

func NewService(store *MemoryStore, ledger Ledger, quotes QuotePort) *Service {
	return &Service{store: store, ledger: ledger, quotes: quotes}
}

// Create places an order. V1 only fills "market" orders immediately
// against the current mock quote (paper trading); "limit" orders are
// accepted and stored as "queued" — matching against future price moves
// is not implemented yet (see RESUME.md future work).
func (s *Service) Create(userID string, req createRequest) (Order, error) {
	detail, ok := s.quotes.Detail(req.Symbol)
	if !ok {
		return Order{}, ErrSymbolNotFound
	}
	now := time.Now()
	o := Order{
		Symbol:    detail.Symbol.Symbol,
		Side:      req.Side,
		Type:      req.Type,
		Quantity:  req.Quantity,
		CreatedAt: formatTime(now),
	}

	if req.Type == "limit" {
		o.Status = "queued"
		return s.store.Append(userID, o), nil
	}

	if err := s.ledger.ApplyFill(userID, detail.Symbol.Symbol, req.Side, req.Quantity, detail.LastPrice); err != nil {
		return Order{}, err
	}
	o.Status = "filled"
	o.FilledPrice = detail.LastPrice
	o.FilledAt = formatTime(now)
	return s.store.Append(userID, o), nil
}

func (s *Service) List(userID string) []Order {
	return s.store.List(userID)
}

func (s *Service) Get(userID, id string) (Order, error) {
	o, ok := s.store.ByID(userID, id)
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

// Cancel marks a still-queued (limit) order as cancelled. Filled market
// orders cannot be cancelled.
func (s *Service) Cancel(userID, id string) (Order, error) {
	o, ok := s.store.ByID(userID, id)
	if !ok {
		return Order{}, ErrNotFound
	}
	if o.Status != "queued" {
		return Order{}, errors.New("only queued orders can be cancelled")
	}
	o.Status = "cancelled"
	return s.store.Replace(userID, o), nil
}
