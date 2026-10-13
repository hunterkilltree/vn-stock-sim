package luk

import (
	"fmt"
	"math"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// Thresholds, all in one place. Luk's own numbers are for US stocks; the
// VN adaptations (ADR, liquidity) are explained in phase-luk-review.md.
const (
	// minBars: EMA 50 needs some warm-up, and RS looks back rsBars.
	minBars = 70

	adrBars = 20
	// Luk prefers ADR above ~5% in the US. HOSE's ±7% band squeezes daily
	// ranges, so 4% is "fast" here and 2.5% the floor for a swing trade.
	adrPass = 4.0
	adrWarn = 2.5

	// rsBars ≈ 3 months of sessions; rsLeader is the outperformance vs
	// VN-Index that marks a leader rather than a follower.
	rsBars   = 63
	rsLeader = 10.0

	// Average traded value (VND) over 20 sessions: enough to get in and
	// out of a full position without moving the price.
	liquidityPass = 10e9
	liquidityWarn = 3e9

	// touchBars: the pullback must have reached the 9/21 EMA within this
	// many sessions; touchBandPct is how close counts as a touch.
	touchBars    = 3
	touchBandPct = 1.0
	// A close more than extendedADR × ADR above the 21 EMA is extended:
	// the stop would be far and the reward/risk poor.
	extendedADR = 2.0

	// Volume dry-up on the pullback: the last 3 sessions vs 20-day average.
	dryUpBars  = 3
	dryUpPass  = 0.8
	dryUpWarn  = 1.2
	avwapBars  = 63
	stopBuffer = 0.002 // stop just under the day's low

	// Luk: "Stop = 1.5-5%, portfolio risk per trade = 0.5%, max position
	// sizing = 35%", and keep the stop within about half the ADR.
	stopMinPct     = 1.5
	stopMaxPct     = 5.0
	riskPct        = 0.5
	maxPositionPct = 35.0
	// His winners pay ~5R; a low win rate only works with that payoff.
	rewardRisk = 5.0
)

// Input is everything Compute needs, so it's testable without providers.
// Bars and IndexBars are daily, oldest first. Tick rounds a price onto the
// exchange grid (nil leaves prices as is).
type Input struct {
	Symbol    string
	Bars      []market.Bar
	IndexBars []market.Bar
	Tick      func(float64) float64
}

// Compute runs every rule and rolls them up into a verdict and a plan.
func Compute(in Input) Result {
	res := Result{Symbol: in.Symbol, Source: "rule-based", Notes: vnNotes()}
	bars := in.Bars
	if len(bars) < minBars {
		res.Verdict = VerdictAvoid
		res.Checks = []Check{{
			Key: "history", Group: GroupSetup, Label: "Đủ lịch sử giá", Core: true, Status: StatusFail,
			Value:  fmt.Sprintf("%d phiên", len(bars)),
			Detail: fmt.Sprintf("Cần ít nhất %d phiên để tính EMA 50 và sức mạnh tương đối.", minBars),
		}}
		res.Total = 1
		return res
	}
	tick := in.Tick
	if tick == nil {
		tick = func(p float64) float64 { return p }
	}

	closes := closesOf(bars)
	ema9, ema21, ema50 := emaFull(closes, 9), emaFull(closes, 21), emaFull(closes, 50)
	last := len(bars) - 1
	close := closes[last]
	adr := adrPct(bars, adrBars)

	checks := []Check{
		trendCheck(close, ema9[last], ema21[last], ema50[last]),
		rsCheck(bars, in.IndexBars),
		adrCheck(adr),
		liquidityCheck(bars),
		pullbackCheck(bars, ema9, ema21, adr),
		confluenceCheck(bars, ema9[last], ema21[last], adr),
		dryUpCheck(bars),
	}

	// Under the day's low, but never closer than stopMinPct: a tighter stop
	// is inside normal noise and gets shaken out.
	stop := tick(math.Min(bars[last].Low*(1-stopBuffer), close*(1-stopMinPct/100)))
	stopPct := (close - stop) / close * 100
	checks = append(checks, stopCheck(stopPct, adr), marketCheck(in.IndexBars))

	res.Checks = checks
	res.Total = len(checks)
	res.Verdict = VerdictBuy
	for _, c := range checks {
		if c.Status == StatusPass {
			res.Passed++
		}
		switch {
		case c.Core && c.Status == StatusFail && c.Group != GroupEntry:
			res.Verdict = VerdictAvoid
		case (c.Core || c.Status == StatusFail) && c.Status != StatusPass && res.Verdict == VerdictBuy:
			res.Verdict = VerdictWatch
		}
	}

	if stopPct > 0 {
		pos := math.Min(maxPositionPct, riskPct/stopPct*100)
		res.Plan = &Plan{
			Entry:       close,
			Stop:        stop,
			StopPct:     round2(stopPct),
			RiskPct:     riskPct,
			PositionPct: round2(pos),
			Target:      tick(close + rewardRisk*(close-stop)),
			RewardRisk:  rewardRisk,
			TrailEMA9:   tick(ema9[last]),
			EMA21:       tick(ema21[last]),
		}
	}
	return res
}

// trendCheck: Luk only buys pullbacks inside an established uptrend --
// price above a rising EMA stack (9 > 21 > 50).
func trendCheck(close, e9, e21, e50 float64) Check {
	c := Check{
		Key: "trend", Group: GroupSetup, Label: "Xu hướng tăng (EMA 9 > 21 > 50)", Core: true,
		Value: fmt.Sprintf("Giá %s EMA21 · EMA21 %s EMA50", cmp(close, e21), cmp(e21, e50)),
	}
	switch {
	case close > e21 && e9 > e21 && e21 > e50:
		c.Status, c.Detail = StatusPass, "Các đường EMA xếp tăng dần, giá trên EMA 21: đúng loại cổ phiếu Luk tìm điểm mua pullback."
	case close > e50 && e21 > e50:
		c.Status, c.Detail = StatusWarn, "Xu hướng trung hạn còn tăng nhưng giá/EMA 9 đã xuống dưới EMA 21 -- chờ lấy lại EMA 21."
	default:
		c.Status, c.Detail = StatusFail, "Không có xu hướng tăng (EMA 21 dưới EMA 50 hoặc giá dưới EMA 50). Luk không bắt đáy."
	}
	return c
}

// rsCheck: he trades leaders -- the stocks up most over recent months --
// so the stock must beat VN-Index over rsBars.
func rsCheck(bars, index []market.Bar) Check {
	c := Check{Key: "rs", Group: GroupSetup, Label: "Sức mạnh tương đối vs VN-Index (3 tháng)", Core: true}
	stock := changePct(bars, rsBars)
	if len(index) <= rsBars {
		c.Status, c.Value = StatusWarn, signedPct(stock)
		c.Detail = "Thiếu dữ liệu VN-Index; chỉ có mức tăng của riêng cổ phiếu."
		return c
	}
	diff := stock - changePct(index, rsBars)
	c.Value = fmt.Sprintf("%s (%s so với chỉ số)", signedPct(stock), signedPct(diff))
	switch {
	case diff >= rsLeader:
		c.Status, c.Detail = StatusPass, "Vượt trội VN-Index rõ rệt: cổ phiếu dẫn dắt."
	case diff > 0:
		c.Status, c.Detail = StatusWarn, "Mạnh hơn thị trường nhưng chưa phải nhóm dẫn đầu."
	default:
		c.Status, c.Detail = StatusFail, "Yếu hơn VN-Index. Luk chỉ giao dịch cổ phiếu mạnh nhất."
	}
	return c
}

// adrCheck: he wants names that move -- a slow stock can't pay 5R before
// the trailing EMA stops it out.
func adrCheck(adr float64) Check {
	c := Check{Key: "adr", Group: GroupSetup, Label: "Biên độ trung bình ngày (ADR 20)", Value: fmt.Sprintf("%.1f%%", adr)}
	switch {
	case adr >= adrPass:
		c.Status, c.Detail = StatusPass, "Đủ biến động để một lệnh thắng đạt nhiều R."
	case adr >= adrWarn:
		c.Status, c.Detail = StatusWarn, "Biến động vừa phải; lợi nhuận kỳ vọng mỗi lệnh thấp hơn."
	default:
		c.Status, c.Detail = StatusFail, "Cổ phiếu đi quá chậm cho phong cách swing của Luk."
	}
	return c
}

func liquidityCheck(bars []market.Bar) Check {
	n := min(20, len(bars))
	var sum float64
	for _, b := range bars[len(bars)-n:] {
		sum += b.Close * float64(b.Volume)
	}
	avg := sum / float64(n)
	c := Check{Key: "liquidity", Group: GroupSetup, Label: "Thanh khoản (GTGD TB 20 phiên)", Core: true, Value: fmt.Sprintf("%.1f tỷ", avg/1e9)}
	switch {
	case avg >= liquidityPass:
		c.Status, c.Detail = StatusPass, "Đủ thanh khoản để vào/ra cả vị thế mà không trượt giá."
	case avg >= liquidityWarn:
		c.Status, c.Detail = StatusWarn, "Thanh khoản mỏng: giảm quy mô vị thế (Luk giảm size với cổ phiếu nhỏ)."
	default:
		c.Status, c.Detail = StatusFail, "Quá kém thanh khoản; điểm dừng lỗ có thể không khớp được."
	}
	return c
}

// pullbackCheck: the entry is weakness into the 9/21 EMA, not a chase.
func pullbackCheck(bars []market.Bar, ema9, ema21 []float64, adr float64) Check {
	last := len(bars) - 1
	close := bars[last].Close
	ext := (close/ema21[last] - 1) * 100
	c := Check{Key: "pullback", Group: GroupEntry, Label: "Điểm vào: pullback về EMA 9/21", Core: true,
		Value: fmt.Sprintf("%s so với EMA21", signedPct(ext))}

	touched := false
	for i := last; i > last-touchBars && i >= 0; i-- {
		support := math.Max(ema9[i], ema21[i])
		if bars[i].Low <= support*(1+touchBandPct/100) {
			touched = true
			break
		}
	}
	switch {
	case close < ema21[last]*(1-touchBandPct/100):
		c.Status, c.Detail = StatusFail, "Giá đóng cửa đã thủng EMA 21: pullback biến thành gãy hỗ trợ."
	case touched && ext <= 1.5*adr:
		c.Status, c.Detail = StatusPass, "Giá vừa chạm EMA 9/21 và giữ được: vùng mua rủi ro thấp."
	case ext > extendedADR*adr:
		c.Status, c.Detail = StatusWarn, "Giá đang kéo giãn xa EMA 21 -- đừng đuổi, chờ nhịp điều chỉnh."
	default:
		c.Status, c.Detail = StatusWarn, "Chưa có nhịp lùi về EMA 9/21 trong 3 phiên gần nhất."
	}
	return c
}

// confluenceCheck: his best entries line up several supports -- the
// 9/21 EMA and an anchored VWAP (here anchored at the 3-month swing low).
func confluenceCheck(bars []market.Bar, e9, e21, adr float64) Check {
	last := len(bars) - 1
	close := bars[last].Close
	vw := avwapFromSwingLow(bars, avwapBars)
	c := Check{Key: "confluence", Group: GroupEntry, Label: "Hội tụ hỗ trợ (EMA + AVWAP đáy 3 tháng)",
		Value: fmt.Sprintf("AVWAP %.2f", vw/1000)}
	if vw == 0 {
		c.Status, c.Detail = StatusWarn, "Không tính được AVWAP (thiếu khối lượng)."
		return c
	}
	hi := math.Max(e9, math.Max(e21, vw))
	lo := math.Min(e9, math.Min(e21, vw))
	spread := (hi - lo) / close * 100
	switch {
	case close < vw:
		c.Status, c.Detail = StatusFail, "Giá dưới AVWAP từ đáy gần nhất: người mua từ đáy đang lỗ."
	case spread <= adr:
		c.Status, c.Detail = StatusPass, fmt.Sprintf("EMA 9, EMA 21 và AVWAP nằm trong %.1f%% -- vùng hỗ trợ dày.", spread)
	default:
		c.Status, c.Detail = StatusWarn, "Giá trên AVWAP nhưng các mức hỗ trợ cách xa nhau."
	}
	return c
}

// dryUpCheck: a healthy pullback comes on lighter volume.
func dryUpCheck(bars []market.Bar) Check {
	n := len(bars)
	var recent, base float64
	for _, b := range bars[n-dryUpBars:] {
		recent += float64(b.Volume)
	}
	recent /= dryUpBars
	m := min(20, n)
	for _, b := range bars[n-m:] {
		base += float64(b.Volume)
	}
	base /= float64(m)
	c := Check{Key: "dryup", Group: GroupEntry, Label: "Khối lượng cạn khi điều chỉnh"}
	if base == 0 {
		c.Status, c.Value, c.Detail = StatusWarn, "—", "Không có dữ liệu khối lượng."
		return c
	}
	r := recent / base
	c.Value = fmt.Sprintf("%.2fx TB 20", r)
	switch {
	case r <= dryUpPass:
		c.Status, c.Detail = StatusPass, "Khối lượng 3 phiên gần nhất thấp: lực bán yếu."
	default:
		c.Status, c.Detail = StatusWarn, "Khối lượng chưa cạn; nếu là phiên giảm thì đó là phân phối."
		if r <= dryUpWarn {
			c.Detail = "Khối lượng quanh mức trung bình."
		}
	}
	return c
}

// stopCheck: a stop under today's low must be tight -- 1.5-5% and within
// about half the ADR -- or the 0.5% risk buys too small a position.
func stopCheck(stopPct, adr float64) Check {
	limit := math.Min(stopMaxPct, math.Max(stopMinPct, adr/2))
	c := Check{Key: "stop", Group: GroupRisk, Label: "Dừng lỗ dưới đáy phiên (1,5–5%, ≤ ½ ADR)", Core: true,
		Value: fmt.Sprintf("%.1f%%", stopPct)}
	switch {
	case stopPct <= limit:
		c.Status, c.Detail = StatusPass, fmt.Sprintf("Dừng lỗ chặt (≤ %.1f%%): rủi ro nhỏ, vị thế đủ lớn.", limit)
	case stopPct <= stopMaxPct:
		c.Status, c.Detail = StatusWarn, "Dừng lỗ hơi rộng so với ADR: giảm quy mô hoặc chờ phiên co hẹp."
	default:
		c.Status, c.Detail = StatusFail, "Dừng lỗ > 5%: phiên hôm nay quá rộng, rủi ro/lợi nhuận không đạt."
	}
	return c
}

// marketCheck: trade with the market -- VN-Index above its 21 EMA.
func marketCheck(index []market.Bar) Check {
	c := Check{Key: "market", Group: GroupRisk, Label: "Thị trường chung (VN-Index trên EMA 21)", Core: true}
	if len(index) < minBars {
		c.Status, c.Value, c.Detail = StatusWarn, "—", "Thiếu dữ liệu VN-Index."
		return c
	}
	closes := closesOf(index)
	e21, e50 := emaFull(closes, 21), emaFull(closes, 50)
	last := len(closes) - 1
	cl := closes[last]
	c.Value = fmt.Sprintf("%s so với EMA21", signedPct((cl/e21[last]-1)*100))
	switch {
	case cl > e21[last] && e21[last] > e50[last]:
		c.Status, c.Detail = StatusPass, "Thị trường đang uptrend: mở vị thế mới được."
	case cl > e50[last]:
		c.Status, c.Detail = StatusWarn, "Thị trường điều chỉnh ngắn hạn: giảm rủi ro mỗi lệnh."
	default:
		c.Status, c.Detail = StatusFail, "VN-Index dưới EMA 50: đứng ngoài, bảo toàn vốn."
	}
	return c
}

func vnNotes() []string {
	return []string{
		"T+2,5: cổ phiếu mua hôm nay chỉ bán được từ chiều phiên T+2 -- điểm dừng lỗ chưa thể thực hiện ngay như ở Mỹ; giảm quy mô cho phù hợp.",
		"Biên độ ±7% (HOSE), ±10% (HNX), ±15% (UPCOM) và phiên trắng bên mua có thể làm giá vượt qua điểm dừng lỗ.",
		"Luk thắng khoảng 1/4 số lệnh; lợi nhuận đến từ vài lệnh thắng lớn (~5R) và cắt lỗ nhỏ, đều đặn. Thoát dần ~10% khi tăng mạnh, phần còn lại giữ theo EMA 9.",
		"Không phải khuyến nghị đầu tư -- công cụ luyện tập theo quy tắc công khai của Martin Luk.",
	}
}

// adrPct is the mean of (high/low − 1) over the last n bars, in percent.
func adrPct(bars []market.Bar, n int) float64 {
	n = min(n, len(bars))
	var sum float64
	cnt := 0
	for _, b := range bars[len(bars)-n:] {
		if b.Low > 0 {
			sum += (b.High/b.Low - 1) * 100
			cnt++
		}
	}
	if cnt == 0 {
		return 0
	}
	return sum / float64(cnt)
}

// avwapFromSwingLow anchors a VWAP at the lowest low of the last n bars.
func avwapFromSwingLow(bars []market.Bar, n int) float64 {
	start := max(0, len(bars)-n)
	anchor := start
	for i := start; i < len(bars); i++ {
		if bars[i].Low < bars[anchor].Low {
			anchor = i
		}
	}
	var pv, v float64
	for _, b := range bars[anchor:] {
		typical := (b.High + b.Low + b.Close) / 3
		pv += typical * float64(b.Volume)
		v += float64(b.Volume)
	}
	if v == 0 {
		return 0
	}
	return pv / v
}

func changePct(bars []market.Bar, n int) float64 {
	if len(bars) <= n || bars[len(bars)-1-n].Close == 0 {
		return 0
	}
	return (bars[len(bars)-1].Close/bars[len(bars)-1-n].Close - 1) * 100
}

func closesOf(bars []market.Bar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.Close
	}
	return out
}

func emaFull(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	k := 2.0 / float64(period+1)
	for i, v := range values {
		if i == 0 {
			out[i] = v
			continue
		}
		out[i] = v*k + out[i-1]*(1-k)
	}
	return out
}

func cmp(a, b float64) string {
	if a > b {
		return ">"
	}
	return "<"
}

func signedPct(v float64) string { return fmt.Sprintf("%+.1f%%", v) }

func round2(v float64) float64 { return math.Round(v*100) / 100 }
