package regime

import (
	"math"
	"strings"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

const day = 24 * 60 * 60

// path builds daily bars from per-day returns; noise alternates so
// volatility is controllable.
func path(n int, start float64, ret func(i int) float64, noise float64) []market.Bar {
	bars := make([]market.Bar, n)
	c := start
	for i := range bars {
		c *= 1 + ret(i)
		v := c
		if i%2 == 0 {
			v *= 1 + noise
		}
		bars[i] = market.Bar{Time: int64(i) * day, Open: v, High: v, Low: v, Close: v}
	}
	return bars
}

func constant(r float64) func(int) float64 { return func(int) float64 { return r } }

func universe(n int, mk func(k int) []market.Bar) map[string][]market.Bar {
	u := map[string][]market.Bar{}
	for k := 0; k < n; k++ {
		u[string(rune('A'+k))] = mk(k)
	}
	return u
}

func signal(t *testing.T, r Regime, key string) Signal {
	t.Helper()
	for _, s := range r.Signals {
		if s.Key == key {
			return s
		}
	}
	t.Fatalf("no signal %q", key)
	return Signal{}
}

func TestSteadyUptrendIsNormal(t *testing.T) {
	up := func(int) []market.Bar { return path(500, 100, constant(0.0005), 0.004) }
	r, ok := Compute(Input{Index: up(0), Universe: universe(20, up)})
	if !ok || r.Level != LevelNormal || r.Score != 0 {
		t.Fatalf("got %s score %d: %+v", r.Level, r.Score, r.Signals)
	}
	if s := signal(t, r, "breadth"); s.Zone != ZoneOK || !strings.Contains(s.Value, "100%") {
		t.Errorf("breadth = %+v", s)
	}
}

func TestCrashIsHighRisk(t *testing.T) {
	// 400 calm sessions up, then 100 sessions falling ~0.4%/day with
	// violent swings: below a falling SMA 200, >20% off the high, volatility
	// several times normal, and most stocks below their own SMA 200.
	ret := func(i int) float64 {
		if i < 400 {
			return 0.0005
		}
		return -0.004
	}
	mk := func(k int) []market.Bar {
		bars := path(500, 100, ret, 0.002)
		for i := 400; i < len(bars); i++ {
			if i%2 == 0 {
				bars[i].Close *= 1.03
			}
		}
		return bars
	}
	r, ok := Compute(Input{Index: mk(0), Universe: universe(20, mk)})
	if !ok {
		t.Fatal("not ok")
	}
	for _, key := range []string{"trend", "drawdown", "volatility", "breadth"} {
		if s := signal(t, r, key); s.Zone != ZoneRisk {
			t.Errorf("%s = %+v, want risk", key, s)
		}
	}
	if r.Level != LevelHighRisk {
		t.Fatalf("level = %s (score %d)", r.Level, r.Score)
	}
}

func TestTrendKnockout(t *testing.T) {
	// Slow bleed: below a falling SMA 200, but a shallow drawdown and calm
	// volatility. The score alone would say normal; the knock-out says caution.
	ret := func(i int) float64 {
		if i < 300 {
			return 0
		}
		return -0.0004
	}
	r, _ := Compute(Input{Index: path(500, 100, ret, 0.003)})
	if signal(t, r, "trend").Zone != ZoneRisk {
		t.Fatalf("trend = %+v", signal(t, r, "trend"))
	}
	if r.Level == LevelNormal {
		t.Fatalf("level normal with index under a falling SMA 200 (score %d)", r.Score)
	}
}

func TestNoDataSignalsDontCount(t *testing.T) {
	r, _ := Compute(Input{Index: path(500, 100, constant(0.0005), 0.004)})
	if s := signal(t, r, "breadth"); s.Zone != ZoneNoData {
		t.Errorf("breadth with no universe = %+v", s)
	}
	for _, key := range []string{"valuation", "macro"} {
		if signal(t, r, key).Zone != ZoneNoData {
			t.Errorf("%s should be nodata", key)
		}
	}
	// Only trend, drawdown and volatility count: max score 6.
	if r.MaxScore != 6 {
		t.Errorf("max score = %d, want 6", r.MaxScore)
	}
}

func TestShortHistory(t *testing.T) {
	if _, ok := Compute(Input{Index: path(150, 100, constant(0), 0)}); ok {
		t.Fatal("want not ok with 150 bars")
	}
}

func TestAnnualVol(t *testing.T) {
	// Alternating +1%/-1% log returns: stdev ~1% daily -> ~15.9% a year.
	bars := make([]market.Bar, 41)
	for i := range bars {
		bars[i].Close = 100 * math.Exp(0.01*float64(i%2))
	}
	if v := annualVol(bars, len(bars), 40); math.Abs(v-16.07) > 0.2 {
		t.Fatalf("vol = %v", v)
	}
}
