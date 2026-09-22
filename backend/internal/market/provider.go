package market

import "math"

// MarketDataProvider is the port Service depends on, per CLAUDE.md's
// dependency-inversion rule: "the Service depends on a port, vendor
// specifics live in an adapter." V1 ships an in-memory deterministic
// generator; a licensed vendor (e.g. SSI) gets its own adapter later
// implementing the same interface.
type MarketDataProvider interface {
	GetBars(sym, resolution string, from, to int64) []Bar
	GetIndex(name string) IndexSnapshot
	GetOrderBook(sym string, lastPrice float64) (bids, asks []PriceLevel)
}

type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

// GetBars synthesizes a deterministic daily-ish OHLCV series so the chart
// and indicator endpoints have real numbers to render without a live feed.
//
// closeFor(sym, t) is a pure function of (symbol, time) -- not an
// iterative walk accumulated from whatever "from" the caller happened to
// pass. That matters: an earlier version computed each bar as a running
// product starting from a fixed seed price at t=from, so the "current"
// price at a given moment depended on how far back the request's window
// started (e.g. GET /market/bars for the last 180 days vs. the last 10
// days would report two different closes for today, and the stock detail
// page's lastPrice -- computed over a short window -- would silently
// disagree with what the chart -- computed over a long window -- showed
// for the same instant). Every bar's open/close here is now anchored to
// its own absolute timestamp, so any window ending at the same instant
// reports the same close for that instant, regardless of how far back it
// starts.
func (p *MockProvider) GetBars(sym, resolution string, from, to int64) []Bar {
	step := resolutionSeconds(resolution)
	if step <= 0 || from >= to {
		return []Bar{}
	}
	// Align to the resolution's grid (multiples of step since the Unix
	// epoch) so requests with a slightly different exact from/to -- e.g.
	// two calls a few seconds apart, each computing "now" independently --
	// still land on the same bar timestamps.
	from -= from % step
	to -= to % step

	var bars []Bar
	for t := from; t <= to; t += step {
		open := closeFor(sym, t-step)
		closePrice := closeFor(sym, t)
		high := max2(open, closePrice) * (1 + 0.004)
		low := min2(open, closePrice) * (1 - 0.004)
		volume := int64(500000 + (seedFromSymbol(sym)+t)%1500000)
		bars = append(bars, Bar{
			Time:   t,
			Open:   round2(open),
			High:   round2(high),
			Low:    round2(low),
			Close:  round2(closePrice),
			Volume: volume,
		})
	}
	return bars
}

// closeFor computes a deterministic, smooth-ish price purely from
// (symbol, absolute time) -- no iteration, no dependency on any
// caller-supplied window. Two sine waves (symbol-seeded periods/phases)
// give slow wavy motion around a base price, plus a small pseudo-random
// jitter per bar so consecutive bars are not perfectly smooth.
func closeFor(sym string, t int64) float64 {
	seed := seedFromSymbol(sym)
	base := 50000.0 + float64(seed%20000)

	days := float64(t) / 86400.0
	period1 := 60.0 + float64(seed%40)      // ~60-100 day cycle
	period2 := 15.0 + float64((seed/7)%20)  // ~15-35 day cycle
	phase1 := float64(seed%628) / 100.0     // 0-2π-ish
	phase2 := float64((seed/13)%628) / 100.0

	wave := 0.15*math.Sin(2*math.Pi*days/period1+phase1) +
		0.06*math.Sin(2*math.Pi*days/period2+phase2)
	jitter := pseudoRandom(sym, t) * 0.3

	return base * (1 + wave + jitter)
}

func resolutionSeconds(resolution string) int64 {
	switch resolution {
	case "1":
		return 60
	case "5":
		return 5 * 60
	case "15":
		return 15 * 60
	case "60":
		return 60 * 60
	case "1D":
		return 24 * 60 * 60
	default:
		return 24 * 60 * 60
	}
}

func seedFromSymbol(sym string) int64 {
	var h int64
	for _, r := range sym {
		h = h*31 + int64(r)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// pseudoRandom returns a deterministic value in [-0.02, 0.02) derived from
// symbol+time — a simple LCG, not math/rand, so results are reproducible.
func pseudoRandom(sym string, t int64) float64 {
	seed := seedFromSymbol(sym) ^ t
	seed = (seed*1103515245 + 12345) & 0x7fffffff
	return (float64(seed%4000)/4000.0 - 0.5) * 0.04
}

func max2(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min2(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}
