package valuation

import (
	"math"
	"sort"
	"time"
)

// Period is one fiscal year (Quarter 0) or quarter of a company's
// reported financials, as a FinancialsProvider returns them. VND, not
// per share. See phase-fundamentals-data.md.
type Period struct {
	Year      int
	Quarter   int     // 1-4, or 0 for the full fiscal year
	NetIncome float64 // attributable to parent shareholders
	Revenue   float64
	Equity    float64 // owners' equity at period end
	TotalDebt float64 // short + long-term borrowings at period end
	Shares    float64 // shares outstanding at period end
	// PublishedAt is when the market could first see these numbers.
	// Vendors that don't carry a release date use FilingDeadline.
	PublishedAt time.Time
}

// Dividend is one corporate event with an ex-date: a cash dividend
// (Cash, VND per share), a stock dividend / bonus issue (StockRatio,
// new shares per old share: 0.15 for 100:15), or both.
type Dividend struct {
	ExDate     time.Time
	Cash       float64
	StockRatio float64
}

// Close is one daily closing price.
type Close struct {
	Time  time.Time
	Price float64
}

var vnLoc = time.FixedZone("ICT", 7*3600)

// End is the last day of the period.
func (p Period) End() time.Time {
	month := 12
	if p.Quarter > 0 {
		month = 3 * p.Quarter
	}
	return time.Date(p.Year, time.Month(month)+1, 0, 0, 0, 0, 0, vnLoc)
}

// FilingDeadline is when a period's report must be public under
// Circular 96/2020/TT-BTC: 45 days after quarter end (consolidated
// parents; 30 for the rest) and 90 days after year end for the audited
// annual report. Erring late delays a signal; erring early would be
// look-ahead bias (phase-fundamentals-data.md "Publication dates").
func FilingDeadline(year, quarter int) time.Time {
	end := Period{Year: year, Quarter: quarter}.End()
	if quarter == 0 {
		return end.AddDate(0, 0, 90)
	}
	return end.AddDate(0, 0, 45)
}

// AsOfOptions: AdjustedCloses says whether the closes passed to AsOf
// are back-adjusted for stock dividends (phase-valuation.md open
// question 2). Per-share history is put on the same basis as the price
// it's divided into, so the P/E and yield series stay consistent either
// way.
type AsOfOptions struct {
	AdjustedCloses bool
}

// Snapshot is what a company looked like on one date, from data
// published on or before it. Fundamentals feeds Evaluate; the rest feeds
// the long-horizon factors (phase-holding-horizon.md H2). Zero means
// unknown, as in Fundamentals.
type Snapshot struct {
	Fundamentals
	DebtToEquity float64 // latest period; meaningless for banks
	ROEStdDev    float64 // percentage points, last 3 fiscal years
}

const (
	growthYears    = 3
	historyYears   = 5
	minHistMonths  = 36 // a P/E band or high yield needs 3+ years of points
	ttmQuarters    = 4
	consistencyYrs = 3
)

// AsOf derives point-in-time valuation inputs at t. Only periods with
// PublishedAt <= t and closes/dividends dated <= t are used, so the same
// call serves today's card (t = now) and a backtest bar.
func AsOf(periods []Period, divs []Dividend, closes []Close, t time.Time, opt AsOfOptions) Snapshot {
	h := newHistory(periods, divs, t)
	var s Snapshot

	price, ok := lastClose(closes, t)
	if !ok {
		return s
	}
	s.Price = price
	s.EPS = h.ttmEPS(t, 1)
	s.CashDPS = h.cashDPS(t, 1)
	if latest, ok := h.latest(t); ok && latest.Shares > 0 {
		s.BVPS = latest.Equity / latest.Shares / h.splitFactor(latest.End(), t)
		if latest.Equity > 0 {
			s.DebtToEquity = latest.TotalDebt / latest.Equity
		}
	}
	s.ROE = h.ttmROE(t)
	s.GrowthPct = h.epsGrowth(t)
	s.ROEStdDev = h.roeStdDev()
	s.RecentLoss = h.recentLoss()

	// Monthly samples over the last historyYears: P/E and cash yield,
	// each with price and per-share figure on the same share basis.
	var pes, yields []float64
	from := t.AddDate(-historyYears, 0, 0)
	for _, c := range monthEnds(closes, from, t) {
		f := 1.0
		if opt.AdjustedCloses {
			f = h.splitFactor(c.Time, t)
		}
		if eps := h.ttmEPS(c.Time, f); eps > 0 {
			pes = append(pes, c.Price/eps)
		}
		yields = append(yields, h.cashDPS(c.Time, f)/c.Price)
	}
	if len(pes) >= minHistMonths {
		s.AvgHistPE, s.StdHistPE = meanStd(pes)
	}
	if len(yields) >= minHistMonths {
		for _, y := range yields {
			s.HighYield = math.Max(s.HighYield, y)
		}
	}
	return s
}

type history struct {
	annual   []Period // visible at t, oldest first
	quarters []Period
	divs     []Dividend // ex-date <= t, oldest first
}

func newHistory(periods []Period, divs []Dividend, t time.Time) history {
	var h history
	for _, p := range periods {
		if p.PublishedAt.IsZero() || p.PublishedAt.After(t) {
			continue
		}
		if p.Quarter == 0 {
			h.annual = append(h.annual, p)
		} else {
			h.quarters = append(h.quarters, p)
		}
	}
	byEnd := func(ps []Period) {
		sort.Slice(ps, func(i, j int) bool { return ps[i].End().Before(ps[j].End()) })
	}
	byEnd(h.annual)
	byEnd(h.quarters)
	for _, d := range divs {
		if !d.ExDate.After(t) {
			h.divs = append(h.divs, d)
		}
	}
	sort.Slice(h.divs, func(i, j int) bool { return h.divs[i].ExDate.Before(h.divs[j].ExDate) })
	return h
}

// visible filters ps to those published on or before d.
func visible(ps []Period, d time.Time) []Period {
	var out []Period
	for _, p := range ps {
		if !p.PublishedAt.After(d) {
			out = append(out, p)
		}
	}
	return out
}

// latest is the most recent visible period, quarterly or annual.
func (h history) latest(d time.Time) (Period, bool) {
	var best Period
	found := false
	for _, ps := range [][]Period{visible(h.annual, d), visible(h.quarters, d)} {
		if n := len(ps); n > 0 && (!found || ps[n-1].End().After(best.End())) {
			best, found = ps[n-1], true
		}
	}
	return best, found
}

// splitFactor is how many shares one share held at d had become by t,
// from stock dividends and bonus issues with ex-dates in (d, t].
func (h history) splitFactor(d, t time.Time) float64 {
	f := 1.0
	for _, e := range h.divs {
		if e.ExDate.After(d) && !e.ExDate.After(t) && e.StockRatio > 0 {
			f *= 1 + e.StockRatio
		}
	}
	return f
}

// lastQuarters returns the last n visible quarters at d if they are
// consecutive, else nil.
func (h history) lastQuarters(d time.Time, n int) []Period {
	qs := visible(h.quarters, d)
	if len(qs) < n {
		return nil
	}
	qs = qs[len(qs)-n:]
	for i := 1; i < n; i++ {
		prev, cur := qs[i-1], qs[i]
		if cur.Year*4+cur.Quarter != prev.Year*4+prev.Quarter+1 {
			return nil
		}
	}
	return qs
}

// ttmNetIncome is trailing-12-month net income at d and the period whose
// share count it should be divided by: the last four quarters, or the
// latest fiscal year when quarters are missing.
func (h history) ttmNetIncome(d time.Time) (float64, Period, bool) {
	if qs := h.lastQuarters(d, ttmQuarters); qs != nil {
		var ni float64
		for _, q := range qs {
			ni += q.NetIncome
		}
		return ni, qs[len(qs)-1], true
	}
	if ys := visible(h.annual, d); len(ys) > 0 {
		return ys[len(ys)-1].NetIncome, ys[len(ys)-1], true
	}
	return 0, Period{}, false
}

// ttmEPS at d on d's share basis (a bonus issue between the period end
// and d is already in d's price), divided by factor to restate it onto a
// later basis.
func (h history) ttmEPS(d time.Time, factor float64) float64 {
	ni, p, ok := h.ttmNetIncome(d)
	if !ok || p.Shares <= 0 {
		return 0
	}
	return ni / p.Shares / h.splitFactor(p.End(), d) / factor
}

// cashDPS is cash dividends per share with ex-date in the year to d.
func (h history) cashDPS(d time.Time, factor float64) float64 {
	var sum float64
	from := d.AddDate(-1, 0, 0)
	for _, e := range h.divs {
		if e.ExDate.After(from) && !e.ExDate.After(d) {
			// Restate each payment onto d's share basis first.
			sum += e.Cash / h.splitFactor(e.ExDate, d)
		}
	}
	return sum / factor
}

func (h history) ttmROE(d time.Time) float64 {
	ni, last, ok := h.ttmNetIncome(d)
	if !ok {
		return 0
	}
	// Average equity: period end and one year earlier, when visible.
	eq, n := last.Equity, 1.0
	for _, p := range append(visible(h.annual, d), visible(h.quarters, d)...) {
		if p.Year == last.Year-1 && p.Quarter == last.Quarter {
			eq, n = eq+p.Equity, 2
			break
		}
	}
	if eq <= 0 {
		return 0
	}
	return ni / (eq / n) * 100
}

// annualEPS is each visible fiscal year's EPS restated onto t's share
// basis, so a bonus issue doesn't read as falling earnings.
func (h history) annualEPS(t time.Time) map[int]float64 {
	out := map[int]float64{}
	for _, y := range h.annual {
		if y.Shares > 0 {
			out[y.Year] = y.NetIncome / y.Shares / h.splitFactor(y.End(), t)
		}
	}
	return out
}

// epsGrowth is the growthYears-year EPS CAGR in percent, or 0 when the
// start or end year is missing or not profitable (CAGR is undefined
// across a loss). Evaluate applies the MaxGrowthPct cap.
func (h history) epsGrowth(t time.Time) float64 {
	if len(h.annual) == 0 {
		return 0
	}
	eps := h.annualEPS(t)
	end := h.annual[len(h.annual)-1].Year
	a, b := eps[end-growthYears], eps[end]
	if a <= 0 || b <= 0 {
		return 0
	}
	return (math.Pow(b/a, 1.0/growthYears) - 1) * 100
}

func (h history) roeStdDev() float64 {
	if len(h.annual) < consistencyYrs {
		return 0
	}
	var roes []float64
	for i := len(h.annual) - consistencyYrs; i < len(h.annual); i++ {
		y := h.annual[i]
		eq := y.Equity
		if i > 0 && h.annual[i-1].Year == y.Year-1 {
			eq = (eq + h.annual[i-1].Equity) / 2
		}
		if eq <= 0 {
			return 0
		}
		roes = append(roes, y.NetIncome/eq*100)
	}
	_, sd := meanStd(roes)
	return sd
}

// recentLoss reports a loss in any of the last minProfitableYears
// fiscal years on record.
func (h history) recentLoss() bool {
	if len(h.annual) == 0 {
		return false
	}
	end := h.annual[len(h.annual)-1].Year
	for _, y := range h.annual {
		if y.Year > end-minProfitableYears && y.NetIncome <= 0 {
			return true
		}
	}
	return false
}

func lastClose(closes []Close, t time.Time) (float64, bool) {
	var best Close
	for _, c := range closes {
		if !c.Time.After(t) && c.Price > 0 && (best.Time.IsZero() || c.Time.After(best.Time)) {
			best = c
		}
	}
	return best.Price, !best.Time.IsZero()
}

// monthEnds is the last close of each calendar month in (from, t].
func monthEnds(closes []Close, from, t time.Time) []Close {
	last := map[[2]int]Close{}
	for _, c := range closes {
		if !c.Time.After(from) || c.Time.After(t) || c.Price <= 0 {
			continue
		}
		v := c.Time.In(vnLoc)
		k := [2]int{v.Year(), int(v.Month())}
		if prev, ok := last[k]; !ok || c.Time.After(prev.Time) {
			last[k] = c
		}
	}
	out := make([]Close, 0, len(last))
	for _, c := range last {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

func meanStd(xs []float64) (float64, float64) {
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(ss / float64(len(xs)))
}
