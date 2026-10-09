package rating

import (
	"math"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// geometric builds n daily bars compounding by rate per day, with a little
// alternating noise so volatility is non-zero, and volume shares/day.
func geometric(n int, start, rate float64, volume int64) []market.Bar {
	bars := make([]market.Bar, n)
	c := start
	for i := range bars {
		c *= 1 + rate
		noise := 1.0
		if i%2 == 0 {
			noise = 1.003
		}
		v := c * noise
		bars[i] = market.Bar{Time: int64(i) * day, Open: v, High: v * 1.005, Low: v * 0.995, Close: v, Volume: volume}
	}
	return bars
}

func stock(eps, roe, pb, pe, dy float64) symbol.Detail {
	return symbol.Detail{
		Symbol: symbol.Symbol{Symbol: "TST", Exchange: "HOSE", Sector: "X"},
		EPS:    eps, ROE: roe, PBRatio: pb, PERatio: pe, DividendYield: dy, MarketCap: 50e12,
	}
}

func factor(t *testing.T, h Horizon, key string) Factor {
	t.Helper()
	for _, f := range h.Factors {
		if f.Key == key {
			return f
		}
	}
	t.Fatalf("no factor %q", key)
	return Factor{}
}

func TestUptrendQualityIsSuitable(t *testing.T) {
	// ~+20%/yr stock vs a flat index, ROE 22% at P/B 1.5 (fair 2.1), P/E
	// below the sector median, 5% yield, deep liquidity.
	h := ComputeHorizons(HorizonInput{
		Detail:         stock(3000, 22, 1.5, 9, 5),
		Bars:           geometric(1500, 20000, 0.0005, 2_000_000),
		IndexBars:      geometric(1500, 1000, 0, 1),
		SectorMedianPE: 12,
	})
	if h == nil {
		t.Fatal("nil horizons")
	}
	if h.Medium.Verdict != VerdictSuitable || h.Long.Verdict != VerdictSuitable {
		t.Fatalf("medium %s (%v), long %s (%v)", h.Medium.Verdict, h.Medium.Score, h.Long.Verdict, h.Long.Score)
	}
	if f := factor(t, h.Medium, "trend"); f.Score != 100 {
		t.Errorf("trend = %+v", f)
	}
	if len(h.Long.History) == 0 || h.Long.History[0].WinRate != 100 || h.Long.History[0].Index == nil {
		t.Errorf("history = %+v", h.Long.History)
	}
	if h.Medium.Risk.StopLoss <= 0 || h.Medium.Risk.StopPct >= 0 {
		t.Errorf("risk = %+v", h.Medium.Risk)
	}
}

func TestDowntrendLossMakingIsAvoid(t *testing.T) {
	h := ComputeHorizons(HorizonInput{
		Detail:    stock(-500, 3, 4, 0, 0),
		Bars:      geometric(1500, 50000, -0.0006, 2_000_000),
		IndexBars: geometric(1500, 1000, 0, 1),
	})
	if h.Medium.Verdict != VerdictAvoid || h.Long.Verdict != VerdictAvoid {
		t.Fatalf("medium %s (%v), long %s (%v)", h.Medium.Verdict, h.Medium.Score, h.Long.Verdict, h.Long.Score)
	}
	if f := factor(t, h.Long, "relative_pe"); !f.Available || f.Score != 0 {
		t.Errorf("loss-making relative P/E = %+v", f)
	}
}

func TestIlliquidIsKnockedOut(t *testing.T) {
	h := ComputeHorizons(HorizonInput{
		Detail: stock(3000, 22, 1.5, 9, 5),
		Bars:   geometric(1500, 20000, 0.0005, 100), // ~3m VND/day
	})
	if h.Medium.Verdict != VerdictAvoid || h.Medium.Knockout == "" || h.Long.Verdict != VerdictAvoid {
		t.Fatalf("medium %+v / long %s", h.Medium.Knockout, h.Long.Verdict)
	}
}

func TestMissingDataRenormalises(t *testing.T) {
	h := ComputeHorizons(HorizonInput{Detail: stock(3000, 22, 1.5, 9, 5), Bars: geometric(1500, 20000, 0.0005, 2_000_000)})
	if !h.Long.Partial {
		t.Error("long should be partial while the H2 factors are missing")
	}
	if f := factor(t, h.Long, "eps_growth"); f.Available {
		t.Errorf("eps_growth = %+v", f)
	}
	// No index: relative strength falls back to the absolute return.
	if f := factor(t, h.Medium, "relative_strength"); !f.Available {
		t.Errorf("relative strength without index = %+v", f)
	}
	// No sector median: relative P/E has no data, and doesn't drag the score.
	if f := factor(t, h.Medium, "relative_pe"); f.Available {
		t.Errorf("relative P/E without median = %+v", f)
	}
	if h.Medium.Score < suitableCutoff {
		t.Errorf("medium score %v: missing factors must not count as zero", h.Medium.Score)
	}
}

func TestShortHistory(t *testing.T) {
	if ComputeHorizons(HorizonInput{Detail: stock(1, 1, 1, 1, 1), Bars: geometric(30, 100, 0, 1000)}) != nil {
		t.Fatal("want nil with 30 bars")
	}
	h := ComputeHorizons(HorizonInput{Detail: stock(3000, 22, 1.5, 9, 5), Bars: geometric(300, 20000, 0.0005, 2_000_000)})
	for _, s := range h.Long.History {
		if s.Days == 3*365 {
			t.Error("3-year history from 300 bars")
		}
	}
	if f := factor(t, h.Long, "long_run"); f.Available {
		t.Errorf("3-year factor from 300 bars = %+v", f)
	}
}

func TestHoldStats(t *testing.T) {
	// Alternating +10% / -10% every 30 days: half the 30-day windows win.
	bars := make([]market.Bar, 400)
	for i := range bars {
		c := 100.0
		if (i/30)%2 == 1 {
			c = 110
		}
		bars[i] = market.Bar{Time: int64(i) * day, Close: c}
	}
	s, ok := holdStats(bars, holdWindow{"30d", 30})
	if !ok || math.Abs(s.WinRate-50) > 5 || s.Samples != 370 {
		t.Fatalf("got %+v", s)
	}
}

func TestLerp(t *testing.T) {
	if lerp(5, 0, 10) != 50 || lerp(-1, 0, 10) != 0 || lerp(20, 0, 10) != 100 || lerp(30, 60, 25) != 85.7 {
		t.Fatal("lerp")
	}
}
