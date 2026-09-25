package order

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var ErrSymbolNotFound = errors.New("symbol not found")
var ErrInsufficientFunds = portfolio.ErrInsufficientCash
var ErrInsufficientShares = portfolio.ErrInsufficientShares
var ErrNotFound = errors.New("order not found")

type Ledger interface {
	// fee is charged to cash on top of the fill value (phase-k.md decision 1).
	ApplyFill(portfolioID, sym, side string, quantity, price, fee float64) error
}

type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

var ErrPortfolioNotFound = errors.New("portfolio not found")
var ErrWrongMarket = errors.New("this symbol belongs to the other market than this portfolio")
var ErrInvalidQuantity = errors.New("invalid quantity for this market")
var ErrOCOPrices = errors.New("an OCO order needs both a limit price and a stop price")
var ErrPriceRequired = errors.New("limit and stop orders need a price")
var ErrReplayPortfolio = errors.New("replay portfolios only accept fills from their own replay session")

// PortfolioResolver lets Service resolve a request's optional
// portfolioId to a real portfolio the caller owns -- satisfied by
// *portfolio.Service. See phase-b.md decision 1.
type PortfolioResolver interface {
	DefaultPortfolioID(userID string) string
	GetPortfolio(userID, id string) (portfolio.Portfolio, bool)
}

type Service struct {
	store      *MemoryStore
	ledger     Ledger
	quotes     QuotePort
	portfolios PortfolioResolver
	// mu serialises fills from the matcher with Create/Cancel, so a
	// cancelled order can't also fill (matcher.go).
	mu sync.Mutex
	// bars and checked are the matcher's bar source and per-order
	// watermark (last bar time examined); nil bars = matching off.
	bars    BarsPort
	checked map[string]int64
}

func NewService(store *MemoryStore, ledger Ledger, quotes QuotePort, portfolios PortfolioResolver) *Service {
	return &Service{store: store, ledger: ledger, quotes: quotes, portfolios: portfolios}
}

// Create places an order. "market" fills immediately at the current
// quote; limit/stop/OCO fill immediately too when they are already
// marketable at that quote (phase-k.md decision 6), otherwise they -- and
// every ATC order -- are stored as "queued" for the matcher (matcher.go).
func (s *Service) Create(userID string, req createRequest) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	detail, ok := s.quotes.Detail(req.Symbol)
	if !ok {
		return Order{}, ErrSymbolNotFound
	}
	portfolioID := req.PortfolioID
	if portfolioID == "" {
		portfolioID = s.portfolios.DefaultPortfolioID(userID)
	}
	pf, ok := s.portfolios.GetPortfolio(userID, portfolioID)
	if !ok {
		return Order{}, ErrPortfolioNotFound
	}
	if pf.Kind == portfolio.KindReplay {
		return Order{}, ErrReplayPortfolio
	}
	// A crypto wallet trades only crypto pairs, a stock portfolio only VN
	// tickers; stocks trade whole shares, crypto up to 8 decimals
	// (phase-i.md decisions 5 and 8).
	isCrypto := detail.Exchange == symbol.ExchangeCrypto
	if isCrypto != (pf.Market == "crypto") {
		return Order{}, ErrWrongMarket
	}
	feeRate := FeeRate
	if isCrypto {
		req.Quantity = math.Round(req.Quantity*1e8) / 1e8
		feeRate = CryptoFeeRate
	} else if req.Quantity != math.Trunc(req.Quantity) {
		return Order{}, ErrInvalidQuantity
	}
	if req.Quantity <= 0 {
		return Order{}, ErrInvalidQuantity
	}
	if req.Type == "oco" && (req.Price <= 0 || req.StopPrice <= 0) {
		return Order{}, ErrOCOPrices
	}
	if (req.Type == "limit" || req.Type == "stop") && req.Price <= 0 {
		return Order{}, ErrPriceRequired
	}
	now := time.Now()
	o := Order{
		PortfolioID: portfolioID,
		Symbol:      detail.Symbol.Symbol,
		Side:        req.Side,
		Type:        req.Type,
		Quantity:    req.Quantity,
		CreatedAt:   formatTime(now),
	}

	if req.Type != "market" {
		o.Price = req.Price
		o.StopPrice = req.StopPrice
		leg, now := triggeredNow(o, detail.LastPrice)
		if !now || req.Type == "atc" {
			o.Status = "queued"
			return s.store.Append(userID, o), nil
		}
		o.TriggeredBy = leg
	}

	fee := round2(detail.LastPrice * req.Quantity * feeRate)
	if err := s.ledger.ApplyFill(portfolioID, detail.Symbol.Symbol, req.Side, req.Quantity, detail.LastPrice, fee); err != nil {
		return Order{}, err
	}
	o.Status = "filled"
	o.FilledPrice = detail.LastPrice
	o.Fee = fee
	o.FilledAt = formatTime(now)
	return s.store.Append(userID, o), nil
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}

func (s *Service) List(userID string) []Order {
	return s.store.List(userID)
}

// FilledOrders satisfies portfolio.OrdersPort -- see phase-e.md item 2.
// Only "filled" orders carry a real FilledPrice/FilledAt (market orders,
// and queued ones once the matcher fills them), so those are the only
// ones portfolio.Service's FIFO trade reconstruction needs.
func (s *Service) FilledOrders(userID, portfolioID string) []portfolio.OrderRecord {
	all := s.store.List(userID)
	out := make([]portfolio.OrderRecord, 0, len(all))
	for _, o := range all {
		if o.PortfolioID != portfolioID || o.Status != "filled" {
			continue
		}
		out = append(out, portfolio.OrderRecord{
			Symbol:      o.Symbol,
			Side:        o.Side,
			Quantity:    o.Quantity,
			FilledPrice: o.FilledPrice,
			FilledAt:    o.FilledAt,
		})
	}
	return out
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
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.store.ByID(userID, id)
	if !ok {
		return Order{}, ErrNotFound
	}
	if o.Status != "queued" {
		return Order{}, errors.New("only queued orders can be cancelled")
	}
	o.Status = "cancelled"
	delete(s.checked, o.ID) // the matcher's watermark for it (nil-map safe)
	return s.store.Replace(userID, o), nil
}
