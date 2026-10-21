package backtest

import (
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/order"
)

// lotSize is the HOSE board lot: orders are whole multiples of 100
// shares (phase-valuation-backtest.md slice 1).
const lotSize = 100

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
	// Slice 1 of phase-valuation-backtest.md: what the rule paid in fees,
	// how long it sat in cash, and the two baselines with the same
	// capital, fees and lots.
	feesPaid        float64
	timeInCashPct   float64
	daysToFirstFill *int
	lumpSum, dca    Baseline
}

// lotsFor is how many shares cash buys in whole lots, fee included.
func lotsFor(cash, price float64) float64 {
	if price <= 0 {
		return 0
	}
	lots := int64(cash / (price * (1 + order.FeeRate)) / lotSize)
	return float64(lots * lotSize)
}

// tracker is the long-only, all-in/all-out bookkeeping both rules share:
// cash vs. shares in whole lots, fees on both sides, closed-trade
// wins/profit/loss, and a bar-by-bar equity peak for max drawdown.
type tracker struct {
	startingCapital float64
	cash            float64
	shares          float64
	entryPrice      float64
	entryCost       float64 // what the open position cost, fee included
	trades, wins    int
	grossProfit     float64
	grossLoss       float64
	fees            float64
	peak            float64
	maxDD           float64
	entryTime       int64
	firstFill       int64
	firstMark       int64
	barsMarked      int
	barsInCash      int
	equity          []EquityPoint
	tradeLog        []Trade
}

func newTracker(startingCapital float64) *tracker {
	return &tracker{startingCapital: startingCapital, cash: startingCapital, peak: startingCapital}
}

func (t *tracker) inPosition() bool { return t.shares > 0 }

// buy goes all-in in whole lots; leftover cash stays cash. It does
// nothing when the cash can't cover one lot.
func (t *tracker) buy(at int64, price float64) {
	qty := lotsFor(t.cash, price)
	if qty == 0 {
		return
	}
	notional := qty * price
	fee := notional * order.FeeRate
	t.cash -= notional + fee
	t.fees += fee
	t.shares = qty
	t.entryPrice = price
	t.entryCost = notional + fee
	t.entryTime = at
	if t.firstFill == 0 {
		t.firstFill = at
	}
}

// netReturn is a round trip's return after both fees, in percent.
func netReturn(cost, proceeds float64) float64 {
	return round2((proceeds - cost) / cost * 100)
}

func (t *tracker) sell(at int64, price float64) {
	notional := t.shares * price
	fee := notional * order.FeeRate
	proceeds := notional - fee
	t.tradeLog = append(t.tradeLog, Trade{
		EntryTime: t.entryTime, EntryPrice: t.entryPrice, ExitTime: at, ExitPrice: price,
		ReturnPercent: netReturn(t.entryCost, proceeds),
	})
	t.cash += proceeds
	t.fees += fee
	t.shares = 0
	t.trades++
	if proceeds > t.entryCost {
		t.wins++
		t.grossProfit += proceeds - t.entryCost
	} else {
		t.grossLoss += t.entryCost - proceeds
	}
}

// mark records the bar's equity (shares at the close, before any exit
// fee). The baselines are filled in afterwards by withBaselines.
func (t *tracker) mark(at int64, price float64) {
	equity := t.cash + t.shares*price
	if t.firstMark == 0 {
		t.firstMark = at
	}
	t.barsMarked++
	if !t.inPosition() {
		t.barsInCash++
	}
	t.equity = append(t.equity, EquityPoint{Time: at, Equity: round2(equity)})
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
		exitFee := t.shares * lastPrice * order.FeeRate
		trades = append(trades, Trade{
			EntryTime: t.entryTime, EntryPrice: t.entryPrice, ExitTime: lastTime, ExitPrice: lastPrice,
			ReturnPercent: netReturn(t.entryCost, t.shares*lastPrice-exitFee), Open: true,
		})
	}
	r := runResult{
		finalCapital:       final,
		returnPercent:      (final - t.startingCapital) / t.startingCapital * 100,
		totalTrades:        t.trades,
		maxDrawdownPercent: t.maxDD,
		equity:             t.equity,
		trades:             trades,
		feesPaid:           t.fees,
	}
	if t.barsMarked > 0 {
		r.timeInCashPct = float64(t.barsInCash) / float64(t.barsMarked) * 100
	}
	if t.firstFill > 0 {
		days := int((t.firstFill - t.firstMark) / 86400)
		r.daysToFirstFill = &days
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

// withBaselines adds lump sum and monthly DCA over the bars the rule
// was marked on, with the same capital, fees and lots, and lines them
// up with the rule's equity points.
func withBaselines(r runResult, marked []market.Bar, capital float64) runResult {
	if len(marked) == 0 || len(marked) != len(r.equity) {
		return r
	}
	lump, lumpCurve := runLumpSum(marked, capital)
	dca, dcaCurve := runMonthlyDCA(marked, capital)
	for i := range r.equity {
		r.equity[i].Benchmark = lumpCurve[i]
		r.equity[i].DCA = dcaCurve[i]
	}
	r.lumpSum, r.dca = lump, dca
	r.benchmarkReturnPercent = lump.ReturnPercent
	return r
}

// baselineCurve tracks equity and drawdown for a baseline.
type baselineCurve struct {
	capital, peak, maxDD float64
	points               []float64
}

func (c *baselineCurve) mark(equity float64) {
	c.points = append(c.points, round2(equity))
	if equity > c.peak {
		c.peak = equity
	}
	if c.peak > 0 {
		if dd := (c.peak - equity) / c.peak * 100; dd > c.maxDD {
			c.maxDD = dd
		}
	}
}

func (c *baselineCurve) result(final, fees float64) Baseline {
	return Baseline{
		FinalCapital:       round2(final),
		ReturnPercent:      round2((final - c.capital) / c.capital * 100),
		MaxDrawdownPercent: round2(c.maxDD),
		FeesPaid:           round2(fees),
	}
}

// runLumpSum buys as many lots as the capital allows at the first close
// and holds to the end.
func runLumpSum(bars []market.Bar, capital float64) (Baseline, []float64) {
	c := &baselineCurve{capital: capital, peak: capital}
	qty := lotsFor(capital, bars[0].Close)
	fee := qty * bars[0].Close * order.FeeRate
	cash := capital - qty*bars[0].Close - fee
	for _, b := range bars {
		c.mark(cash + qty*b.Close)
	}
	return c.result(cash+qty*bars[len(bars)-1].Close, fee), c.points
}

// runMonthlyDCA splits the capital evenly over the calendar months in
// the range and buys on each month's first session, as many lots as
// the cash on hand allows; what a lot can't absorb rolls into the next
// month.
func runMonthlyDCA(bars []market.Bar, capital float64) (Baseline, []float64) {
	month := func(b market.Bar) int {
		t := time.Unix(b.Time, 0).In(vnTime)
		return t.Year()*12 + int(t.Month())
	}
	months := 1
	for i := 1; i < len(bars); i++ {
		if month(bars[i]) != month(bars[i-1]) {
			months++
		}
	}
	instalment := capital / float64(months)

	c := &baselineCurve{capital: capital, peak: capital}
	var cash, shares, fees, unfunded = 0.0, 0.0, 0.0, capital
	for i, b := range bars {
		if i == 0 || month(b) != month(bars[i-1]) {
			add := instalment
			if unfunded-add < instalment/2 { // last instalment takes any rounding
				add = unfunded
			}
			unfunded -= add
			cash += add
			if qty := lotsFor(cash, b.Close); qty > 0 {
				fee := qty * b.Close * order.FeeRate
				cash -= qty*b.Close + fee
				shares += qty
				fees += fee
			}
		}
		c.mark(unfunded + cash + shares*b.Close)
	}
	last := bars[len(bars)-1].Close
	return c.result(unfunded+cash+shares*last, fees), c.points
}

// vnTime: month boundaries for DCA are Vietnamese calendar months.
var vnTime = time.FixedZone("ICT", 7*3600)

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
	return withBaselines(t.result(last.Time, last.Close), bars[1:], startingCapital)
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
	return withBaselines(t.result(last.Time, last.Close), bars[1:], startingCapital)
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
