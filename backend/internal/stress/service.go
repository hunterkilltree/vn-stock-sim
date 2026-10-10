package stress

import (
	"errors"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var (
	ErrNotFound  = errors.New("portfolio not found")
	ErrNotStocks = errors.New("stress test covers stock portfolios only")
)

// lookback: as far back as VCI's 2000-bar countBack reaches (~8 years),
// to find as many real VN-Index falls as the data holds.
const lookback = 8 * 365 * 24 * 60 * 60

const indexName = "VN-Index"

type PortfolioPort interface {
	GetPortfolio(userID, id string) (portfolio.Portfolio, bool)
	Positions(portfolioID string) []portfolio.Position
	Summary(portfolioID string) portfolio.Summary
}

type SymbolPort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type MarketPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

type Service struct {
	portfolios PortfolioPort
	symbols    SymbolPort
	market     MarketPort
}

func NewService(portfolios PortfolioPort, symbols SymbolPort, market MarketPort) *Service {
	return &Service{portfolios: portfolios, symbols: symbols, market: market}
}

func (s *Service) Run(userID, portfolioID string) (Result, error) {
	p, ok := s.portfolios.GetPortfolio(userID, portfolioID)
	if !ok {
		return Result{}, ErrNotFound
	}
	if p.Market == "crypto" {
		return Result{}, ErrNotStocks
	}
	now := time.Now().Unix()
	from := now - lookback
	in := Input{Cash: s.portfolios.Summary(p.ID).CashBalance, Index: s.market.GetBars(indexName, "1D", from, now)}
	for _, pos := range s.portfolios.Positions(p.ID) {
		band := 0.07
		if d, ok := s.symbols.Detail(pos.Symbol); ok {
			band = bandFor(d.Exchange)
		}
		in.Holdings = append(in.Holdings, Holding{
			Symbol: pos.Symbol, Value: pos.MarketValue, Band: band,
			Bars: s.market.GetBars(pos.Symbol, "1D", from, now),
		})
	}
	return Compute(in), nil
}

// bandFor mirrors symbol.priceBandPercent (unexported there).
func bandFor(exchange string) float64 {
	switch exchange {
	case "HNX":
		return 0.10
	case "UPCOM":
		return 0.15
	default:
		return 0.07
	}
}
