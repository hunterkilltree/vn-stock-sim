package backtest

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/market"

type runResult struct {
	finalCapital       float64
	returnPercent      float64
	totalTrades        int
	winRate            float64
	maxDrawdownPercent float64
	profitFactor       float64
	// equity, trades and benchmarkReturnPercent feed the /backtest results
	// page (phase-k.md decision 13).
	equity                 []EquityPoint
	trades                 []Trade
	benchmarkReturnPercent float64
}

// tracker is the long-only, all-in/all-out bookkeeping both rules share:
// cash vs. shares, closed-trade wins/profit/loss, and a bar-by-bar equity
// peak for max drawdown.
type tracker struct {
	startingCapital float64
	cash            float64
	shares          float64
	entryPrice      float64
	trades, wins    int
	grossProfit     float64
	grossLoss       float64
	peak            float64
	maxDD           float64
	entryTime       int64
	firstClose      float64
	equity          []EquityPoint
	tradeLog        []Trade
}

func newTracker(startingCapital float64) *tracker {
	return &tracker{startingCapital: startingCapital, cash: startingCapital, peak: startingCapital}
}

func (t *tracker) inPosition() bool { return t.shares > 0 }

func (t *tracker) buy(at int64, price float64) {
	t.shares = t.cash / price
	t.cash = 0
	t.entryPrice = price
	t.entryTime = at
}

func (t *tracker) sell(at int64, price float64) {
	proceeds := t.shares * price
	cost := t.shares * t.entryPrice
	t.tradeLog = append(t.tradeLog, Trade{
		EntryTime: t.entryTime, EntryPrice: t.entryPrice, ExitTime: at, ExitPrice: price,
		ReturnPercent: round2((price - t.entryPrice) / t.entryPrice * 100),
	})
	t.cash = proceeds
	t.shares = 0
	t.trades++
	if proceeds > cost {
		t.wins++
		t.grossProfit += proceeds - cost
	} else {
		t.grossLoss += cost - proceeds
	}
}

// mark records the bar's equity next to buy-and-hold of the same
// capital from the first bar marked.
func (t *tracker) mark(at int64, price float64) {
	equity := t.cash + t.shares*price
	if t.firstClose == 0 {
		t.firstClose = price
	}
	t.equity = append(t.equity, EquityPoint{Time: at, Equity: round2(equity), Benchmark: round2(t.startingCapital * price / t.firstClose)})
	if equity > t.peak {
		t.peak = equity
	}
	if t.peak > 0 {
		if dd := (t.peak - equity) / t.peak * 100; dd > t.maxDD {
			t.maxDD = dd
		}
	}
}

func (t *tracker) result(lastTime int64, lastPrice float64) runResult {
	final := t.cash + t.shares*lastPrice
	trades := t.tradeLog
	if t.inPosition() { // still holding at the end: shown, not counted
		trades = append(trades, Trade{
			EntryTime: t.entryTime, EntryPrice: t.entryPrice, ExitTime: lastTime, ExitPrice: lastPrice,
			ReturnPercent: round2((lastPrice - t.entryPrice) / t.entryPrice * 100), Open: true,
		})
	}
	r := runResult{
		finalCapital:       final,
		returnPercent:      (final - t.startingCapital) / t.startingCapital * 100,
		totalTrades:        t.trades,
		maxDrawdownPercent: t.maxDD,
		equity:             t.equity,
		trades:             trades,
	}
	if t.firstClose > 0 {
		r.benchmarkReturnPercent = (lastPrice - t.firstClose) / t.firstClose * 100
	}
	if t.trades > 0 {
		r.winRate = float64(t.wins) / float64(t.trades) * 100
	}
	// 0 when there are no losing trades -- an undefined ratio, not a
	// fabricated large number (same convention as portfolio.Stats).
	if t.grossLoss > 0 {
		r.profitFactor = t.grossProfit / t.grossLoss
	}
	return r
}

// runEMACrossover is a minimal long-only simulation: buy all-in when the
// fast EMA crosses above the slow EMA, sell all when it crosses below.
func runEMACrossover(bars []market.Bar, fastPeriod, slowPeriod int, startingCapital float64) runResult {
	if len(bars) < slowPeriod+1 {
		return runResult{finalCapital: startingCapital}
	}

	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	fastEMA := emaSeries(closes, fastPeriod)
	slowEMA := emaSeries(closes, slowPeriod)

	t := newTracker(startingCapital)
	for i := 1; i < len(bars); i++ {
		if fastEMA[i] != 0 && slowEMA[i] != 0 && fastEMA[i-1] != 0 && slowEMA[i-1] != 0 {
			crossedUp := fastEMA[i-1] <= slowEMA[i-1] && fastEMA[i] > slowEMA[i]
			crossedDown := fastEMA[i-1] >= slowEMA[i-1] && fastEMA[i] < slowEMA[i]
			if !t.inPosition() && crossedUp {
				t.buy(bars[i].Time, bars[i].Close)
			} else if t.inPosition() && crossedDown {
				t.sell(bars[i].Time, bars[i].Close)
			}
		}
		t.mark(bars[i].Time, bars[i].Close)
	}
	last := bars[len(bars)-1]
	return t.result(last.Time, last.Close)
}

type rsiReversionParams struct {
	period          int
	entry, exit     float64
	stopLossPercent float64
	trendSMA        int // 0 = no trend filter
}

// runRSIReversion buys when RSI crosses up through `entry` (and, if set,
// the close is above SMA(trendSMA)), and sells when RSI rises above
// `exit` or the close falls stopLossPercent below the entry price. This
// is Quant.dc.html's sample strategy ("mua khi RSI cắt lên 35, bán khi
// RSI vượt 70 hoặc lỗ 7%") -- see phase-h.md decision 3.
func runRSIReversion(bars []market.Bar, p rsiReversionParams, startingCapital float64) runResult {
	if len(bars) < p.period+2 {
		return runResult{finalCapital: startingCapital}
	}

	rsiAt := make(map[int64]float64, len(bars))
	for _, pt := range market.RSI(bars, p.period) {
		rsiAt[pt.Time] = pt.Value
	}
	smaAt := map[int64]float64{}
	if p.trendSMA > 0 {
		for _, pt := range market.SMA(bars, p.trendSMA) {
			smaAt[pt.Time] = pt.Value
		}
	}

	t := newTracker(startingCapital)
	for i := 1; i < len(bars); i++ {
		price := bars[i].Close
		cur, okCur := rsiAt[bars[i].Time]
		prev, okPrev := rsiAt[bars[i-1].Time]
		if okCur && okPrev {
			if !t.inPosition() {
				trendOK := true
				if p.trendSMA > 0 {
					s, ok := smaAt[bars[i].Time]
					trendOK = ok && price > s
				}
				if prev < p.entry && cur >= p.entry && trendOK {
					t.buy(bars[i].Time, price)
				}
			} else {
				stopHit := p.stopLossPercent > 0 && price <= t.entryPrice*(1-p.stopLossPercent/100)
				if cur > p.exit || stopHit {
					t.sell(bars[i].Time, price)
				}
			}
		}
		t.mark(bars[i].Time, price)
	}
	last := bars[len(bars)-1]
	return t.result(last.Time, last.Close)
}

// emaSeries returns an EMA aligned index-for-index with closes; entries
// before the EMA has enough data are 0 (checked by the caller).
func emaSeries(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if period <= 0 || len(closes) < period {
		return out
	}
	k := 2.0 / float64(period+1)
	var sum float64
	var prev float64
	for i, c := range closes {
		if i < period {
			sum += c
			if i == period-1 {
				prev = sum / float64(period)
				out[i] = prev
			}
			continue
		}
		prev = c*k + prev*(1-k)
		out[i] = prev
	}
	return out
}
