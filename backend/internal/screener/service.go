package screener

import (
	"sort"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
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
	GetBars(sym, resolution string, from, to int64) []market.Bar
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
// universe loads every symbol's detail (live price via the quote source)
// concurrently: with ~40 tickers and a live data source, doing it one at
// a time would make every Main/heatmap request wait on 40 round trips.
func (s *Service) universe(exchange string) []symbol.Detail {
	all, _ := s.symbols.Search("", exchange, 1, 1000)
	out := make([]symbol.Detail, len(all))
	ok := make([]bool, len(all))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, sym := range all {
		wg.Add(1)
		go func(i int, code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i], ok[i] = s.symbols.Detail(code)
		}(i, sym.Symbol)
	}
	wg.Wait()
	details := make([]symbol.Detail, 0, len(all))
	for i, d := range out {
		if ok[i] {
			details = append(details, d)
		}
	}
	return details
}

// periodChange is the % change over `sessions` daily closes, from real
// bars; 0 when there isn't enough history.
func (s *Service) periodChange(sym string, sessions int) float64 {
	now := time.Now().Unix()
	from := now - int64(sessions*2+15)*24*60*60
	bars := s.market.GetBars(sym, "1D", from, now)
	if len(bars) <= sessions {
		return 0
	}
	last, base := bars[len(bars)-1].Close, bars[len(bars)-1-sessions].Close
	if base == 0 {
		return 0
	}
	return (last - base) / base * 100
}

// GetSectorHeatmap groups the universe by sector for Main's heatmap panel
// and the full /heatmap dashboard. period is one of PeriodSessions' keys;
// exchange "" means all exchanges. Sectors and tiles are ordered by
// market cap, largest first.
func (s *Service) GetSectorHeatmap(period, exchange string) []SectorGroup {
	sessions, known := PeriodSessions[period]
	if !known {
		sessions = 1
	}
	details := s.universe(exchange)
	changes := make([]float64, len(details))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, d := range details {
		if sessions == 1 {
			changes[i] = d.ChangePercent
			continue
		}
		wg.Add(1)
		go func(i int, code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			changes[i] = s.periodChange(code, sessions)
		}(i, d.Symbol.Symbol)
	}
	wg.Wait()

	groups := map[string]*SectorGroup{}
	for i, d := range details {
		g, ok := groups[d.Sector]
		if !ok {
			g = &SectorGroup{Sector: d.Sector, Tickers: []TickerChange{}}
			groups[d.Sector] = g
		}
		pct := round2(changes[i])
		tc := TickerChange{
			Symbol: d.Symbol.Symbol, ChangePercent: pct, Price: round2(d.LastPrice),
			CompanyName: d.CompanyName, Exchange: d.Exchange, MarketCap: d.MarketCap,
		}
		if _, vol, ok := s.market.LatestQuote(d.Symbol.Symbol); ok {
			tc.Volume = vol
		}
		g.Tickers = append(g.Tickers, tc)
		g.MarketCap += d.MarketCap
		switch {
		case pct > 0:
			g.Up++
		case pct < 0:
			g.Down++
		default:
			g.Flat++
		}
	}

	out := make([]SectorGroup, 0, len(groups))
	for _, g := range groups {
		var sum float64
		for _, t := range g.Tickers {
			sum += t.ChangePercent
		}
		g.AvgChangePercent = round2(sum / float64(len(g.Tickers)))
		sort.SliceStable(g.Tickers, func(i, j int) bool { return g.Tickers[i].MarketCap > g.Tickers[j].MarketCap })
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MarketCap > out[j].MarketCap })
	return out
}

func (s *Service) GetTopMovers(direction string, limit int) []TickerChange {
	universe := s.universe("")
	out := make([]TickerChange, 0, len(universe))
	for _, d := range universe {
		// Name and exchange feed the phone movers list (Mobile-Market.dc.html).
		tc := TickerChange{Symbol: d.Symbol.Symbol, ChangePercent: round2(d.ChangePercent), CompanyName: d.CompanyName, Exchange: d.Exchange}
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
