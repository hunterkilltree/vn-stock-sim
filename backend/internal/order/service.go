package order

import (
	"errors"
	"math"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var ErrSymbolNotFound = errors.New("symbol not found")
var ErrInsufficientFunds = portfolio.ErrInsufficientCash
var ErrInsufficientShares = portfolio.ErrInsufficientShares
var ErrNotFound = errors.New("order not found")

type Ledger interface {
	ApplyFill(portfolioID, sym, side string, quantity float64, price float64) error
}

type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

var ErrPortfolioNotFound = errors.New("portfolio not found")
var ErrWrongMarket = errors.New("this symbol belongs to the other market than this portfolio")
var ErrInvalidQuantity = errors.New("invalid quantity for this market")
var ErrOCOPrices = errors.New("an OCO order needs both a limit price and a stop price")
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
}

func NewService(store *MemoryStore, ledger Ledger, quotes QuotePort, portfolios PortfolioResolver) *Service {
	return &Service{store: store, ledger: ledger, quotes: quotes, portfolios: portfolios}
}

// Create places an order. V1 only fills "market" orders immediately
// against the current mock quote (paper trading); "limit"/"atc"/"stop"
// orders are accepted and stored as "queued" — matching against future
// price moves is not implemented yet (see RESUME.md future work).
func (s *Service) Create(userID string, req createRequest) (Order, error) {
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
		o.Status = "queued"
		o.Price = req.Price
		o.StopPrice = req.StopPrice
		return s.store.Append(userID, o), nil
	}

	if err := s.ledger.ApplyFill(portfolioID, detail.Symbol.Symbol, req.Side, req.Quantity, detail.LastPrice); err != nil {
		return Order{}, err
	}
	o.Status = "filled"
	o.FilledPrice = detail.LastPrice
	o.Fee = round2(detail.LastPrice * req.Quantity * feeRate)
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
// Only "filled" orders carry a real FilledPrice/FilledAt (queued
// limit/atc/stop orders never fill in V1, see Create above), so those
// are the only ones portfolio.Service's FIFO trade reconstruction needs.
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
