package crypto

import (
	"errors"
	"hash/fnv"
	"math"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// ErrNoData means the source answered but has nothing for this pair
// (e.g. Binance doesn't list it). Unlike a connection failure it says
// nothing about whether the source is reachable.
var ErrNoData = errors.New("crypto: no data for this pair")

// DataProvider is the port the service reads market data through --
// Binance (binance.go) or the mock generator below, combined by
// LiveProvider (live.go). Same shape as market.MarketDataProvider.
type DataProvider interface {
	Ticker(p Pair) (Ticker, error)
	// Klines returns bars with open times in [from, to] (unix seconds).
	Klines(p Pair, interval string, from, to int64) ([]market.Bar, error)
	Depth(p Pair, limit int) (bids, asks []DepthLevel, err error)
}

// Intervals the app uses, in seconds (Crypto-Detail's 5 phút ... 1 tuần).
var intervalSeconds = map[string]int64{
	"5m": 300, "15m": 900, "1h": 3600, "4h": 14400, "1d": 86400, "1w": 604800,
}

func validInterval(iv string) bool {
	_, ok := intervalSeconds[iv]
	return ok
}

// MockProvider is the deterministic 24/7 generator used when Binance is
// unreachable or doesn't list a pair: price is a pure function of (pair,
// time) -- two pair-seeded sine waves plus jitter around the pair's seed
// price -- so any window ending at the same instant agrees on "now", and
// there are no session or weekend gaps.
type MockProvider struct{}

func seedOf(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func mockPrice(pr Pair, t int64) float64 {
	seed := seedOf(pr.Symbol)
	days := float64(t) / 86400
	p1 := 45 + float64(seed%40)
	p2 := 6 + float64((seed/7)%9)
	ph1 := float64(seed%628) / 100
	ph2 := float64((seed/13)%628) / 100
	hourJitter := math.Sin(float64(t/3600)*12.9898+float64(seed%97)) * 0.004
	w := 1 + 0.16*math.Sin(2*math.Pi*days/p1+ph1) + 0.05*math.Sin(2*math.Pi*days/p2+ph2) + hourJitter
	return pr.seedPrice * w
}

func (MockProvider) Klines(pr Pair, interval string, from, to int64) ([]market.Bar, error) {
	step, ok := intervalSeconds[interval]
	if !ok {
		return nil, ErrNoData
	}
	start := (from / step) * step
	if start < from {
		start += step
	}
	seed := int64(seedOf(pr.Symbol) % 1000)
	var out []market.Bar
	for t := start; t <= to && len(out) < 5000; t += step {
		open, close := mockPrice(pr, t), mockPrice(pr, t+step)
		hi, lo := math.Max(open, close), math.Min(open, close)
		wiggle := math.Abs(math.Sin(float64(t)*0.37)) * 0.003
		volBase := 1e6 / math.Max(pr.seedPrice, 1e-6) * float64(step) / 3600
		out = append(out, market.Bar{
			Time:   t,
			Open:   open,
			High:   hi * (1 + wiggle),
			Low:    lo * (1 - wiggle),
			Close:  close,
			Volume: int64(volBase * (0.6 + float64((t/step+seed)%70)/100)),
		})
	}
	return out, nil
}

func (m MockProvider) Ticker(pr Pair) (Ticker, error) {
	return tickerFromHourly(pr, m, nowFunc())
}

// tickerFromHourly derives a 24h ticker from the last 24 hourly bars.
func tickerFromHourly(pr Pair, src DataProvider, now int64) (Ticker, error) {
	bars, err := src.Klines(pr, "1h", now-25*3600, now)
	if err != nil || len(bars) < 2 {
		return Ticker{}, ErrNoData
	}
	if len(bars) > 24 {
		bars = bars[len(bars)-24:]
	}
	first, last := bars[0], bars[len(bars)-1]
	t := Ticker{LastPrice: last.Close, High24h: first.High, Low24h: first.Low, Source: "mock"}
	for _, b := range bars {
		t.High24h = math.Max(t.High24h, b.High)
		t.Low24h = math.Min(t.Low24h, b.Low)
		t.Volume24h += float64(b.Volume)
		t.QuoteVolume24h += float64(b.Volume) * b.Close
	}
	t.Change = last.Close - first.Open
	if first.Open > 0 {
		t.ChangePercent = t.Change / first.Open * 100
	}
	return t, nil
}

func (MockProvider) Depth(pr Pair, limit int) ([]DepthLevel, []DepthLevel, error) {
	mid := mockPrice(pr, nowFunc())
	tick := priceTick(mid)
	seed := int64(seedOf(pr.Symbol))
	now := nowFunc()
	var bids, asks []DepthLevel
	for i := 0; i < limit; i++ {
		size := func(side int64) float64 {
			v := float64((seed+now/5+int64(i)*31+side*17)%900+100) / 100 // 1.00-9.99
			return v * 20_000 / math.Max(mid, 1e-9) / 10
		}
		bids = append(bids, DepthLevel{Price: roundTo(mid-tick*float64(i+1), tick), Size: size(1)})
		asks = append(asks, DepthLevel{Price: roundTo(mid+tick*float64(i+1), tick), Size: size(2)})
	}
	return bids, asks, nil
}

// priceTick is a sensible order-book step for a price's magnitude
// (0.10 USDT for BTC, like the design's "Gộp 0,10 USDT").
func priceTick(price float64) float64 {
	if price <= 0 {
		return 0.00000001
	}
	exp := math.Floor(math.Log10(price))
	return math.Pow(10, exp-5)
}

func roundTo(v, step float64) float64 {
	return math.Round(v/step) * step
}
