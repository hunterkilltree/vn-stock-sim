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
