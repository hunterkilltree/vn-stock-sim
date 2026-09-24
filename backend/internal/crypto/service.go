package crypto

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var nowFunc = func() int64 { return time.Now().Unix() }

type Service struct {
	data *LiveProvider
}

func NewService(data *LiveProvider) *Service {
	return &Service{data: data}
}

func (s *Service) quote(p Pair) PairQuote {
	t := s.data.Ticker(p)
	return PairQuote{Pair: p, Ticker: t, MarketCap: t.LastPrice * p.CirculatingSupply}
}

// Pairs quotes the whole universe concurrently, in the design's order.
func (s *Service) Pairs() []PairQuote {
	out := make([]PairQuote, len(universe))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, p := range universe {
		wg.Add(1)
		go func(i int, p Pair) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = s.quote(p)
		}(i, p)
	}
	wg.Wait()
	return out
}

func (s *Service) Detail(raw string) (PairDetail, bool) {
	p, ok := Lookup(raw)
	if !ok {
		return PairDetail{}, false
	}
	d := PairDetail{PairQuote: s.quote(p), AllTimeHigh: p.seedATH}
	now := nowFunc()
	daily, _ := s.data.Klines(p, "1d", now-1000*86400, now)
	for _, b := range daily {
		d.AllTimeHigh = math.Max(d.AllTimeHigh, b.High)
	}
	d.AllTimeHigh = math.Max(d.AllTimeHigh, d.High24h)
	if d.AllTimeHigh > 0 {
		d.DistanceFromATHPercent = (d.LastPrice - d.AllTimeHigh) / d.AllTimeHigh * 100
	}
	d.Volatility30dPercentDay = dailyVolatility(daily, 30)
	return d, true
}

// dailyVolatility is the standard deviation of the last n daily returns,
// in % per day (Crypto-Detail's "Biến động 30 ngày").
func dailyVolatility(bars []market.Bar, n int) float64 {
	if len(bars) < 3 {
		return 0
	}
	if len(bars) > n+1 {
		bars = bars[len(bars)-n-1:]
	}
	var rets []float64
	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			rets = append(rets, (bars[i].Close-bars[i-1].Close)/bars[i-1].Close*100)
		}
	}
	var mean float64
	for _, r := range rets {
		mean += r
	}
	mean /= float64(len(rets))
	var v float64
	for _, r := range rets {
		v += (r - mean) * (r - mean)
	}
	return math.Sqrt(v / float64(len(rets)))
}

func (s *Service) Bars(raw, interval string, from, to int64) ([]market.Bar, string, bool) {
	p, ok := Lookup(raw)
	if !ok || !validInterval(interval) {
		return nil, "", false
	}
	bars, src := s.data.Klines(p, interval, from, to)
	return bars, src, true
}

// GetBars satisfies replay.BarsPort for crypto sessions: resolution "60"
// means 1-hour candles (phase-i.md decision 11).
func (s *Service) GetBars(raw, resolution string, from, to int64) []market.Bar {
	iv := map[string]string{"5": "5m", "15": "15m", "60": "1h", "240": "4h", "1D": "1d", "1W": "1w"}[resolution]
	if iv == "" {
		iv = "1h"
	}
	bars, _, _ := s.Bars(raw, iv, from, to)
	return bars
}

// OrderBook returns Crypto-Detail's 8-level book: the 4 best asks and 4
// best bids, each with a running total from the best price outward.
func (s *Service) OrderBook(raw string, levels int) (OrderBook, bool) {
	p, ok := Lookup(raw)
	if !ok {
		return OrderBook{}, false
	}
	bids, asks, src := s.data.Depth(p, 20)
	cum := func(in []DepthLevel) []DepthLevel {
		if len(in) > levels {
			in = in[:levels]
		}
		out := make([]DepthLevel, len(in))
		var run float64
		for i, lv := range in {
			run += lv.Size
			out[i] = DepthLevel{Price: lv.Price, Size: lv.Size, Cumulative: run}
		}
		return out
	}
	return OrderBook{Bids: cum(bids), Asks: cum(asks), Source: src}, true
}

// Heatmap groups the universe by category (24h change), design order.
func (s *Service) Heatmap() []HeatGroup {
	quotes := s.Pairs()
	groups := map[string]*HeatGroup{}
	for _, q := range quotes {
		g, ok := groups[q.Category]
		if !ok {
			g = &HeatGroup{Sector: q.Category, Tickers: []HeatTile{}}
			groups[q.Category] = g
		}
		pct := round2(q.ChangePercent)
		g.Tickers = append(g.Tickers, HeatTile{
			Symbol: q.Base, CompanyName: q.Name, Exchange: "CRYPTO", ChangePercent: pct,
			Price: q.LastPrice, Volume: int64(q.QuoteVolume24h), MarketCap: int64(q.MarketCap),
		})
		g.MarketCap += int64(q.MarketCap)
		// The design's crypto legend has a neutral band of +-0.15%.
		switch {
		case pct > 0.15:
			g.Up++
		case pct < -0.15:
			g.Down++
		default:
			g.Flat++
		}
	}
	out := make([]HeatGroup, 0, len(categoryOrder))
	for _, c := range categoryOrder {
		g := groups[c]
		if g == nil {
			continue
		}
		var sum float64
		for _, t := range g.Tickers {
			sum += t.ChangePercent
		}
		g.AvgChangePercent = round2(sum / float64(len(g.Tickers)))
		out = append(out, *g)
	}
	return out
}

func (s *Service) Movers(direction string, limit int) []PairQuote {
	quotes := s.Pairs()
	sort.Slice(quotes, func(i, j int) bool {
		if direction == "down" {
			return quotes[i].ChangePercent < quotes[j].ChangePercent
		}
		return quotes[i].ChangePercent > quotes[j].ChangePercent
	})
	if limit > 0 && limit < len(quotes) {
		quotes = quotes[:limit]
	}
	return quotes
}

func (s *Service) sparkline(p Pair) []float64 {
	now := nowFunc()
	bars, _ := s.data.Klines(p, "1h", now-24*3600, now)
	out := make([]float64, 0, len(bars))
	for _, b := range bars {
		out = append(out, b.Close)
	}
	return out
}

func (s *Service) Overview() Overview {
	quotes := s.Pairs()
	o := Overview{PairCount: len(quotes), CategoriesInOrder: categoryOrder}
	var prevTotal float64
	for _, q := range quotes {
		o.TotalMarketCap += q.MarketCap
		o.TotalQuoteVolume24h += q.QuoteVolume24h
		if q.ChangePercent > -100 {
			prevTotal += q.MarketCap / (1 + q.ChangePercent/100)
		}
		switch q.Base {
		case "BTC":
			o.BTC = q
		case "ETH":
			o.ETH = q
		}
	}
	if prevTotal > 0 {
		o.TotalMarketCapChangePct = round2((o.TotalMarketCap - prevTotal) / prevTotal * 100)
	}
	if o.TotalMarketCap > 0 {
		o.BTCDominancePercent = round2(o.BTC.MarketCap / o.TotalMarketCap * 100)
		o.ETHDominancePercent = round2(o.ETH.MarketCap / o.TotalMarketCap * 100)
	}
	o.BTCSparkline = s.sparkline(o.BTC.Pair)
	o.ETHSparkline = s.sparkline(o.ETH.Pair)
	return o
}

func IsPair(sym string) bool {
	_, ok := bySymbol[sym]
	return ok
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// StockQuotes is the stock symbol service's Detail.
type StockQuotes interface {
	Detail(sym string) (symbol.Detail, bool)
}

// QuoteRouter answers Detail for both markets: "…USDT" pairs from the
// crypto service, everything else from the stock symbol service. order
// and portfolio read prices through it, so their existing stacks serve
// crypto unchanged (phase-i.md decision 7).
type QuoteRouter struct {
	Crypto *Service
	Stock  StockQuotes
}

func (r QuoteRouter) Detail(sym string) (symbol.Detail, bool) {
	p, ok := bySymbol[sym]
	if !ok {
		return r.Stock.Detail(sym)
	}
	q := r.Crypto.quote(p)
	return symbol.Detail{
		Symbol:        symbol.Symbol{Symbol: p.Symbol, CompanyName: p.Name, Exchange: symbol.ExchangeCrypto, Sector: p.Category},
		LastPrice:     q.LastPrice,
		Change:        q.Change,
		ChangePercent: q.ChangePercent,
		MarketCap:     int64(q.MarketCap),
	}, true
}
