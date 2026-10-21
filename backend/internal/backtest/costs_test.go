package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// Hand-computed to the đồng: 10,000,000 at 23,450 with a 0.15% fee buys
// floor(10,000,000 / 23,485.175 / 100) = 4 lots; selling at 25,000
// pays the fee again.
func TestTrackerLotsAndFeesBothSides(t *testing.T) {
	tr := newTracker(10_000_000)
	tr.buy(0, 23_450)
	if tr.shares != 400 {
		t.Fatalf("shares = %v, want 400", tr.shares)
	}
	// 400 * 23,450 = 9,380,000; fee 14,070.
	if !approx(tr.cash, 10_000_000-9_380_000-14_070) {
		t.Fatalf("cash after buy = %v, want 605,930", tr.cash)
	}
	tr.sell(86400, 25_000)
	// 400 * 25,000 = 10,000,000; fee 15,000.
	if !approx(tr.cash, 605_930+10_000_000-15_000) || !approx(tr.fees, 14_070+15_000) {
		t.Fatalf("cash/fees after sell = %v/%v", tr.cash, tr.fees)
	}
	// Net round trip: 9,985,000 back on 9,394,070 paid.
	if want := round2((9_985_000 - 9_394_070.0) / 9_394_070 * 100); tr.tradeLog[0].ReturnPercent != want {
		t.Fatalf("trade return = %v, want %v", tr.tradeLog[0].ReturnPercent, want)
	}
}

func TestTrackerSkipsBuyBelowOneLot(t *testing.T) {
	tr := newTracker(1_000_000)
	tr.buy(0, 10_000) // one lot costs 1,001,500 with the fee
	if tr.inPosition() || tr.cash != 1_000_000 {
		t.Fatalf("want no fill, got shares=%v cash=%v", tr.shares, tr.cash)
	}
}

// A rule that never fires sits in cash 100% of the time, never fills,
// and ends where it started, while lump sum rides the rally.
func TestNeverFiringRuleIsAllCash(t *testing.T) {
	closes := make([]float64, 120)
	for i := range closes {
		closes[i] = 100 + float64(i) // steady rise: the fast EMA never crosses back
	}
	// A slow period longer than the series means no crossover at all.
	r := runEMACrossover(barsFromCloses(closes), 20, 119, capital)
	if r.totalTrades != 0 || r.finalCapital != capital {
		t.Fatalf("trades=%d final=%v", r.totalTrades, r.finalCapital)
	}
	if r.timeInCashPct != 100 || r.daysToFirstFill != nil {
		t.Fatalf("timeInCash=%v daysToFirstFill=%v", r.timeInCashPct, r.daysToFirstFill)
	}
	if r.lumpSum.ReturnPercent <= 0 || r.benchmarkReturnPercent != r.lumpSum.ReturnPercent {
		t.Fatalf("lump sum = %+v, benchmark %v", r.lumpSum, r.benchmarkReturnPercent)
	}
}

func TestDaysToFirstFill(t *testing.T) {
	tr := newTracker(capital)
	tr.mark(10*86400, 100)
	tr.mark(11*86400, 100)
	tr.buy(13*86400, 100)
	tr.mark(13*86400, 100)
	r := tr.result(13*86400, 100)
	if r.daysToFirstFill == nil || *r.daysToFirstFill != 3 {
		t.Fatalf("daysToFirstFill = %v, want 3", r.daysToFirstFill)
	}
	if !approx(r.timeInCashPct, 200.0/3) {
		t.Fatalf("timeInCash = %v, want 66.67", r.timeInCashPct)
	}
}

// dailyBars is one bar per day from start for n days, all at price.
func dailyBars(start time.Time, n int, price float64) []market.Bar {
	out := make([]market.Bar, n)
	for i := range out {
		c := price
		out[i] = market.Bar{Time: start.AddDate(0, 0, i).Unix(), Open: c, High: c, Low: c, Close: c}
	}
	return out
}

// Three months at 10,000: each 1,000,000 instalment is just short of one
// lot (1,001,500 with the fee), so month 1 buys nothing and the cash
// rolls over -- month 2 buys one lot, month 3 one more.
func TestMonthlyDCARollsOverCashALotCantAbsorb(t *testing.T) {
	bars := dailyBars(time.Date(2025, 1, 1, 15, 0, 0, 0, vnTime), 31+28+31, 10_000)
	b, curve := runMonthlyDCA(bars, 3_000_000)
	if len(curve) != len(bars) {
		t.Fatalf("curve has %d points for %d bars", len(curve), len(bars))
	}
	// Two lots bought, two fees of 1,500.
	if b.FeesPaid != 3_000 {
		t.Fatalf("fees = %v, want 3,000", b.FeesPaid)
	}
	if b.FinalCapital != 3_000_000-3_000 {
		t.Fatalf("final = %v, want 2,997,000", b.FinalCapital)
	}
	// Before the second instalment (end of January), nothing was bought.
	if curve[30] != 3_000_000 {
		t.Fatalf("end of January equity = %v, want all cash", curve[30])
	}
}

func TestMonthlyDCABuysEveryMonth(t *testing.T) {
	start := time.Date(2025, 1, 1, 15, 0, 0, 0, vnTime)
	var bars []market.Bar
	// Prices fall each month, so DCA ends with more shares than lump sum.
	for m, p := range []float64{20_000, 10_000, 5_000} {
		mb := dailyBars(start.AddDate(0, m, 0), 28, p)
		bars = append(bars, mb...)
	}
	dca, _ := runMonthlyDCA(bars, 30_000_000)
	lump, _ := runLumpSum(bars, 30_000_000)
	// Lump sum: 1,400 shares at 20,000 (28,000,000 + 42,000 fee), worth
	// 7,000,000 at 5,000 plus 1,958,000 cash.
	if lump.FinalCapital != 8_958_000 {
		t.Fatalf("lump final = %v", lump.FinalCapital)
	}
	// DCA: 10m per month buys 400 @ 20k, then 900 @ 10k (10m + 1.98m
	// carried), then lots @ 5k with the rest.
	if dca.FinalCapital <= lump.FinalCapital || dca.FeesPaid == 0 {
		t.Fatalf("DCA should beat lump sum into a falling market: dca %+v lump %+v", dca, lump)
	}
	if lump.MaxDrawdownPercent <= dca.MaxDrawdownPercent {
		t.Fatalf("lump sum should draw down more: lump %v dca %v", lump.MaxDrawdownPercent, dca.MaxDrawdownPercent)
	}
}
