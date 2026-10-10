package stress

import (
	"math"
	"sort"
	"strconv"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

const (
	betaWindow     = 252 // sessions of daily returns behind each beta
	minBetaSamples = 60
	maxBeta        = 3.0

	episodeThreshold = -20.0 // a VN-Index fall at least this deep is an episode
	maxEpisodes      = 3

	// minCrisisBeta: shocks use max(beta, 1). In a crash correlations go to
	// one -- in 2007-2009 almost every VN stock fell with the index -- so a
	// low beta measured in calm markets would understate the loss.
	minCrisisBeta = 1.0

	// floorTolerance: a close within this many percentage points of the
	// band's limit counts as a floor session (ticks round the floor up).
	floorTolerance = 0.5
)

// Hypothetical shocks. The 2008 row is approximate (VN-Index ~1,170 in
// Mar 2007 to ~235 in Feb 2009) and labelled as such; it predates the
// app's price history, so it can only be an estimate through beta.
var shocks = []struct {
	key, label, note string
	indexPct         float64
}{
	{"shock_20", "VN-Index giảm 20%", "Một đợt điều chỉnh mạnh, như 2018.", -20},
	{"shock_40", "VN-Index giảm 40%", "Gần mức giảm của năm 2022.", -40},
	{"like_2008", "Lặp lại 2007–2009 (≈ −80%)", "Số liệu xấp xỉ; giai đoạn này nằm ngoài dữ liệu giá của ứng dụng nên chỉ ước tính qua beta.", -80},
}

// Holding is one stock position with its daily bars (oldest first) and
// its exchange's daily price band (0.07 HOSE, 0.10 HNX, 0.15 UPCOM).
type Holding struct {
	Symbol string
	Value  float64
	Band   float64
	Bars   []market.Bar
}

type Input struct {
	Cash     float64
	Holdings []Holding
	Index    []market.Bar
}

func Compute(in Input) Result {
	r := Result{Cash: in.Cash, Source: "rule-based"}
	for _, h := range in.Holdings {
		r.StockValue += h.Value
	}
	r.Equity = r.Cash + r.StockValue

	betas := make([]float64, len(in.Holdings))
	for i, h := range in.Holdings {
		b, n := beta(h.Bars, in.Index)
		betas[i] = b
		r.Betas = append(r.Betas, HoldingBeta{Symbol: h.Symbol, Value: h.Value, Beta: round2(b), Samples: n, CrisisBeta: round2(math.Max(b, minCrisisBeta))})
	}

	for i, ep := range episodes(in.Index) {
		sc := Scenario{Key: "episode_" + strconv.Itoa(i+1), Kind: KindHistorical, From: ep.from, To: ep.to, IndexChangePct: round1(ep.pct),
			Label: "Đợt giảm thực tế của VN-Index"}
		for j, h := range in.Holdings {
			imp := HoldingImpact{Symbol: h.Symbol}
			if pct, ok := pathReturn(h.Bars, ep.from, ep.to); ok {
				imp.ChangePct = pct
				imp.FloorDays, imp.LongestFloors = floorSessions(h.Bars, ep.from, ep.to, h.Band)
			} else {
				imp.ChangePct, imp.Estimated = shocked(betas[j], ep.pct), true
			}
			addImpact(&sc, imp, h.Value)
		}
		finish(&sc, r.Equity)
		r.Scenarios = append(r.Scenarios, sc)
	}

	for _, s := range shocks {
		sc := Scenario{Key: s.key, Label: s.label, Kind: KindHypothetical, IndexChangePct: s.indexPct, Note: s.note}
		for j, h := range in.Holdings {
			addImpact(&sc, HoldingImpact{Symbol: h.Symbol, ChangePct: shocked(betas[j], s.indexPct), Estimated: true}, h.Value)
		}
		finish(&sc, r.Equity)
		r.Scenarios = append(r.Scenarios, sc)
	}
	return r
}

func addImpact(sc *Scenario, imp HoldingImpact, value float64) {
	imp.ChangePct = round1(imp.ChangePct)
	imp.LossVND = math.Round(-value * imp.ChangePct / 100)
	sc.LossVND += imp.LossVND
	sc.Holdings = append(sc.Holdings, imp)
}

// finish sorts the worst holdings first and expresses the loss against
// total equity, so cash visibly cushions it.
func finish(sc *Scenario, equity float64) {
	sort.SliceStable(sc.Holdings, func(a, b int) bool { return sc.Holdings[a].LossVND > sc.Holdings[b].LossVND })
	if equity > 0 {
		sc.LossPct = round1(sc.LossVND / equity * 100)
	}
}

// shocked applies an index move through the crisis beta; a stock can't
// lose more than everything.
func shocked(beta, indexPct float64) float64 {
	return math.Max(-100, math.Max(beta, minCrisisBeta)*indexPct)
}

type episode struct {
	from, to int64
	pct      float64
}

// episodes finds VN-Index falls of at least episodeThreshold: from a
// running peak to the lowest close before the index regains that peak.
// The deepest maxEpisodes are returned, most recent first.
func episodes(bars []market.Bar) []episode {
	var out []episode
	peakIdx := 0
	troughIdx := -1
	for i := 1; i <= len(bars); i++ {
		if i == len(bars) || bars[i].Close >= bars[peakIdx].Close {
			if troughIdx >= 0 {
				pct := (bars[troughIdx].Close/bars[peakIdx].Close - 1) * 100
				if pct <= episodeThreshold {
					out = append(out, episode{bars[peakIdx].Time, bars[troughIdx].Time, pct})
				}
			}
			if i == len(bars) {
				break
			}
			peakIdx, troughIdx = i, -1
			continue
		}
		if troughIdx < 0 || bars[i].Close < bars[troughIdx].Close {
			troughIdx = i
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].pct < out[b].pct })
	if len(out) > maxEpisodes {
		out = out[:maxEpisodes]
	}
	sort.Slice(out, func(a, b int) bool { return out[a].from > out[b].from })
	return out
}

// pathReturn is the stock's percent change from the close at (or just
// before) from to the close at (or just before) to; false if its history
// doesn't reach back to from.
func pathReturn(bars []market.Bar, from, to int64) (float64, bool) {
	if len(bars) == 0 || bars[0].Time > from {
		return 0, false
	}
	a, b := closeAt(bars, from), closeAt(bars, to)
	if a <= 0 {
		return 0, false
	}
	return (b/a - 1) * 100, true
}

func closeAt(bars []market.Bar, t int64) float64 {
	i := sort.Search(len(bars), func(i int) bool { return bars[i].Time > t })
	if i == 0 {
		return 0
	}
	return bars[i-1].Close
}

// floorSessions counts sessions in (from, to] that closed at about the
// floor price (down by the band, within floorTolerance), and the longest
// consecutive run -- days a stop-loss likely couldn't fill
// (phase-risk-rules.md rule 8).
func floorSessions(bars []market.Bar, from, to int64, band float64) (days, longest int) {
	limit := -band*100 + floorTolerance
	run := 0
	for i := 1; i < len(bars); i++ {
		if bars[i].Time <= from || bars[i].Time > to || bars[i-1].Close <= 0 {
			continue
		}
		if (bars[i].Close/bars[i-1].Close-1)*100 <= limit {
			days++
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return days, longest
}

// beta = cov(stock, index) / var(index) over the last betaWindow daily log
// returns on dates both series have. Too few samples -> 1 (assume it
// moves with the market), clamped to [0, maxBeta].
func beta(stock, index []market.Bar) (float64, int) {
	idx := make(map[int64]float64, len(index))
	for _, b := range index {
		idx[b.Time] = b.Close
	}
	var xs, ys []float64
	for i := 1; i < len(stock); i++ {
		p0, ok0 := idx[stock[i-1].Time]
		p1, ok1 := idx[stock[i].Time]
		if !ok0 || !ok1 || p0 <= 0 || p1 <= 0 || stock[i-1].Close <= 0 || stock[i].Close <= 0 {
			continue
		}
		xs = append(xs, math.Log(p1/p0))
		ys = append(ys, math.Log(stock[i].Close/stock[i-1].Close))
	}
	if len(xs) > betaWindow {
		xs, ys = xs[len(xs)-betaWindow:], ys[len(ys)-betaWindow:]
	}
	if len(xs) < minBetaSamples {
		return 1, 0
	}
	var mx, my float64
	for i := range xs {
		mx += xs[i]
		my += ys[i]
	}
	mx /= float64(len(xs))
	my /= float64(len(ys))
	var cov, vx float64
	for i := range xs {
		cov += (xs[i] - mx) * (ys[i] - my)
		vx += (xs[i] - mx) * (xs[i] - mx)
	}
	if vx == 0 {
		return 1, 0
	}
	return math.Max(0, math.Min(maxBeta, cov/vx)), len(xs)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
