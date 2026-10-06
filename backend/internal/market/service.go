package market

import (
	"context"
	"sync"
	"time"
)

// Service holds market-data rules (V1: none beyond delegation — caching,
// gap-filling, and session-hours clipping are noted as future work in
// RESUME.md, matching charting-library-integration.md section 7).
type Service struct {
	data   MarketDataProvider
	quotes *swrCache[quote]
}

// quote is the per-symbol snapshot behind LatestClose/LatestQuote.
type quote struct {
	today, yesterday float64
	volume           int64
	hasYesterday     bool
}

func NewService(data MarketDataProvider) *Service {
	return &Service{data: data, quotes: newSWRCache[quote](2000)}
}

// getQuote reads sym's latest daily snapshot from the shared quote table.
// The market, heatmap, movers and detail endpoints all read this one table,
// so a page load costs in-memory lookups, not one upstream fetch per symbol
// per endpoint. StartQuoteRefresher keeps it warm; on a cold or stale
// symbol this still self-heals (one coalesced load, or an instant stale
// read plus a background reload).
func (s *Service) getQuote(sym string) (quote, bool) {
	return s.quotes.get(sym, liveTTL, func() (quote, bool) { return s.loadQuote(sym) })
}

func (s *Service) loadQuote(sym string) (quote, bool) {
	to := time.Now().Unix()
	from := to - 10*24*60*60 // 10 days is comfortably more than 2 daily bars
	bars := s.data.GetBars(sym, "1D", from, to)
	if len(bars) == 0 {
		return quote{}, false
	}
	last := bars[len(bars)-1]
	q := quote{today: last.Close, volume: last.Volume}
	if len(bars) >= 2 {
		q.yesterday = bars[len(bars)-2].Close
		q.hasYesterday = true
	}
	return q, true
}

// StartQuoteRefresher warms the quote table for every symbol now and then
// re-warms it every interval until ctx is cancelled, with bounded
// concurrency, so request handlers find it filled.
func (s *Service) StartQuoteRefresher(ctx context.Context, symbols func() []string, interval time.Duration) {
	refresh := func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for _, sym := range symbols() {
			wg.Add(1)
			sem <- struct{}{}
			go func(sym string) {
				defer wg.Done()
				defer func() { <-sem }()
				s.quotes.reload(sym, liveTTL, func() (quote, bool) { return s.loadQuote(sym) })
			}(sym)
		}
		wg.Wait()
	}
	go func() {
		refresh()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				refresh()
			}
		}
	}()
}

func (s *Service) GetBars(sym, resolution string, from, to int64) []Bar {
	bars := s.data.GetBars(sym, resolution, from, to)
	// LiveProvider caches and returns its slice to every caller, so split
	// on a copy rather than mutating the cached bars.
	return withVolumeSplit(append([]Bar(nil), bars...))
}

func (s *Service) GetIndex(name string) IndexSnapshot {
	return s.data.GetIndex(name)
}

func (s *Service) GetOrderBook(sym string, lastPrice float64) (bids, asks []PriceLevel) {
	return s.data.GetOrderBook(sym, lastPrice)
}

// GetIndicator computes a basic SMA/EMA/RSI over the underlying bars. Only
// "sma" and "ema" are implemented for V1; others return an empty slice
// until the full indicator library lands (see RESUME.md).
func (s *Service) GetIndicator(sym, resolution, indicator string, period int, from, to int64) []IndicatorPoint {
	bars := s.data.GetBars(sym, resolution, from, to)
	switch indicator {
	case "sma":
		return sma(bars, period)
	case "ema":
		return ema(bars, period)
	case "rsi":
		return rsi(bars, period)
	default:
		return []IndicatorPoint{}
	}
}

func sma(bars []Bar, period int) []IndicatorPoint {
	if period <= 0 || len(bars) < period {
		return []IndicatorPoint{}
	}
	out := make([]IndicatorPoint, 0, len(bars)-period+1)
	var sum float64
	for i, b := range bars {
		sum += b.Close
		if i >= period {
			sum -= bars[i-period].Close
		}
		if i >= period-1 {
			out = append(out, IndicatorPoint{Time: b.Time, Value: round2(sum / float64(period))})
		}
	}
	return out
}

func ema(bars []Bar, period int) []IndicatorPoint {
	if period <= 0 || len(bars) < period {
		return []IndicatorPoint{}
	}
	k := 2.0 / float64(period+1)
	out := make([]IndicatorPoint, 0, len(bars)-period+1)
	var prev float64
	var sum float64
	for i, b := range bars {
		if i < period {
			sum += b.Close
			if i == period-1 {
				prev = sum / float64(period)
				out = append(out, IndicatorPoint{Time: b.Time, Value: round2(prev)})
			}
			continue
		}
		prev = b.Close*k + prev*(1-k)
		out = append(out, IndicatorPoint{Time: b.Time, Value: round2(prev)})
	}
	return out
}

// LatestClose returns the most recent two daily closes for sym, so
// callers (symbol.Service, for its quote fields) can derive a live
// last price/change instead of carrying a separately-seeded static
// value that can drift arbitrarily far from what the chart actually
// shows -- see phase-0-mvp.md for the bug this fixes. ok is false if fewer
// than two bars are available.
func (s *Service) LatestClose(sym string) (today, yesterday float64, ok bool) {
	q, ok := s.getQuote(sym)
	if !ok || !q.hasYesterday {
		return 0, 0, false
	}
	return q.today, q.yesterday, true
}

// LatestQuote returns the most recent bar's close price and volume --
// used by screener.Service to fill the price/volume columns of the
// Main screen's top-movers table (design/screens/Main.dc.html), which
// GetTopMovers's underlying symbol.Detail alone can't supply (Detail has
// a live price but no volume). ok is false if no bars are available.
func (s *Service) LatestQuote(sym string) (price float64, volume int64, ok bool) {
	q, ok := s.getQuote(sym)
	if !ok {
		return 0, 0, false
	}
	return q.today, q.volume, true
}
