package rating

import (
	"errors"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var ErrSymbolNotFound = errors.New("symbol not found")

// lookbackSeconds covers ~200 HOSE sessions (for SMA 200) with room for
// weekends and Tết.
const lookbackSeconds = 420 * 24 * 60 * 60

type SymbolPort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type MarketPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

// Service rates a symbol Strong Buy .. Strong Sell from fixed rules over
// its daily bars and fundamentals (rules.go) -- deterministic, no LLM,
// same reasoning as internal/insight.
type Service struct {
	symbols SymbolPort
	market  MarketPort
}

func NewService(symbols SymbolPort, market MarketPort) *Service {
	return &Service{symbols: symbols, market: market}
}

func (s *Service) Rate(sym string) (Result, error) {
	detail, ok := s.symbols.Detail(sym)
	if !ok {
		return Result{}, ErrSymbolNotFound
	}
	now := time.Now()
	bars := s.market.GetBars(detail.Symbol.Symbol, "1D", now.Unix()-lookbackSeconds, now.Unix())
	res := Compute(detail, bars)
	res.GeneratedAt = now.UTC().Format(time.RFC3339)
	return res, nil
}
