package screener

import (
	"sort"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// SymbolPort is the small interface Service depends on -- satisfied by
// *symbol.Service's existing public methods, no new method added to
// symbol (phase-b.md decision 6).
type SymbolPort interface {
	Search(query, exchange string, page, pageSize int) ([]symbol.Symbol, int)
	Detail(sym string) (symbol.Detail, bool)
}

// MarketPort supplies the price/volume the top-movers table needs
// beyond what SymbolPort's Detail already has -- satisfied by
// *market.Service (phase-c.md decision 3).
type MarketPort interface {
	LatestQuote(sym string) (price float64, volume int64, ok bool)
}

type Service struct {
	symbols SymbolPort
	market  MarketPort
}

func NewService(symbols SymbolPort, market MarketPort) *Service {
	return &Service{symbols: symbols, market: market}
}

// universe enumerates every known symbol's live detail. pageSize 100 is
// comfortably larger than V1's small fixture universe (5 symbols); this
// will need real pagination once a licensed data vendor replaces the
// mock provider (see RESUME.md).
func (s *Service) universe() []symbol.Detail {
	all, _ := s.symbols.Search("", "", 1, 100)
	out := make([]symbol.Detail, 0, len(all))
	for _, sym := range all {
		if d, ok := s.symbols.Detail(sym.Symbol); ok {
			out = append(out, d)
		}
	}
	return out
}

func (s *Service) GetSectorHeatmap() []SectorGroup {
	bySector := make(map[string][]TickerChange)
	var order []string
	for _, d := range s.universe() {
		if _, ok := bySector[d.Sector]; !ok {
			order = append(order, d.Sector)
		}
		bySector[d.Sector] = append(bySector[d.Sector], TickerChange{Symbol: d.Symbol.Symbol, ChangePercent: round2(d.ChangePercent)})
	}

	out := make([]SectorGroup, 0, len(order))
	for _, sector := range order {
		tickers := bySector[sector]
		var sum float64
		for _, t := range tickers {
			sum += t.ChangePercent
		}
		out = append(out, SectorGroup{
			Sector:           sector,
			AvgChangePercent: round2(sum / float64(len(tickers))),
			Tickers:          tickers,
		})
	}
	return out
}

// GetTopMovers returns the top `limit` symbols by day change, sorted
// descending for direction "up" and ascending for direction "down".
// Price/Volume come from MarketPort -- a symbol missing quote data is
// still included with those fields left at zero rather than dropped, so
// one bad quote can't silently shrink the movers list.
func (s *Service) GetTopMovers(direction string, limit int) []TickerChange {
	universe := s.universe()
	out := make([]TickerChange, 0, len(universe))
	for _, d := range universe {
		tc := TickerChange{Symbol: d.Symbol.Symbol, ChangePercent: round2(d.ChangePercent)}
		if price, volume, ok := s.market.LatestQuote(d.Symbol.Symbol); ok {
			tc.Price = round2(price)
			tc.Volume = volume
		}
		out = append(out, tc)
	}

	sort.Slice(out, func(i, j int) bool {
		if direction == "down" {
			return out[i].ChangePercent < out[j].ChangePercent
		}
		return out[i].ChangePercent > out[j].ChangePercent
	})

	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}
