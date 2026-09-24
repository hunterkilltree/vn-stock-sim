package replay

import (
	"math"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// computeSkillScore is the heuristic from phase-f.md section 7 -- never
// presented as a validated skill assessment, always returned with
// Source: "heuristic" and Final reflecting whether the session has
// actually ended.
//
// Entry/exit quality look at a window of nearby bars to judge how close
// a fill was to the best price achievable around it. Even though
// view() now calls this on every response (not just after End, to match
// the design's "provisional score" panel), the window is clamped to
// bars actually revealed at the time -- sess.Bars[:sess.CurrentBar] --
// unless the session is completed, in which case the full known series
// is fair game. Without this clamp, a mid-session score would leak
// future prices into a number the client is allowed to see before the
// session ends, defeating the entire point of Advance's reveal gating.
func computeSkillScore(sess *Session) SkillScore {
	final := sess.Status == "completed"
	windowBars := sess.Bars[:sess.CurrentBar]
	if final {
		windowBars = sess.Bars
	}

	var buys, sells []Fill
	for _, f := range sess.Fills {
		if f.Side == "buy" {
			buys = append(buys, f)
		} else {
			sells = append(sells, f)
		}
	}

	return SkillScore{
		Source:         "heuristic",
		Final:          final,
		EntryQuality:   entryExitQuality(buys, windowBars, true),
		ExitQuality:    entryExitQuality(sells, windowBars, false),
		StopDiscipline: stopDiscipline(sess.Fills),
		PositionSizing: positionSizing(sess.Fills),
		Overall: average(
			entryExitQuality(buys, windowBars, true),
			entryExitQuality(sells, windowBars, false),
			stopDiscipline(sess.Fills),
			positionSizing(sess.Fills),
		),
	}
}

// replayAccounting computes NAV/PnlPercent/MaxDrawdownPercent purely
// from this session's own fills and its own revealed bar closes --
// never a live market quote. See view()'s comment on why: this is the
// same accounting style backtest/rule.go already uses for a simulated
// run over historical data (cash +/- fill notional, mark the remaining
// position to the *replay's own* current close), applied bar-by-bar so
// max drawdown can be computed too.
func replayAccounting(sess *Session) (nav, pnlPercent, maxDrawdownPercent float64) {
	fillsByBar := make(map[int][]Fill)
	for _, f := range sess.Fills {
		fillsByBar[f.BarIndex] = append(fillsByBar[f.BarIndex], f)
	}

	cash := sess.Capital
	var qty float64
	peak := sess.Capital
	var maxDD float64

	for i := 0; i < sess.CurrentBar; i++ {
		for _, f := range fillsByBar[i] {
			notional := f.Price * float64(f.Quantity)
			if f.Side == "buy" {
				cash -= notional
				qty += f.Quantity
			} else {
				cash += notional
				qty -= f.Quantity
			}
		}
		equity := cash + float64(qty)*sess.Bars[i].Close
		if equity > peak {
			peak = equity
		}
		if peak > 0 {
			if dd := (peak - equity) / peak * 100; dd > maxDD {
				maxDD = dd
			}
		}
	}

	finalEquity := cash + float64(qty)*sess.Bars[sess.CurrentBar-1].Close
	return round2(finalEquity), round2((finalEquity - sess.Capital) / sess.Capital * 100), round2(maxDD)
}

const scoreWindow = 5

// entryExitQuality scores each fill by how close its price was to the
// best price achievable in a +/-scoreWindow-bar window around it --
// lowest close for a buy (best entry), highest close for a sell (best
// exit). No fills of that side -> 100 (nothing to penalize).
func entryExitQuality(fills []Fill, bars []market.Bar, wantLowest bool) float64 {
	if len(fills) == 0 {
		return 100
	}
	var sum float64
	for _, f := range fills {
		lo := f.BarIndex - scoreWindow
		if lo < 0 {
			lo = 0
		}
		hi := f.BarIndex + scoreWindow
		if hi >= len(bars) {
			hi = len(bars) - 1
		}
		best := bars[lo].Close
		for i := lo; i <= hi; i++ {
			if wantLowest && bars[i].Close < best {
				best = bars[i].Close
			}
			if !wantLowest && bars[i].Close > best {
				best = bars[i].Close
			}
		}
		if best <= 0 {
			continue
		}
		diff := math.Abs(f.Price-best) / best * 100
		sum += clamp(100-diff, 0, 100)
	}
	return round2(sum / float64(len(fills)))
}

// stopDiscipline rewards setting a stop-loss on a buy (Fill.StopSet,
// recorded at fill time -- not inferred after the fact) and separately
// rewards stops that actually triggered (a later sell with the auto-
// exit note), per phase-f.md section 7. No buys at all -> 100 (nothing
// to judge); buys exist but no stop was ever set -> 0 (there is
// nothing to credit).
func stopDiscipline(fills []Fill) float64 {
	var buys, setCount, triggeredCount int
	for _, f := range fills {
		switch {
		case f.Side == "buy":
			buys++
			if f.StopSet > 0 {
				setCount++
			}
		case f.Side == "sell" && f.Note != "":
			triggeredCount++
		}
	}
	if buys == 0 {
		return 100
	}
	setRatio := float64(setCount) / float64(buys)
	var honoredRatio float64
	if setCount > 0 {
		honoredRatio = float64(triggeredCount) / float64(setCount)
	}
	return round2(clamp(setRatio*70+honoredRatio*30, 0, 100))
}

// positionSizing rewards consistent bet sizing (low coefficient of
// variation in each fill's notional value) over wildly varying sizes.
// Fewer than 2 fills -> 100 (nothing to compare).
func positionSizing(fills []Fill) float64 {
	if len(fills) < 2 {
		return 100
	}
	notionals := make([]float64, len(fills))
	var sum float64
	for i, f := range fills {
		notionals[i] = f.Price * float64(f.Quantity)
		sum += notionals[i]
	}
	mean := sum / float64(len(notionals))
	if mean <= 0 {
		return 100
	}
	var variance float64
	for _, n := range notionals {
		variance += (n - mean) * (n - mean)
	}
	variance /= float64(len(notionals))
	stddev := math.Sqrt(variance)
	cv := stddev / mean * 100
	return round2(clamp(100-cv, 0, 100))
}

func average(vals ...float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return round2(sum / float64(len(vals)))
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
