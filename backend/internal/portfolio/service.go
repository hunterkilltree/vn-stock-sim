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

func (s *Service) Summary(userID string) Summary {
	cash := s.store.Cash(userID)
	positions := s.valuedPositions(userID)

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

func (s *Service) Positions(userID string) []Position {
	return s.valuedPositions(userID)
}

func (s *Service) valuedPositions(userID string) []Position {
	raw := s.store.Positions(userID)
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
