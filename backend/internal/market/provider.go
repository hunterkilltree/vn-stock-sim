package market

// MarketDataProvider is the port Service depends on, per CLAUDE.md's
// dependency-inversion rule: "the Service depends on a port, vendor
// specifics live in an adapter." V1 ships an in-memory deterministic
// generator; a licensed vendor (e.g. SSI) gets its own adapter later
// implementing the same interface.
type MarketDataProvider interface {
	GetBars(sym, resolution string, from, to int64) []Bar
}

type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

// GetBars synthesizes a deterministic daily-ish OHLCV series so the chart
// and indicator endpoints have real numbers to render without a live feed.
// Deterministic (seeded by symbol+time, not math/rand) so repeated calls
// for the same range return identical bars.
func (p *MockProvider) GetBars(sym, resolution string, from, to int64) []Bar {
	step := resolutionSeconds(resolution)
	if step <= 0 || from >= to {
		return []Bar{}
	}
	var bars []Bar
	price := 50000.0 + float64(seedFromSymbol(sym)%20000)
	for t := from; t <= to; t += step {
		drift := pseudoRandom(sym, t)
		open := price
		closePrice := open * (1 + drift)
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
		price = closePrice
	}
	return bars
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
