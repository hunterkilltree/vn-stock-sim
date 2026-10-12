package regime

import (
	"fmt"
	"math"
	"sort"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// Thresholds; each one is a common convention, explained in
// phase-market-risk.md and in the Signal.Rule text the card shows.
const (
	trendPeriod   = 200
	slopeLookback = 20 // sessions over which the 200-day average's slope is read

	drawdownCaution = -10.0 // a "correction"
	drawdownRisk    = -20.0 // a "bear market"

	volWindow       = 20
	volLookback     = 252
	volRatioCaution = 1.25
	volRatioRisk    = 1.75

	breadthOK   = 60.0
	breadthRisk = 40.0

	// Level cut-offs on the summed score (caution 1, risk 2).
	cautionScore  = 2
	highRiskScore = 4
)

// Input: daily bars, oldest first. Universe maps symbol -> its bars, for
// breadth; it can be empty.
type Input struct {
	IndexName string
	Index     []market.Bar
	Universe  map[string][]market.Bar
}

// Compute is the whole regime read: pure, so it's testable on synthetic
// series. Returns ok=false when the index history is too short to judge.
func Compute(in Input) (Regime, bool) {
	bars := in.Index
	if len(bars) < trendPeriod+slopeLookback {
		return Regime{}, false
	}
	last := bars[len(bars)-1]
	r := Regime{IndexName: in.IndexName, IndexValue: last.Close, AsOf: last.Time, Universe: len(in.Universe), Source: "rule-based"}

	trend := trendSignal(bars)
	r.Signals = append(r.Signals, trend, drawdownSignal(bars), volatilitySignal(bars), breadthSignal(in.Universe),
		Signal{Key: "valuation", Label: "Định giá thị trường", Value: "Chưa có dữ liệu", Zone: ZoneNoData,
			Rule: "P/E trung vị so với lịch sử 5–10 năm",
			Why:  "Thị trường đắt (như đầu 2007) giảm sâu hơn khi có cú sốc. Cần lịch sử P/E (phase-valuation-backtest.md)."},
		Signal{Key: "macro", Label: "Lãi suất & tín dụng", Value: "Chưa có dữ liệu", Zone: ZoneNoData,
			Rule: "Xu hướng lãi suất điều hành/liên ngân hàng, lạm phát, tăng trưởng tín dụng, dư nợ margin",
			Why:  "Năm 2008 và 2022 đều bắt đầu khi lãi suất tăng và tín dụng bị siết."},
	)

	for _, s := range r.Signals {
		switch s.Zone {
		case ZoneCaution:
			r.Score++
			r.MaxScore += 2
		case ZoneRisk:
			r.Score += 2
			r.MaxScore += 2
		case ZoneOK:
			r.MaxScore += 2
		}
	}
	switch {
	case r.Score >= highRiskScore:
		r.Level = LevelHighRisk
	case r.Score >= cautionScore:
		r.Level = LevelCaution
	default:
		r.Level = LevelNormal
	}
	// Knock-out: the index under a falling 200-day average is never
	// "normal", whatever the other signals say.
	if trend.Zone == ZoneRisk && r.Level == LevelNormal {
		r.Level = LevelCaution
		r.Knockout = "VN-Index nằm dưới đường trung bình 200 phiên đang đi xuống"
	}
	r.Limits = LimitsFor(r.Level)
	return r, true
}

// LimitsFor: risk per trade halves, then quarters, as conditions worsen;
// the exposure cap and the monthly loss lock tighten with it. The
// "normal" row matches phase-risk-rules.md's defaults (1%, 6%).
func LimitsFor(level string) Limits {
	switch level {
	case LevelHighRisk:
		return Limits{ExposureCapPct: 40, RiskPerTradePct: 0.25, LossLockPct: 3}
	case LevelCaution:
		return Limits{ExposureCapPct: 70, RiskPerTradePct: 0.5, LossLockPct: 4}
	default:
		return Limits{ExposureCapPct: 100, RiskPerTradePct: 1, LossLockPct: 6}
	}
}

func trendSignal(bars []market.Bar) Signal {
	s := Signal{Key: "trend", Label: "Xu hướng dài hạn",
		Rule: "VN-Index so với đường trung bình 200 phiên (SMA 200) và độ dốc của nó",
		Why:  "Các đợt giảm sâu (2008, 2022) đều bắt đầu khi chỉ số thủng SMA 200 và đường này quay đầu đi xuống."}
	sma := market.SMA(bars, trendPeriod)
	price := bars[len(bars)-1].Close
	cur := sma[len(sma)-1].Value
	prev := sma[len(sma)-1-slopeLookback].Value
	above, rising := price > cur, cur > prev
	diff := (price/cur - 1) * 100
	switch {
	case above && rising:
		s.Zone, s.Value = ZoneOK, fmt.Sprintf("Trên SMA 200 (%+.1f%%), SMA 200 đi lên", diff)
	case !above && !rising:
		s.Zone, s.Value = ZoneRisk, fmt.Sprintf("Dưới SMA 200 (%+.1f%%), SMA 200 đi xuống", diff)
	case above:
		s.Zone, s.Value = ZoneCaution, fmt.Sprintf("Trên SMA 200 (%+.1f%%) nhưng SMA 200 đi xuống", diff)
	default:
		s.Zone, s.Value = ZoneCaution, fmt.Sprintf("Dưới SMA 200 (%+.1f%%), SMA 200 vẫn đi lên", diff)
	}
	return s
}

func drawdownSignal(bars []market.Bar) Signal {
	s := Signal{Key: "drawdown", Label: "Mức giảm từ đỉnh 52 tuần",
		Rule: "Giảm hơn 10% là điều chỉnh, hơn 20% là thị trường giá xuống",
		Why:  "Thị trường đã giảm sâu thường biến động mạnh hơn và có thể còn giảm tiếp."}
	from := len(bars) - volLookback
	if from < 0 {
		from = 0
	}
	var hi float64
	for _, b := range bars[from:] {
		hi = math.Max(hi, b.High)
	}
	dd := (bars[len(bars)-1].Close/hi - 1) * 100
	s.Value = fmt.Sprintf("%.1f%% so với đỉnh", dd)
	switch {
	case dd <= drawdownRisk:
		s.Zone = ZoneRisk
	case dd <= drawdownCaution:
		s.Zone = ZoneCaution
	default:
		s.Zone = ZoneOK
	}
	return s
}

func volatilitySignal(bars []market.Bar) Signal {
	s := Signal{Key: "volatility", Label: "Biến động",
		Rule: "Biến động 20 phiên so với mức trung vị 1 năm",
		Why:  "Biến động tăng vọt thường đi kèm các đợt bán tháo; khi đó nên giảm quy mô vị thế."}
	cur := annualVol(bars, len(bars), volWindow)
	var hist []float64
	start := len(bars) - volLookback
	if start < volWindow+1 {
		start = volWindow + 1
	}
	for end := start; end <= len(bars); end++ {
		if v := annualVol(bars, end, volWindow); v > 0 {
			hist = append(hist, v)
		}
	}
	if cur <= 0 || len(hist) < 20 {
		s.Zone, s.Value = ZoneNoData, "Chưa đủ dữ liệu"
		return s
	}
	sort.Float64s(hist)
	med := hist[len(hist)/2]
	ratio := cur / med
	s.Value = fmt.Sprintf("%.0f%%/năm, gấp %.1f lần bình thường", cur, ratio)
	switch {
	case ratio >= volRatioRisk:
		s.Zone = ZoneRisk
	case ratio >= volRatioCaution:
		s.Zone = ZoneCaution
	default:
		s.Zone = ZoneOK
	}
	return s
}

func breadthSignal(universe map[string][]market.Bar) Signal {
	s := Signal{Key: "breadth", Label: "Độ rộng thị trường",
		Rule: "Tỷ lệ cổ phiếu nằm trên SMA 200 của chính nó; số mã lập đỉnh/đáy 52 tuần",
		Why:  "Khi chỉ số còn tăng nhờ vài mã lớn nhưng đa số cổ phiếu đã giảm, xu hướng thường yếu đi."}
	var above, total, highs, lows int
	for _, bars := range universe {
		if len(bars) < trendPeriod {
			continue
		}
		total++
		sma := market.SMA(bars, trendPeriod)
		last := bars[len(bars)-1].Close
		if last > sma[len(sma)-1].Value {
			above++
		}
		from := len(bars) - volLookback
		if from < 0 {
			from = 0
		}
		hi, lo := 0.0, math.Inf(1)
		for _, b := range bars[from:] {
			hi, lo = math.Max(hi, b.Close), math.Min(lo, b.Close)
		}
		if last >= hi {
			highs++
		} else if last <= lo {
			lows++
		}
	}
	if total < 5 {
		s.Zone, s.Value = ZoneNoData, "Chưa đủ dữ liệu"
		return s
	}
	pct := float64(above) / float64(total) * 100
	s.Value = fmt.Sprintf("%.0f%% trên SMA 200 (%d/%d mã); %d đỉnh, %d đáy 52 tuần", pct, above, total, highs, lows)
	switch {
	case pct < breadthRisk:
		s.Zone = ZoneRisk
	case pct < breadthOK:
		s.Zone = ZoneCaution
	default:
		s.Zone = ZoneOK
	}
	return s
}

// annualVol: annualised stdev (percent) of the n daily log returns ending
// at bars[end-1].
func annualVol(bars []market.Bar, end, n int) float64 {
	if end-n < 1 {
		return 0
	}
	rets := make([]float64, 0, n)
	for i := end - n; i < end; i++ {
		if bars[i-1].Close > 0 && bars[i].Close > 0 {
			rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
		}
	}
	if len(rets) < 2 {
		return 0
	}
	var mean float64
	for _, r := range rets {
		mean += r
	}
	mean /= float64(len(rets))
	var ss float64
	for _, r := range rets {
		ss += (r - mean) * (r - mean)
	}
	return math.Sqrt(ss/float64(len(rets)-1)) * math.Sqrt(252) * 100
}
