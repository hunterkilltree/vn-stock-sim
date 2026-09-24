package market

// rsi computes Wilder-smoothed RSI over period, aligned index-for-index
// with bars (entries before enough data exists are omitted, same
// convention as sma/ema in service.go). Ported from
// design/screens/Detail.dc.html's rsiCalc -- design/README.md notes this
// math "is real and can be lifted directly" (Wilder smoothing: the
// average gain/loss is itself an exponential average seeded by a simple
// average of the first `period` changes, not a plain moving average).
func rsi(bars []Bar, period int) []IndicatorPoint {
	if period <= 0 || len(bars) < period+1 {
		return []IndicatorPoint{}
	}
	out := make([]IndicatorPoint, 0, len(bars)-period)
	var avgGain, avgLoss float64
	for i := 1; i < len(bars); i++ {
		delta := bars[i].Close - bars[i-1].Close
		gain := max2(delta, 0)
		loss := max2(-delta, 0)

		switch {
		case i < period:
			avgGain += gain
			avgLoss += loss
		case i == period:
			avgGain = (avgGain + gain) / float64(period)
			avgLoss = (avgLoss + loss) / float64(period)
			out = append(out, IndicatorPoint{Time: bars[i].Time, Value: round2(rsiFromAvg(avgGain, avgLoss))})
		default:
			avgGain = (avgGain*float64(period-1) + gain) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
			out = append(out, IndicatorPoint{Time: bars[i].Time, Value: round2(rsiFromAvg(avgGain, avgLoss))})
		}
	}
	return out
}

func rsiFromAvg(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50
		}
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}

// RSI and SMA are exported for packages that already hold bars (backtest,
// quant) so they don't re-implement the same indicator math.
func RSI(bars []Bar, period int) []IndicatorPoint { return rsi(bars, period) }

func SMA(bars []Bar, period int) []IndicatorPoint { return sma(bars, period) }
