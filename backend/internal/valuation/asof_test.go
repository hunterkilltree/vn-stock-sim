package valuation

import (
	"math"
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 15, 0, 0, 0, vnLoc) }

func closeTo(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// quarters builds every quarter from fromYear Q1 through toYear Q4 with
// the given quarterly net income and shares, published on the deadline.
func quarters(fromYear, toYear int, ni func(y, q int) float64, shares float64) []Period {
	var ps []Period
	for y := fromYear; y <= toYear; y++ {
		for q := 1; q <= 4; q++ {
			ps = append(ps, Period{Year: y, Quarter: q, NetIncome: ni(y, q), Equity: 100e9, TotalDebt: 50e9,
				Shares: shares, PublishedAt: FilingDeadline(y, q)})
		}
	}
	return ps
}

func annual(y int, ni, equity, shares float64) Period {
	return Period{Year: y, NetIncome: ni, Equity: equity, Shares: shares, PublishedAt: FilingDeadline(y, 0)}
}

// dailyCloses is one close per calendar day in [from, to] at price(d).
func dailyCloses(from, to time.Time, price func(time.Time) float64) []Close {
	var cs []Close
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		cs = append(cs, Close{Time: d, Price: price(d)})
	}
	return cs
}

func TestFilingDeadline(t *testing.T) {
	cases := []struct {
		year, quarter int
		want          string
	}{
		{2025, 1, "2025-05-15"}, // Mar 31 + 45
		{2025, 4, "2026-02-14"}, // Dec 31 + 45
		{2025, 0, "2026-03-31"}, // Dec 31 + 90
	}
	for _, c := range cases {
		if got := FilingDeadline(c.year, c.quarter).Format(time.DateOnly); got != c.want {
			t.Errorf("FilingDeadline(%d, %d) = %s, want %s", c.year, c.quarter, got, c.want)
		}
	}
}

func TestAsOfHidesUnpublishedPeriods(t *testing.T) {
	// 2024 quarters earn 1bn each; 2025 quarters earn 2bn. 100m shares.
	ps := quarters(2024, 2025, func(y, q int) float64 {
		if y == 2025 {
			return 2e9
		}
		return 1e9
	}, 100e6)
	cs := dailyCloses(day(2024, 1, 1), day(2026, 6, 30), func(time.Time) float64 { return 50000 })

	// Q4 2025 is due 2026-02-14: the day before, TTM is Q4'24..Q3'25.
	before := AsOf(ps, nil, cs, day(2026, 2, 13), AsOfOptions{})
	if want := (1e9 + 3*2e9) / 100e6; !closeTo(before.EPS, want, 0.01) {
		t.Errorf("EPS before Q4 publication = %v, want %v", before.EPS, want)
	}
	after := AsOf(ps, nil, cs, day(2026, 2, 14), AsOfOptions{})
	if want := 4 * 2e9 / 100e6; !closeTo(after.EPS, want, 0.01) {
		t.Errorf("EPS after Q4 publication = %v, want %v", after.EPS, want)
	}
	if after.Price != 50000 || !closeTo(after.BVPS, 1000, 0.01) || !closeTo(after.DebtToEquity, 0.5, 1e-9) {
		t.Errorf("price/BVPS/DE = %v/%v/%v", after.Price, after.BVPS, after.DebtToEquity)
	}
	// TTM 8bn on equity 100bn at both ends.
	if !closeTo(after.ROE, 8, 1e-9) {
		t.Errorf("ROE = %v, want 8", after.ROE)
	}
}

func TestAsOfFallsBackToAnnualWhenQuartersHaveAGap(t *testing.T) {
	ps := quarters(2025, 2025, func(int, int) float64 { return 1e9 }, 100e6)
	ps = append(ps[:1], ps[2:]...) // drop Q2 2025
	ps = append(ps, annual(2025, 5e9, 100e9, 100e6))
	s := AsOf(ps, nil, []Close{{day(2026, 6, 1), 40000}}, day(2026, 6, 1), AsOfOptions{})
	if !closeTo(s.EPS, 50, 1e-9) {
		t.Fatalf("EPS = %v, want 50 (annual)", s.EPS)
	}
}

func TestAsOfGrowthRestatesBonusIssues(t *testing.T) {
	// Net income +10%/yr. A 100:10 bonus issue in mid-2023 lifts shares
	// from 100m to 110m: raw EPS growth looks lower, restated it's 10%.
	ps := []Period{
		annual(2021, 100e9, 1000e9, 100e6),
		annual(2022, 110e9, 1000e9, 100e6),
		annual(2023, 121e9, 1000e9, 110e6),
		annual(2024, 133.1e9, 1000e9, 110e6),
	}
	divs := []Dividend{{ExDate: day(2023, 7, 1), StockRatio: 0.10}}
	s := AsOf(ps, divs, []Close{{day(2025, 6, 1), 30000}}, day(2025, 6, 1), AsOfOptions{})
	if !closeTo(s.GrowthPct, 10, 1e-6) {
		t.Fatalf("GrowthPct = %v, want 10", s.GrowthPct)
	}
	if !closeTo(s.EPS, 133.1e9/110e6, 1e-6) {
		t.Fatalf("EPS = %v", s.EPS)
	}
}

func TestAsOfGrowthUndefinedAfterLoss(t *testing.T) {
	ps := []Period{
		annual(2021, -10e9, 1000e9, 100e6),
		annual(2022, 50e9, 1000e9, 100e6),
		annual(2023, 80e9, 1000e9, 100e6),
		annual(2024, 120e9, 1000e9, 100e6),
	}
	s := AsOf(ps, nil, []Close{{day(2025, 6, 1), 30000}}, day(2025, 6, 1), AsOfOptions{})
	if s.GrowthPct != 0 {
		t.Errorf("GrowthPct = %v, want 0 (start year was a loss)", s.GrowthPct)
	}
	if s.RecentLoss {
		t.Error("2021 is outside the last 3 years: RecentLoss should be false")
	}
	ps[1].NetIncome = -1
	if s := AsOf(ps, nil, []Close{{day(2025, 6, 1), 30000}}, day(2025, 6, 1), AsOfOptions{}); !s.RecentLoss {
		t.Error("loss in 2022 should set RecentLoss")
	}
	if q := CheckQuality(Fundamentals{EPS: 1000, RecentLoss: true}); q.Passed {
		t.Error("quality gate should fail on a recent loss")
	}
}

func TestAsOfROEStdDev(t *testing.T) {
	ps := []Period{
		annual(2022, 100e9, 1000e9, 100e6), // 10%
		annual(2023, 200e9, 1000e9, 100e6), // 20%
		annual(2024, 300e9, 1000e9, 100e6), // 30%
	}
	s := AsOf(ps, nil, []Close{{day(2025, 6, 1), 30000}}, day(2025, 6, 1), AsOfOptions{})
	if want := math.Sqrt(200.0 / 3); !closeTo(s.ROEStdDev, want, 1e-9) {
		t.Fatalf("ROEStdDev = %v, want %v", s.ROEStdDev, want)
	}
}

func TestAsOfPEBandNeedsThreeYears(t *testing.T) {
	// EPS 2000 every year from 2018, price 20,000: P/E is 10 throughout.
	ps := quarters(2018, 2025, func(int, int) float64 { return 50e9 }, 100e6)
	price := func(time.Time) float64 { return 20000 }

	long := AsOf(ps, nil, dailyCloses(day(2019, 1, 1), day(2026, 6, 30), price), day(2026, 6, 30), AsOfOptions{})
	if !closeTo(long.AvgHistPE, 10, 1e-9) || long.StdHistPE > 1e-9 {
		t.Fatalf("P/E band = %v ± %v, want 10 ± 0", long.AvgHistPE, long.StdHistPE)
	}
	short := AsOf(ps, nil, dailyCloses(day(2024, 7, 1), day(2026, 6, 30), price), day(2026, 6, 30), AsOfOptions{})
	if short.AvgHistPE != 0 {
		t.Fatalf("two years of prices should leave the band unknown, got %v", short.AvgHistPE)
	}
}

func TestAsOfDividendYield(t *testing.T) {
	ps := quarters(2019, 2025, func(int, int) float64 { return 50e9 }, 100e6)
	var divs []Dividend
	for y := 2020; y <= 2026; y++ {
		cash := 1000.0
		if y == 2022 {
			cash = 3000 // one generous year
		}
		divs = append(divs, Dividend{ExDate: day(y, 5, 10), Cash: cash})
	}
	cs := dailyCloses(day(2020, 1, 1), day(2026, 6, 30), func(time.Time) float64 { return 25000 })
	s := AsOf(ps, divs, cs, day(2026, 6, 30), AsOfOptions{})
	if s.CashDPS != 1000 {
		t.Errorf("CashDPS = %v, want 1000 (last 12 months)", s.CashDPS)
	}
	if !closeTo(s.HighYield, 3000.0/25000, 1e-12) {
		t.Errorf("HighYield = %v, want 0.12", s.HighYield)
	}
	// The 2022 payment must not be visible from 2021.
	if s := AsOf(ps, divs, cs, day(2021, 12, 31), AsOfOptions{}); s.CashDPS != 1000 {
		t.Errorf("CashDPS in 2021 = %v, want 1000", s.CashDPS)
	}
}

// A 1:1 bonus issue halves the unadjusted price on its ex-date. Either
// price basis must give the same, flat P/E.
func TestAsOfPEConsistentAcrossPriceBasis(t *testing.T) {
	ex := day(2023, 1, 16)
	var ps []Period
	for y := 2018; y <= 2025; y++ {
		for q := 1; q <= 4; q++ {
			shares := 100e6
			if (Period{Year: y, Quarter: q}).End().After(ex) {
				shares = 200e6
			}
			ps = append(ps, Period{Year: y, Quarter: q, NetIncome: 50e9, Equity: 1000e9, Shares: shares,
				PublishedAt: FilingDeadline(y, q)})
		}
	}
	divs := []Dividend{{ExDate: ex, StockRatio: 1}}
	from, to := day(2019, 1, 1), day(2026, 6, 30)

	unadjusted := dailyCloses(from, to, func(d time.Time) float64 {
		if d.Before(ex) {
			return 20000
		}
		return 10000
	})
	adjusted := dailyCloses(from, to, func(time.Time) float64 { return 10000 })

	for name, c := range map[string]struct {
		closes []Close
		opt    AsOfOptions
	}{
		"unadjusted": {unadjusted, AsOfOptions{}},
		"adjusted":   {adjusted, AsOfOptions{AdjustedCloses: true}},
	} {
		s := AsOf(ps, divs, c.closes, to, c.opt)
		if !closeTo(s.AvgHistPE, 10, 1e-9) || s.StdHistPE > 1e-9 {
			t.Errorf("%s: P/E band = %v ± %v, want 10 ± 0", name, s.AvgHistPE, s.StdHistPE)
		}
	}
}

func TestBondYieldOn(t *testing.T) {
	latest := LatestBondYield()
	if _, ok := BondYieldOn(day(2000, 1, 1)); ok {
		t.Error("before the series starts the yield must be unknown")
	}
	got, ok := BondYieldOn(day(2030, 1, 1))
	if !ok || got != latest {
		t.Errorf("BondYieldOn(2030) = %+v, %v; want %+v", got, ok, latest)
	}
	if p := DefaultParams(); p.BondYieldPct != latest.YieldPct || p.BondYieldAsOf != latest.Date {
		t.Errorf("DefaultParams yield = %v @ %s", p.BondYieldPct, p.BondYieldAsOf)
	}
}
