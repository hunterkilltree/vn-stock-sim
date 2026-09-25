package backtest

import (
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

func barsFromCloses(closes []float64) []market.Bar {
	out := make([]market.Bar, len(closes))
	for i, c := range closes {
		out[i] = market.Bar{Time: int64(i) * 86400, Open: c, High: c, Low: c, Close: c}
	}
	return out
}

// A slide (RSI -> near 0), a rebound (RSI crosses up through 35 -> buy),
// then a rally until RSI > 70 -> sell at a profit.
func TestRSIReversionRoundTrip(t *testing.T) {
	var closes []float64
	p := 100.0
	for i := 0; i < 20; i++ {
		p -= 2
		closes = append(closes, p)
	}
	for i := 0; i < 30; i++ {
		p += 1.5
		closes = append(closes, p)
	}
	r := runRSIReversion(barsFromCloses(closes), rsiReversionParams{period: 14, entry: 35, exit: 70}, 1000)
	if r.totalTrades != 1 {
		t.Fatalf("want 1 closed trade, got %d", r.totalTrades)
	}
	if r.winRate != 100 || r.finalCapital <= 1000 {
		t.Fatalf("want a winning trade, got winRate=%v final=%v", r.winRate, r.finalCapital)
	}
	if r.profitFactor != 0 {
		t.Fatalf("profit factor must be 0 with no losing trades, got %v", r.profitFactor)
	}
}

// After entry the price keeps falling: the 7% stop must close the trade
// at a loss before RSI ever reaches the exit level.
func TestRSIReversionStopLoss(t *testing.T) {
	var closes []float64
	p := 100.0
	for i := 0; i < 20; i++ {
		p -= 2
		closes = append(closes, p)
	}
	for i := 0; i < 6; i++ { // rebound enough for RSI to cross 35
		p += 2
		closes = append(closes, p)
	}
	for i := 0; i < 10; i++ {
		p -= 3
		closes = append(closes, p)
	}
	r := runRSIReversion(barsFromCloses(closes), rsiReversionParams{period: 14, entry: 35, exit: 70, stopLossPercent: 7}, 1000)
	if r.totalTrades != 1 || r.winRate != 0 {
		t.Fatalf("want exactly one losing trade, got trades=%d winRate=%v", r.totalTrades, r.winRate)
	}
	if r.finalCapital >= 1000 || r.maxDrawdownPercent <= 0 {
		t.Fatalf("want a loss and a drawdown, got final=%v dd=%v", r.finalCapital, r.maxDrawdownPercent)
	}
}

func TestTrendFilterBlocksEntryBelowSMA(t *testing.T) {
	var closes []float64
	p := 100.0
	for i := 0; i < 20; i++ {
		p -= 2
		closes = append(closes, p)
	}
	for i := 0; i < 30; i++ {
		p += 1.5
		closes = append(closes, p)
	}
	// When RSI crosses 35 (early in the rebound) SMA(40) has no value yet,
	// and the filter requires price > SMA, so no entry is allowed.
	r := runRSIReversion(barsFromCloses(closes), rsiReversionParams{period: 14, entry: 35, exit: 70, trendSMA: 40}, 1000)
	if r.totalTrades != 0 || r.finalCapital != 1000 {
		t.Fatalf("trend filter should block entry, got trades=%d final=%v", r.totalTrades, r.finalCapital)
	}
}

// phase-k.md decision 13: every bar has an equity point next to
// buy-and-hold, each round trip is listed, and a position still held at
// the end shows as an open trade without counting as a closed one.
func TestResultCarriesEquityTradesAndBenchmark(t *testing.T) {
	var closes []float64
	p := 100.0
	for i := 0; i < 20; i++ {
		p -= 2
		closes = append(closes, p)
	}
	for i := 0; i < 30; i++ {
		p += 1.5
		closes = append(closes, p)
	}
	bars := barsFromCloses(closes)
	r := runRSIReversion(bars, rsiReversionParams{period: 14, entry: 35, exit: 70}, 1000)
	if len(r.equity) != len(bars)-1 {
		t.Fatalf("want one equity point per bar after the first, got %d", len(r.equity))
	}
	if len(r.trades) != r.totalTrades || r.trades[0].Open || r.trades[0].ExitPrice <= r.trades[0].EntryPrice {
		t.Fatalf("closed trades should be listed as winning round trips: %+v", r.trades)
	}
	first, last := closes[1], closes[len(closes)-1]
	if want := (last - first) / first * 100; r.benchmarkReturnPercent != want {
		t.Fatalf("buy-and-hold: want %v, got %v", want, r.benchmarkReturnPercent)
	}
	if b := r.equity[len(r.equity)-1].Benchmark; b != round2(1000*last/first) {
		t.Fatalf("benchmark curve ends at %v, want %v", b, round2(1000*last/first))
	}

	// Stop the series right after the buy: the trade is open.
	entry := int(r.trades[0].EntryTime / 86400)
	open := runRSIReversion(bars[:entry+2], rsiReversionParams{period: 14, entry: 35, exit: 70}, 1000)
	if open.totalTrades != 0 || len(open.trades) != 1 || !open.trades[0].Open {
		t.Fatalf("a position held at the end is one open trade: %+v", open.trades)
	}
}
