package rating

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// Thresholds and weights for the holding-horizon verdicts; every number
// is explained in phase-holding-horizon.md.
const (
	suitableCutoff = 65.0
	watchCutoff    = 45.0

	minTradedValue = 1e9 // VND/day; below this you can't reliably exit

	costOfEquity  = 13.0 // % -- VN 10-year yield ~3% + equity risk premium ~10%
	longRunGrowth = 5.0  // %

	day = 24 * 60 * 60
)

// HorizonInput is everything ComputeHorizons needs. Bars and IndexBars
// are daily, oldest first; IndexBars may be empty (relative measures then
// fall back to absolute ones).
type HorizonInput struct {
	Detail         symbol.Detail
	Bars           []market.Bar
	IndexBars      []market.Bar
	SectorMedianPE float64
}

// lerp maps v onto 0..100 between bad and good (either order).
func lerp(v, bad, good float64) float64 {
	t := (v - bad) / (good - bad)
	return math.Round(math.Max(0, math.Min(1, t))*100*10) / 10
}

func ComputeHorizons(in HorizonInput) *Horizons {
	if len(in.Bars) < 60 {
		return nil
	}
	return &Horizons{Medium: medium(in), Long: long(in)}
}

func medium(in HorizonInput) Horizon {
	bars, d := in.Bars, in.Detail
	last := bars[len(bars)-1]
	price := last.Close
	now := last.Time
	var fs []Factor

	// Trend structure: above SMA 50 and 200, SMA 50 rising, 50 over 200.
	trend := Factor{Key: "trend", Label: "Xu hướng", Weight: 0.25,
		Why: "Cổ phiếu tăng tốt trong 3–6 tháng thường đã nằm trong xu hướng tăng."}
	s50 := market.SMA(bars, 50)
	s200 := market.SMA(bars, 200)
	if len(s50) > 20 {
		pts, n := 0, 0
		var parts []string
		check := func(ok bool, yes, no string) {
			n++
			if ok {
				pts++
				parts = append(parts, yes)
			} else {
				parts = append(parts, no)
			}
		}
		cur50 := s50[len(s50)-1].Value
		check(price > cur50, "trên SMA 50", "dưới SMA 50")
		check(cur50 > s50[len(s50)-21].Value, "SMA 50 đi lên", "SMA 50 đi xuống")
		if len(s200) > 0 {
			cur200 := s200[len(s200)-1].Value
			check(price > cur200, "trên SMA 200", "dưới SMA 200")
			check(cur50 > cur200, "SMA 50 > SMA 200", "SMA 50 < SMA 200")
		}
		trend.Available = true
		trend.Score = math.Round(float64(pts) / float64(n) * 100)
		trend.Value = fmt.Sprintf("%d/%d tiêu chí", pts, n)
		trend.Why += " Hiện: " + strings.Join(parts, ", ") + "."
	}
	fs = append(fs, trend)

	// Relative strength: 6-month return skipping the last month, vs VN-Index.
	rs := Factor{Key: "relative_strength", Label: "Sức mạnh tương đối 6 tháng", Weight: 0.25,
		Why: "Động lượng 3–12 tháng là một trong những yếu tố sinh lời được kiểm chứng nhiều nhất (Jegadeesh & Titman)."}
	if r, ok := periodReturn(bars, now-182*day, now-30*day); ok {
		rs.Available = true
		if ir, ok := periodReturn(in.IndexBars, now-182*day, now-30*day); ok {
			rs.Score = lerp(r-ir, -15, 15)
			rs.Value = fmt.Sprintf("%+.1f điểm so với VN-Index", r-ir)
		} else {
			rs.Score = lerp(r, -15, 15)
			rs.Value = fmt.Sprintf("%+.1f%% (tuyệt đối)", r)
		}
	}
	fs = append(fs, rs)

	// Distance from the 52-week high.
	nh := Factor{Key: "near_high", Label: "So với đỉnh 52 tuần", Weight: 0.10,
		Why: "Cổ phiếu gần đỉnh 52 tuần thường tiếp tục đi lên (hiệu ứng đỉnh 52 tuần)."}
	if hi := highSince(bars, now-365*day); hi > 0 {
		ratio := price / hi * 100
		nh.Available, nh.Score, nh.Value = true, lerp(ratio, 70, 95), fmt.Sprintf("%.0f%% đỉnh", ratio)
	}
	fs = append(fs, nh)

	// Volatility.
	vol := annualVol(bars, 60)
	fs = append(fs, Factor{Key: "volatility", Label: "Biến động 60 phiên", Weight: 0.15, Available: vol > 0,
		Score: lerp(vol, 60, 25), Value: fmt.Sprintf("%.0f%%/năm", vol),
		Why: "Biến động thấp dễ nắm giữ hơn và quyết định khối lượng vị thế nên mua."})

	// Liquidity.
	tv := avgTradedValue(bars, 20)
	fs = append(fs, Factor{Key: "liquidity", Label: "Thanh khoản 20 phiên", Weight: 0.10, Available: tv > 0,
		Score: lerp(tv, 1e9, 20e9), Value: fmt.Sprintf("%.1f tỷ/phiên", tv/1e9),
		Why: "Phải mua vào và bán ra được dễ dàng."})

	// Valuation sanity vs sector.
	fs = append(fs, relativePE(d, in.SectorMedianPE, 0.15, 2.0, 1.0,
		"Tránh trả bất kỳ giá nào chỉ vì giá đang tăng."))

	h := score(fs)
	h.Risk = Risk{VolatilityPct: round1(vol), MaxDrawdownPct: round1(maxDrawdown(bars, now-182*day)), DrawdownWindow: "6 tháng"}
	// Reference stop: the tighter of 2xATR(14) below price and just under
	// SMA 50 (only while price is above it).
	if atr := atr14(bars); atr > 0 {
		stop := price - 2*atr
		if len(s50) > 0 {
			if under := s50[len(s50)-1].Value - symbol.TickFor(d.Exchange, price); under < price && under > stop {
				stop = under
			}
		}
		if stop > 0 {
			stop = symbol.FloorToTick(stop, d.Exchange)
			h.Risk.StopLoss, h.Risk.StopPct = stop, round1((stop/price-1)*100)
		}
	}
	h.History = history(bars, in.IndexBars, []holdWindow{{"3 tháng", 91}, {"6 tháng", 182}})

	// Knock-outs (decision 3).
	if tv > 0 && tv < minTradedValue {
		h.Knockout, h.Verdict = "Thanh khoản dưới 1 tỷ đồng/phiên", VerdictAvoid
	} else if len(s200) > 0 && len(s50) > 20 && price < s200[len(s200)-1].Value && s50[len(s50)-1].Value < s50[len(s50)-21].Value && h.Verdict == VerdictSuitable {
		h.Knockout, h.Verdict = "Giá dưới SMA 200 và SMA 50 đang đi xuống: không đi ngược xu hướng giảm", VerdictWatch
	}
	return h
}

func long(in HorizonInput) Horizon {
	bars, d := in.Bars, in.Detail
	now := bars[len(bars)-1].Time
	var fs []Factor

	roe := Factor{Key: "roe", Label: "Hiệu quả sinh lời (ROE)", Weight: 0.20,
		Why: "Doanh nghiệp có ROE cao tạo ra lãi kép qua nhiều năm."}
	if d.ROE != 0 {
		roe.Available, roe.Score, roe.Value = true, lerp(d.ROE, 8, 20), fmt.Sprintf("%.1f%%", d.ROE)
	}
	fs = append(fs, roe)

	// Value vs quality: P/B against the justified P/B (decision 6).
	vq := Factor{Key: "value_vs_quality", Label: "P/B so với P/B hợp lý", Weight: 0.20,
		Why: fmt.Sprintf("P/B hợp lý ≈ (ROE − %.0f%%) / (%.0f%% − %.0f%%): doanh nghiệp tốt chỉ là khoản đầu tư tốt khi mua ở giá hợp lý.", longRunGrowth, costOfEquity, longRunGrowth)}
	if d.PBRatio > 0 && d.ROE != 0 {
		vq.Available = true
		if fair := (d.ROE - longRunGrowth) / (costOfEquity - longRunGrowth); fair > 0 {
			r := d.PBRatio / fair
			vq.Score, vq.Value = lerp(r, 1.5, 0.7), fmt.Sprintf("P/B %.1f, hợp lý %.1f", d.PBRatio, fair)
		} else {
			vq.Value = "ROE thấp hơn chi phí vốn"
		}
	}
	fs = append(fs, vq)

	fs = append(fs, relativePE(d, in.SectorMedianPE, 0.10, 2.0, 0.7, "Rẻ hay đắt so với các doanh nghiệp cùng ngành."))

	div := Factor{Key: "dividend", Label: "Tỷ suất cổ tức", Weight: 0.10, Available: d.EPS != 0 || d.DividendYield > 0,
		Score: lerp(d.DividendYield, 0, 5), Value: fmt.Sprintf("%.1f%%", d.DividendYield),
		Why: "Tiền mặt nhận về trong lúc chờ đợi; 5% ≈ lãi tiết kiệm."}
	fs = append(fs, div)

	size := Factor{Key: "size", Label: "Quy mô vốn hóa", Weight: 0.05,
		Why: "Cổ phiếu vốn hóa nhỏ ở Việt Nam rủi ro hơn về quản trị và thanh khoản khi giữ nhiều năm."}
	if d.MarketCap > 0 {
		size.Available, size.Score = true, lerp(float64(d.MarketCap), 1e12, 20e12)
		if d.MarketCap >= 1e12 {
			size.Value = fmt.Sprintf("%.1f nghìn tỷ", float64(d.MarketCap)/1e12)
		} else {
			size.Value = fmt.Sprintf("%.0f tỷ", float64(d.MarketCap)/1e9)
		}
	}
	fs = append(fs, size)

	// Long-run price behaviour: 3-year CAGR vs VN-Index, and 3-year drawdown.
	lr := Factor{Key: "long_run", Label: "Lịch sử giá 3 năm", Weight: 0.10,
		Why: "Cho biết nắm giữ cổ phiếu này trước đây đã trải qua những gì."}
	if r, ok := periodReturn(bars, now-3*365*day, now); ok {
		cagr := (math.Pow(1+r/100, 1.0/3) - 1) * 100
		dd := maxDrawdown(bars, now-3*365*day)
		perf := lerp(cagr, -5, 15)
		lr.Value = fmt.Sprintf("%+.1f%%/năm, giảm sâu nhất %.0f%%", cagr, dd)
		if ir, ok := periodReturn(in.IndexBars, now-3*365*day, now); ok {
			icagr := (math.Pow(1+ir/100, 1.0/3) - 1) * 100
			perf = lerp(cagr-icagr, -10, 10)
			lr.Value = fmt.Sprintf("%+.1f%%/năm (VN-Index %+.1f%%), giảm sâu nhất %.0f%%", cagr, icagr, dd)
		}
		lr.Available, lr.Score = true, math.Round((perf+lerp(dd, -60, -20))/2*10)/10
	}
	fs = append(fs, lr)

	// Waiting on H2 data (phase-holding-horizon.md decision 4).
	for _, f := range []Factor{
		{Key: "eps_growth", Label: "Tăng trưởng EPS 3 năm", Weight: 0.15, Why: "Qua nhiều năm, giá cổ phiếu đi theo lợi nhuận."},
		{Key: "health", Label: "Sức khỏe tài chính (D/E)", Weight: 0.05, Why: "Nợ vay biến một năm khó khăn thành rủi ro sống còn."},
		{Key: "consistency", Label: "Độ ổn định ROE", Weight: 0.05, Why: "Doanh nghiệp lãi đều đặn tốt hơn lãi đột biến một lần."},
	} {
		f.Value = "Thiếu dữ liệu"
		fs = append(fs, f)
	}

	h := score(fs)
	h.Risk = Risk{VolatilityPct: round1(annualVol(bars, 250)), MaxDrawdownPct: round1(maxDrawdown(bars, now-3*365*day)), DrawdownWindow: "3 năm"}
	h.History = history(bars, in.IndexBars, []holdWindow{{"1 năm", 365}, {"3 năm", 3 * 365}})

	if tv := avgTradedValue(bars, 20); tv > 0 && tv < minTradedValue {
		h.Knockout, h.Verdict = "Thanh khoản dưới 1 tỷ đồng/phiên", VerdictAvoid
	} else if d.EPS < 0 && h.Verdict == VerdictSuitable {
		h.Knockout, h.Verdict = "Doanh nghiệp đang lỗ", VerdictWatch
	}
	return h
}

func relativePE(d symbol.Detail, median, weight, bad, good float64, why string) Factor {
	f := Factor{Key: "relative_pe", Label: "P/E so với ngành", Weight: weight, Why: why}
	switch {
	case d.EPS < 0:
		f.Available, f.Value = true, "Đang lỗ"
	case d.PERatio > 0 && median > 0:
		r := d.PERatio / median
		f.Available, f.Score = true, lerp(r, bad, good)
		f.Value = fmt.Sprintf("P/E %.1f, ngành %.1f", d.PERatio, median)
	}
	return f
}

// score takes the weighted mean over available factors, re-normalising
// the weights (decision 1).
func score(fs []Factor) Horizon {
	var sum, w float64
	h := Horizon{Factors: fs}
	for _, f := range fs {
		if !f.Available {
			h.Partial = true
			continue
		}
		sum += f.Score * f.Weight
		w += f.Weight
	}
	if w > 0 {
		h.Score = math.Round(sum / w)
	}
	switch {
	case h.Score >= suitableCutoff:
		h.Verdict = VerdictSuitable
	case h.Score >= watchCutoff:
		h.Verdict = VerdictWatch
	default:
		h.Verdict = VerdictAvoid
	}
	return h
}

// closeAt is the last close at or before t.
func closeAt(bars []market.Bar, t int64) (float64, bool) {
	i := sort.Search(len(bars), func(i int) bool { return bars[i].Time > t })
	if i == 0 {
		return 0, false
	}
	return bars[i-1].Close, true
}

// periodReturn is the percent change from the close at from to the close
// at to; false when the bars don't reach back to from.
func periodReturn(bars []market.Bar, from, to int64) (float64, bool) {
	if len(bars) == 0 || bars[0].Time > from {
		return 0, false
	}
	a, ok1 := closeAt(bars, from)
	b, ok2 := closeAt(bars, to)
	if !ok1 || !ok2 || a <= 0 {
		return 0, false
	}
	return (b/a - 1) * 100, true
}

func highSince(bars []market.Bar, from int64) float64 {
	var hi float64
	for _, b := range bars {
		if b.Time >= from && b.High > hi {
			hi = b.High
		}
	}
	return hi
}

// maxDrawdown is the worst peak-to-trough close decline since from, as a
// negative percent.
func maxDrawdown(bars []market.Bar, from int64) float64 {
	var peak, worst float64
	for _, b := range bars {
		if b.Time < from {
			continue
		}
		if b.Close > peak {
			peak = b.Close
		}
		if peak > 0 {
			worst = math.Min(worst, (b.Close/peak-1)*100)
		}
	}
	return worst
}

// annualVol is the annualised standard deviation of the last n daily log
// returns, in percent.
func annualVol(bars []market.Bar, n int) float64 {
	if len(bars) < n+1 {
		n = len(bars) - 1
	}
	if n < 10 {
		return 0
	}
	rets := make([]float64, 0, n)
	for i := len(bars) - n; i < len(bars); i++ {
		if bars[i-1].Close > 0 && bars[i].Close > 0 {
			rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
		}
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

func avgTradedValue(bars []market.Bar, n int) float64 {
	if len(bars) < n {
		n = len(bars)
	}
	var sum float64
	for _, b := range bars[len(bars)-n:] {
		sum += b.Close * float64(b.Volume)
	}
	return sum / float64(n)
}

func atr14(bars []market.Bar) float64 {
	const n = 14
	if len(bars) < n+1 {
		return 0
	}
	var sum float64
	for i := len(bars) - n; i < len(bars); i++ {
		prev := bars[i-1].Close
		tr := math.Max(bars[i].High-bars[i].Low, math.Max(math.Abs(bars[i].High-prev), math.Abs(bars[i].Low-prev)))
		sum += tr
	}
	return sum / n
}

type holdWindow struct {
	label string
	days  int
}

// history: for every start bar with a full window after it, the return
// from holding `days` calendar days (decision: calendar time, not bar
// counts, so mock and VCI data mean the same thing).
func history(bars, index []market.Bar, windows []holdWindow) []HoldStats {
	var out []HoldStats
	for _, w := range windows {
		s, ok := holdStats(bars, w)
		if !ok {
			continue
		}
		if is, ok := holdStats(index, w); ok {
			s.Index = &is
		}
		out = append(out, s)
	}
	return out
}

const minSamples = 20

func holdStats(bars []market.Bar, w holdWindow) (HoldStats, bool) {
	span := int64(w.days) * day
	var rets []float64
	j := 0
	for i := range bars {
		target := bars[i].Time + span
		for j < len(bars) && bars[j].Time < target {
			j++
		}
		if j >= len(bars) {
			break
		}
		if bars[i].Close > 0 {
			rets = append(rets, (bars[j].Close/bars[i].Close-1)*100)
		}
	}
	if len(rets) < minSamples {
		return HoldStats{}, false
	}
	sort.Float64s(rets)
	wins := 0
	for _, r := range rets {
		if r > 0 {
			wins++
		}
	}
	return HoldStats{
		Label: w.label, Days: w.days, Samples: len(rets),
		WinRate: round1(float64(wins) / float64(len(rets)) * 100),
		Median:  round1(percentile(rets, 0.5)),
		P10:     round1(percentile(rets, 0.1)),
		P90:     round1(percentile(rets, 0.9)),
	}, true
}

func percentile(sorted []float64, q float64) float64 {
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	return sorted[lo] + (sorted[hi]-sorted[lo])*(pos-float64(lo))
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
