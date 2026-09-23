package market

// emaFull is a full-length EMA (every bar gets a value, seeded from the
// first close, no leading gap) -- distinct from service.go's ema(),
// which is windowed (skips the first period-1 bars). MACD needs the
// full-length variant: design/screens/Detail.dc.html's own ema() helper
// seeds prev with the first value and runs immediately, so the chart's
// MACD/signal lines span the whole visible range with no gap. Ported
// directly, per design/README.md's note that this math can be lifted as-is.
func emaFull(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) == 0 {
		return out
	}
	k := 2.0 / float64(period+1)
	prev := closes[0]
	out[0] = prev
	for i := 1; i < len(closes); i++ {
		prev = closes[i]*k + prev*(1-k)
		out[i] = prev
	}
	return out
}

// GetMACD computes MACD(fastPeriod, slowPeriod, signalPeriod) over the
// symbol's bars -- three series per point (macd, signal, histogram),
// which doesn't fit the single-value IndicatorPoint shape GetIndicator
// uses for sma/ema/rsi, hence the dedicated IndicatorMultiPoint return
// (already defined in types.go, unused until now -- see phase-d.md
// decision 2).
func (s *Service) GetMACD(sym, resolution string, fastPeriod, slowPeriod, signalPeriod int, from, to int64) []IndicatorMultiPoint {
	bars := s.data.GetBars(sym, resolution, from, to)
	if len(bars) == 0 {
		return []IndicatorMultiPoint{}
	}
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}

	fast := emaFull(closes, fastPeriod)
	slow := emaFull(closes, slowPeriod)
	macdLine := make([]float64, len(closes))
	for i := range closes {
		macdLine[i] = fast[i] - slow[i]
	}
	signalLine := emaFull(macdLine, signalPeriod)

	out := make([]IndicatorMultiPoint, len(bars))
	for i, b := range bars {
		out[i] = IndicatorMultiPoint{
			Time: b.Time,
			Values: map[string]float64{
				"macd":      round2(macdLine[i]),
				"signal":    round2(signalLine[i]),
				"histogram": round2(macdLine[i] - signalLine[i]),
			},
		}
	}
	return out
}
