package backtest

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/market"

type runResult struct {
	finalCapital  float64
	returnPercent float64
	totalTrades   int
	winRate       float64
}

// runEMACrossover is a minimal long-only simulation: buy all-in when the
// fast EMA crosses above the slow EMA, sell all when it crosses below.
// Deliberately simple for V1 — see vn-stock-sim-version-highlights.md's
// "Run a basic backtest ... EMA crossover" scope, and RESUME.md for the
// richer Strategy Builder rules planned for V2.
func runEMACrossover(bars []market.Bar, fastPeriod, slowPeriod int, startingCapital float64) runResult {
	if len(bars) < slowPeriod+1 {
		return runResult{finalCapital: startingCapital, returnPercent: 0}
	}

	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	fastEMA := emaSeries(closes, fastPeriod)
	slowEMA := emaSeries(closes, slowPeriod)

	cash := startingCapital
	var shares float64
	inPosition := false
	var entryPrice float64
	trades := 0
	wins := 0

	for i := 1; i < len(bars); i++ {
		if fastEMA[i] == 0 || slowEMA[i] == 0 || fastEMA[i-1] == 0 || slowEMA[i-1] == 0 {
			continue
		}
		crossedUp := fastEMA[i-1] <= slowEMA[i-1] && fastEMA[i] > slowEMA[i]
		crossedDown := fastEMA[i-1] >= slowEMA[i-1] && fastEMA[i] < slowEMA[i]

		if !inPosition && crossedUp {
			shares = cash / bars[i].Close
			cash = 0
			entryPrice = bars[i].Close
			inPosition = true
		} else if inPosition && crossedDown {
			cash = shares * bars[i].Close
			trades++
			if bars[i].Close > entryPrice {
				wins++
			}
			shares = 0
			inPosition = false
		}
	}

	finalCapital := cash + shares*bars[len(bars)-1].Close
	winRate := 0.0
	if trades > 0 {
		winRate = float64(wins) / float64(trades) * 100
	}
	return runResult{
		finalCapital:  finalCapital,
		returnPercent: (finalCapital - startingCapital) / startingCapital * 100,
		totalTrades:   trades,
		winRate:       winRate,
	}
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
