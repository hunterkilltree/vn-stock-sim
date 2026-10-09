package rating

import (
	"fmt"
	"math"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// Thresholds, all in one place so the rules stay inspectable. Each is a
// common rule of thumb, not a tuned parameter -- see phase-rating.md for
// the reasoning behind every number.
const (
	rsiOversold   = 30.0
	rsiOverbought = 70.0
	// trendBandPct: price within ±1% of an average counts as "on" it, so
	// noise around the line doesn't flip the vote every session.
	trendBandPct = 1.0
	// momentumBars ≈ 3 months of HOSE sessions; momentumPct is the move
	// that counts as a trend rather than drift.
	momentumBars = 63
	momentumPct  = 5.0
	// volumeSpike: a session at 1.5x its 20-day average volume confirms
	// that day's direction.
	volumeSpike = 1.5

	peCheap     = 10.0
	peExpensive = 25.0
	pbCheap     = 1.0
	pbExpensive = 3.0
	roeStrong   = 15.0
	roeWeak     = 8.0
	// dividendHigh: roughly a Vietnamese 12-month bank deposit rate.
	dividendHigh = 4.0

	// Overall = technicalWeight*technical + (1-technicalWeight)*fundamental.
	technicalWeight = 0.6

	strongCutoff = 0.5
	weakCutoff   = 0.15
)

// Compute is the whole rating: pure over a symbol's detail and its daily
// bars (oldest first), so it's unit-testable without a market provider.
func Compute(detail symbol.Detail, bars []market.Bar) Result {
	tech := technicalSignals(bars)
	fund := fundamentalSignals(detail)

	signals := append(append([]Signal{}, tech...), fund...)
	res := Result{
		Symbol:    detail.Symbol.Symbol,
		Technical: summarize(tech),
		Signals:   signals,
		Source:    "rule-based",
	}
	res.Overall = res.Technical
	if len(fund) > 0 {
		f := summarize(fund)
		res.Fundamental = &f
		score := technicalWeight*res.Technical.Score + (1-technicalWeight)*f.Score
		if len(tech) == 0 {
			score = f.Score
		}
		score = round2(score)
		res.Overall = Summary{
			Rating:  ratingFor(score),
			Score:   score,
			Buy:     res.Technical.Buy + f.Buy,
			Neutral: res.Technical.Neutral + f.Neutral,
			Sell:    res.Technical.Sell + f.Sell,
		}
	}
	return res
}

func technicalSignals(bars []market.Bar) []Signal {
	if len(bars) < 2 {
		return nil
	}
	last := bars[len(bars)-1]
	price := last.Close
	var out []Signal

	// Price vs. moving averages: short, medium and long-term trend.
	for _, ma := range []struct {
		period int
		label  string
	}{{20, "Giá so với SMA 20"}, {50, "Giá so với SMA 50"}, {200, "Giá so với SMA 200"}} {
		avg, ok := lastSMA(bars, ma.period)
		if !ok {
			continue
		}
		diff := (price - avg) / avg * 100
		v := VerdictNeutral
		detail := fmt.Sprintf("Giá đi ngang quanh đường trung bình %d phiên", ma.period)
		if diff > trendBandPct {
			v, detail = VerdictBuy, fmt.Sprintf("Giá nằm trên đường trung bình %d phiên", ma.period)
		} else if diff < -trendBandPct {
			v, detail = VerdictSell, fmt.Sprintf("Giá nằm dưới đường trung bình %d phiên", ma.period)
		}
		out = append(out, Signal{GroupTechnical, ma.label, signedPct(diff), v, detail})
	}

	// Golden cross / death cross: SMA 50 vs SMA 200.
	if s50, ok := lastSMA(bars, 50); ok {
		if s200, ok := lastSMA(bars, 200); ok {
			sig := Signal{Group: GroupTechnical, Label: "SMA 50 so với SMA 200", Value: signedPct((s50 - s200) / s200 * 100)}
			switch {
			case s50 > s200:
				sig.Verdict, sig.Detail = VerdictBuy, "Golden cross: xu hướng dài hạn tăng"
			case s50 < s200:
				sig.Verdict, sig.Detail = VerdictSell, "Death cross: xu hướng dài hạn giảm"
			default:
				sig.Verdict, sig.Detail = VerdictNeutral, "SMA 50 trùng SMA 200"
			}
			out = append(out, sig)
		}
	}

	// RSI 14: contrarian at the extremes, neutral in between.
	if pts := market.RSI(bars, 14); len(pts) > 0 {
		r := pts[len(pts)-1].Value
		sig := Signal{GroupTechnical, "RSI 14", fmt.Sprintf("%.1f", r), VerdictNeutral, "RSI trong vùng trung tính (30–70)"}
		if r < rsiOversold {
			sig.Verdict, sig.Detail = VerdictBuy, "Quá bán (RSI < 30), khả năng hồi phục"
		} else if r > rsiOverbought {
			sig.Verdict, sig.Detail = VerdictSell, "Quá mua (RSI > 70), rủi ro điều chỉnh"
		}
		out = append(out, sig)
	}

	// MACD(12,26,9): MACD line vs. its signal line.
	if len(bars) >= 35 {
		m, s := macd(bars)
		// Value as % of price: raw MACD is in price units, which differ
		// between stocks (VND) and crypto (USDT).
		sig := Signal{Group: GroupTechnical, Label: "MACD (12, 26, 9)", Value: fmt.Sprintf("%+.2f%% giá", (m-s)/price*100)}
		// Tolerance: a float residue on a flat series isn't a crossover.
		switch d := m - s; {
		case d > 1e-9:
			sig.Verdict, sig.Detail = VerdictBuy, "MACD nằm trên đường tín hiệu"
		case d < -1e-9:
			sig.Verdict, sig.Detail = VerdictSell, "MACD nằm dưới đường tín hiệu"
		default:
			sig.Verdict, sig.Detail = VerdictNeutral, "MACD trùng đường tín hiệu"
		}
		out = append(out, sig)
	}

	// 3-month momentum (rate of change).
	if len(bars) > momentumBars {
		base := bars[len(bars)-1-momentumBars].Close
		roc := (price - base) / base * 100
		sig := Signal{GroupTechnical, "Động lượng 3 tháng", signedPct(roc), VerdictNeutral, "Biến động 3 tháng không đáng kể"}
		if roc > momentumPct {
			sig.Verdict, sig.Detail = VerdictBuy, "Tăng mạnh trong 3 tháng"
		} else if roc < -momentumPct {
			sig.Verdict, sig.Detail = VerdictSell, "Giảm mạnh trong 3 tháng"
		}
		out = append(out, sig)
	}

	// Volume confirmation: a volume spike confirms the session's direction.
	if len(bars) >= 21 {
		var sum int64
		for _, b := range bars[len(bars)-21 : len(bars)-1] {
			sum += b.Volume
		}
		avg := float64(sum) / 20
		if avg > 0 {
			ratio := float64(last.Volume) / avg
			prev := bars[len(bars)-2].Close
			sig := Signal{GroupTechnical, "Khối lượng", fmt.Sprintf("%.1fx TB20", ratio), VerdictNeutral, "Khối lượng không xác nhận xu hướng"}
			if ratio >= volumeSpike && price > prev {
				sig.Verdict, sig.Detail = VerdictBuy, "Tăng giá kèm khối lượng đột biến"
			} else if ratio >= volumeSpike && price < prev {
				sig.Verdict, sig.Detail = VerdictSell, "Giảm giá kèm khối lượng đột biến"
			}
			out = append(out, sig)
		}
	}
	return out
}

// fundamentalSignals skips any field that's zero: the symbol package
// leaves fundamentals at zero when it has none (crypto, unseeded tickers),
// and a missing number shouldn't vote.
func fundamentalSignals(d symbol.Detail) []Signal {
	var out []Signal

	switch {
	case d.EPS < 0:
		out = append(out, Signal{GroupFundamental, "P/E", "Âm", VerdictSell, "Doanh nghiệp đang lỗ (EPS âm)"})
	case d.PERatio > 0:
		sig := Signal{GroupFundamental, "P/E", fmt.Sprintf("%.1f", d.PERatio), VerdictNeutral, "Định giá P/E hợp lý (10–25)"}
		if d.PERatio < peCheap {
			sig.Verdict, sig.Detail = VerdictBuy, "P/E thấp (< 10), định giá rẻ"
		} else if d.PERatio > peExpensive {
			sig.Verdict, sig.Detail = VerdictSell, "P/E cao (> 25), định giá đắt"
		}
		out = append(out, sig)
	}

	if d.PBRatio > 0 {
		sig := Signal{GroupFundamental, "P/B", fmt.Sprintf("%.1f", d.PBRatio), VerdictNeutral, "P/B trong vùng hợp lý (1–3)"}
		if d.PBRatio < pbCheap {
			sig.Verdict, sig.Detail = VerdictBuy, "Giá dưới giá trị sổ sách (P/B < 1)"
		} else if d.PBRatio > pbExpensive {
			sig.Verdict, sig.Detail = VerdictSell, "P/B cao (> 3)"
		}
		out = append(out, sig)
	}

	if d.ROE != 0 {
		sig := Signal{GroupFundamental, "ROE", fmt.Sprintf("%.1f%%", d.ROE), VerdictNeutral, "Hiệu quả sử dụng vốn trung bình (8–15%)"}
		if d.ROE >= roeStrong {
			sig.Verdict, sig.Detail = VerdictBuy, "Hiệu quả sử dụng vốn cao (ROE ≥ 15%)"
		} else if d.ROE < roeWeak {
			sig.Verdict, sig.Detail = VerdictSell, "Hiệu quả sử dụng vốn thấp (ROE < 8%)"
		}
		out = append(out, sig)
	}

	if d.DividendYield > 0 {
		sig := Signal{GroupFundamental, "Tỷ suất cổ tức", fmt.Sprintf("%.1f%%", d.DividendYield), VerdictNeutral, "Cổ tức thấp hơn lãi tiết kiệm"}
		if d.DividendYield >= dividendHigh {
			sig.Verdict, sig.Detail = VerdictBuy, "Cổ tức cao (≥ 4%), ngang lãi tiết kiệm"
		}
		out = append(out, sig)
	}
	return out
}

func summarize(signals []Signal) Summary {
	var s Summary
	for _, sig := range signals {
		switch sig.Verdict {
		case VerdictBuy:
			s.Buy++
		case VerdictSell:
			s.Sell++
		default:
			s.Neutral++
		}
	}
	if n := len(signals); n > 0 {
		s.Score = round2(float64(s.Buy-s.Sell) / float64(n))
	}
	s.Rating = ratingFor(s.Score)
	return s
}

func ratingFor(score float64) string {
	switch {
	case score >= strongCutoff:
		return RatingStrongBuy
	case score >= weakCutoff:
		return RatingBuy
	case score > -weakCutoff:
		return RatingNeutral
	case score > -strongCutoff:
		return RatingSell
	default:
		return RatingStrongSell
	}
}

func lastSMA(bars []market.Bar, period int) (float64, bool) {
	pts := market.SMA(bars, period)
	if len(pts) == 0 || pts[len(pts)-1].Value == 0 {
		return 0, false
	}
	return pts[len(pts)-1].Value, true
}

// macd returns the last MACD(12,26,9) and signal values, with the same
// full-length EMA seeding as market's GetMACD so the vote matches the
// chart's MACD pane.
func macd(bars []market.Bar) (macdLine, signal float64) {
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	fast, slow := emaFull(closes, 12), emaFull(closes, 26)
	line := make([]float64, len(closes))
	for i := range closes {
		line[i] = fast[i] - slow[i]
	}
	sig := emaFull(line, 9)
	return line[len(line)-1], sig[len(sig)-1]
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

func signedPct(v float64) string {
	return fmt.Sprintf("%+.1f%%", v)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
