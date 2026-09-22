package portfolio

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"

type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type Service struct {
	store  *MemoryStore
	quotes QuotePort
}

func NewService(store *MemoryStore, quotes QuotePort) *Service {
	return &Service{store: store, quotes: quotes}
}

// DefaultPortfolioID returns userID's lazily-created default portfolio,
// so callers that don't yet pick a specific portfolio (pre-Phase-B
// frontend, order requests with no portfolioId) keep working -- see
// phase-b.md decision 1.
func (s *Service) DefaultPortfolioID(userID string) string {
	return s.store.DefaultFor(userID)
}

func (s *Service) CreatePortfolio(userID, name, market string, startingCapital float64, currency string) Portfolio {
	if name == "" {
		name = "Danh muc moi"
	}
	if market == "" {
		market = "stock"
	}
	if currency == "" {
		currency = "VND"
	}
	if startingCapital <= 0 {
		startingCapital = StartingCash
	}
	return s.store.Create(userID, name, market, startingCapital, currency)
}

func (s *Service) ListPortfolios(userID string) []Portfolio {
	// DefaultFor ensures every user has at least one portfolio to list,
	// even if they've never explicitly created one.
	s.store.DefaultFor(userID)
	return s.store.List(userID)
}

func (s *Service) GetPortfolio(userID, id string) (Portfolio, bool) {
	return s.store.Get(userID, id)
}

func (s *Service) Summary(portfolioID string) Summary {
	cash := s.store.Cash(portfolioID)
	positions := s.valuedPositions(portfolioID)

	var marketValue, costBasis float64
	for _, p := range positions {
		marketValue += p.MarketValue
		costBasis += p.AvgCost * float64(p.Quantity)
	}
	unrealized := marketValue - costBasis
	var unrealizedPct float64
	if costBasis > 0 {
		unrealizedPct = unrealized / costBasis * 100
	}
	return Summary{
		CashBalance:          round2(cash),
		MarketValue:          round2(marketValue),
		TotalEquity:          round2(cash + marketValue),
		UnrealizedPnl:        round2(unrealized),
		UnrealizedPnlPercent: round2(unrealizedPct),
	}
}

func (s *Service) Positions(portfolioID string) []Position {
	return s.valuedPositions(portfolioID)
}

func (s *Service) valuedPositions(portfolioID string) []Position {
	raw := s.store.Positions(portfolioID)
	out := make([]Position, 0, len(raw))
	for sym, p := range raw {
		lastPrice := p.avgCost
		if d, ok := s.quotes.Detail(sym); ok {
			lastPrice = d.LastPrice
		}
		marketValue := lastPrice * float64(p.quantity)
		out = append(out, Position{
			Symbol:        sym,
			Quantity:      p.quantity,
			AvgCost:       round2(p.avgCost),
			LastPrice:     round2(lastPrice),
			MarketValue:   round2(marketValue),
			UnrealizedPnl: round2(marketValue - p.avgCost*float64(p.quantity)),
		})
	}
	return out
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}
