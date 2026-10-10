package regime

import (
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// indexName is VN-Index as market.GetBars knows it (VCI maps it to
// VNINDEX; the mock generates a series).
const indexName = "VN-Index"

// lookback covers SMA 200 + its 20-session slope + a year of rolling
// volatility, with room for weekends and holidays.
const lookback = 2 * 365 * 24 * 60 * 60

// cacheTTL: the regime is a daily read; recomputing it means one bars
// fetch per universe symbol, so don't do it on every page view.
const cacheTTL = 10 * time.Minute

type SymbolPort interface {
	Search(query, exchange string, page, pageSize int) ([]symbol.Symbol, int)
}

type MarketPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

type Service struct {
	symbols SymbolPort
	market  MarketPort

	mu     sync.Mutex
	cached *Regime
	at     time.Time
}

func NewService(symbols SymbolPort, market MarketPort) *Service {
	return &Service{symbols: symbols, market: market}
}

// Current returns the regime, ok=false if VN-Index history is too short.
func (s *Service) Current() (Regime, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil && time.Since(s.at) < cacheTTL {
		return *s.cached, true
	}
	now := time.Now()
	from, to := now.Unix()-lookback, now.Unix()
	in := Input{IndexName: indexName, Index: s.market.GetBars(indexName, "1D", from, to), Universe: map[string][]market.Bar{}}
	all, _ := s.symbols.Search("", "", 1, 10_000)
	for _, sym := range all {
		if sym.Exchange == symbol.ExchangeCrypto {
			continue
		}
		in.Universe[sym.Symbol] = s.market.GetBars(sym.Symbol, "1D", from, to)
	}
	r, ok := Compute(in)
	if !ok {
		return Regime{}, false
	}
	r.GeneratedAt = now.UTC().Format(time.RFC3339)
	s.cached, s.at = &r, now
	return r, true
}
