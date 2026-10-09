package valuation

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.5 }

func TestGrahamNumber(t *testing.T) {
	// sqrt(22.5 * 4000 * 20000) = sqrt(1.8e9) = 42426.4
	v := Evaluate(GrahamNumber, Fundamentals{EPS: 4000, BVPS: 20000}, DefaultParams())
	if !v.Applicable || !near(v.FairValue, 42426.4) || !near(v.BuyPrice, 42426.4*0.75) {
		t.Fatalf("got %+v", v)
	}
}

func TestGrahamFormula(t *testing.T) {
	p := DefaultParams()
	p.BondYieldPct = 4.4 // 4.4/Y = 1
	// 3000 * (8.5 + 2*10) = 85,500
	v := Evaluate(GrahamFormula, Fundamentals{EPS: 3000, GrowthPct: 10}, p)
	if !v.Applicable || !near(v.FairValue, 85500) {
		t.Fatalf("got %+v", v)
	}
	p.BondYieldPct = 8.8 // doubling the yield halves the value
	if v := Evaluate(GrahamFormula, Fundamentals{EPS: 3000, GrowthPct: 10}, p); !near(v.FairValue, 42750) {
		t.Fatalf("got %+v", v)
	}
}

func TestLynch(t *testing.T) {
	if v := Evaluate(LynchFair, Fundamentals{EPS: 2000, GrowthPct: 15}, DefaultParams()); !near(v.FairValue, 30000) || v.BuyPrice != v.FairValue {
		t.Fatalf("got %+v", v)
	}
	if v := Evaluate(LynchFair, Fundamentals{EPS: 2000, GrowthPct: 4}, DefaultParams()); v.Applicable {
		t.Fatal("g < 5 should not apply")
	}
}

func TestRule1Sensitivity(t *testing.T) {
	p := DefaultParams()
	hi := Evaluate(Rule1Sticker, Fundamentals{EPS: 1000, GrowthPct: 15}, p)
	lo := Evaluate(Rule1Sticker, Fundamentals{EPS: 1000, GrowthPct: 10}, p)
	// 1000 * 1.15^10 * 30 / 1.15^10 = 30,000; g=10 -> 1000*1.1^10*20/1.15^10 = 12,823.
	if !near(hi.FairValue, 30000) || !near(lo.FairValue, 12822.7) {
		t.Fatalf("hi %+v lo %+v", hi, lo)
	}
	if !near(hi.BuyPrice, 15000) {
		t.Fatalf("buy price = %v, want 50%% of sticker", hi.BuyPrice)
	}
	if lo.FairValue/hi.FairValue > 0.5 {
		t.Fatal("cutting g 15->10 should more than halve the sticker price")
	}
}

func TestWeissAndPEBand(t *testing.T) {
	// DPS 2000, high yield 8% -> buy at <= 2000 / 0.072 = 27,778
	if v := Evaluate(WeissYield, Fundamentals{CashDPS: 2000, HighYield: 0.08}, DefaultParams()); !near(v.FairValue, 27777.8) {
		t.Fatalf("got %+v", v)
	}
	if v := Evaluate(PEBand, Fundamentals{EPS: 2000, AvgHistPE: 12, StdHistPE: 3}, DefaultParams()); !near(v.FairValue, 18000) {
		t.Fatalf("got %+v", v)
	}
}

func TestNotApplicable(t *testing.T) {
	for _, m := range Methods {
		v := Evaluate(m, Fundamentals{EPS: -100, BVPS: 10000}, DefaultParams())
		if v.Applicable || v.Reason == "" {
			t.Errorf("%s with negative EPS: %+v", m, v)
		}
	}
}

func TestSummarizePicksLowestAndRounds(t *testing.T) {
	f := Fundamentals{Price: 30000, EPS: 4000, BVPS: 20000, ROE: 20}
	s := Summarize(f, DefaultParams(), func(p float64) float64 { return math.Floor(p/50) * 50 })
	// Only Graham Number applies: 42426 * 0.75 = 31819.8 -> 31800.
	if s.BuyMethod != GrahamNumber || s.BuyPrice != 31800 || !s.InBuyZone || !s.Quality.Passed {
		t.Fatalf("got %+v", s)
	}
	if len(s.Values) != len(Methods) {
		t.Fatalf("want every method listed, got %d", len(s.Values))
	}
}

func TestQualityGate(t *testing.T) {
	if q := CheckQuality(Fundamentals{EPS: 1000, ROE: 6}); q.Passed {
		t.Fatal("ROE 6% should fail")
	}
}
