package market

// Service holds market-data rules (V1: none beyond delegation — caching,
// gap-filling, and session-hours clipping are noted as future work in
// RESUME.md, matching charting-library-integration.md section 7).
type Service struct {
	data MarketDataProvider
}

func NewService(data MarketDataProvider) *Service {
	return &Service{data: data}
}

func (s *Service) GetBars(sym, resolution string, from, to int64) []Bar {
	return s.data.GetBars(sym, resolution, from, to)
}

// GetIndicator computes a basic SMA/EMA/RSI over the underlying bars. Only
// "sma" and "ema" are implemented for V1; others return an empty slice
// until the full indicator library lands (see RESUME.md).
func (s *Service) GetIndicator(sym, resolution, indicator string, period int, from, to int64) []IndicatorPoint {
	bars := s.data.GetBars(sym, resolution, from, to)
	switch indicator {
	case "sma":
		return sma(bars, period)
	case "ema":
		return ema(bars, period)
	default:
		return []IndicatorPoint{}
	}
}

func sma(bars []Bar, period int) []IndicatorPoint {
	if period <= 0 || len(bars) < period {
		return []IndicatorPoint{}
	}
	out := make([]IndicatorPoint, 0, len(bars)-period+1)
	var sum float64
	for i, b := range bars {
		sum += b.Close
		if i >= period {
			sum -= bars[i-period].Close
		}
		if i >= period-1 {
			out = append(out, IndicatorPoint{Time: b.Time, Value: round2(sum / float64(period))})
		}
	}
	return out
}

func ema(bars []Bar, period int) []IndicatorPoint {
	if period <= 0 || len(bars) < period {
		return []IndicatorPoint{}
	}
	k := 2.0 / float64(period+1)
	out := make([]IndicatorPoint, 0, len(bars)-period+1)
	var prev float64
	var sum float64
	for i, b := range bars {
		if i < period {
			sum += b.Close
			if i == period-1 {
				prev = sum / float64(period)
				out = append(out, IndicatorPoint{Time: b.Time, Value: round2(prev)})
			}
			continue
		}
		prev = b.Close*k + prev*(1-k)
		out = append(out, IndicatorPoint{Time: b.Time, Value: round2(prev)})
	}
	return out
}
