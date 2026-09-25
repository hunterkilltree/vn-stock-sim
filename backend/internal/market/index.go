package market

import (
	"math"
	"time"
)

// IndexSnapshot is a market index's current value + day change, plus a
// short sparkline for the Main screen's index cards (Main.dc.html).
type IndexSnapshot struct {
	Name          string    `json:"name"`
	Value         float64   `json:"value"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"changePercent"`
	Sparkline     []float64 `json:"sparkline"`
}

// Indices are VN Stock Sim's four tracked market indices (Main.dc.html's
// four index cards).
var Indices = []string{"VN-Index", "VN30", "HNX-Index", "UPCOM-Index"}

const sparklinePoints = 20

// GetIndex synthesizes a deterministic index value, same technique as
// closeFor in provider.go (a pure function of (name, t), not an
// iterative walk) so repeated calls at nearly the same instant report
// identical numbers -- see phase-0-mvp.md's 2026-09-21 price-consistency bug
// for why this matters.
func (p *MockProvider) GetIndex(name string) IndexSnapshot {
	now := time.Now().Unix()
	step := int64(24 * 60 * 60)
	today := now - now%step

	sparkline := make([]float64, 0, sparklinePoints)
	for i := sparklinePoints - 1; i >= 0; i-- {
		sparkline = append(sparkline, round2(indexValueFor(name, today-int64(i)*step)))
	}

	value := indexValueFor(name, today)
	yesterday := indexValueFor(name, today-step)
	change := value - yesterday
	var changePercent float64
	if yesterday != 0 {
		changePercent = change / yesterday * 100
	}

	return IndexSnapshot{
		Name:          name,
		Value:         round2(value),
		Change:        round2(change),
		ChangePercent: round2(changePercent),
		Sparkline:     sparkline,
	}
}

// indexValueFor mirrors closeFor's sine-wave + jitter technique with an
// index-specific seed and base level, so each of the four indices moves
// independently but reproducibly.
func indexValueFor(name string, t int64) float64 {
	seed := seedFromSymbol(name)
	base := 800.0 + float64(seed%700) // spreads the four indices across a plausible 800-1500 range

	days := float64(t) / 86400.0
	period1 := 50.0 + float64(seed%40)
	period2 := 12.0 + float64((seed/7)%18)
	phase1 := float64(seed%628) / 100.0
	phase2 := float64((seed/13)%628) / 100.0

	wave := 0.12*math.Sin(2*math.Pi*days/period1+phase1) +
		0.05*math.Sin(2*math.Pi*days/period2+phase2)
	jitter := pseudoRandom(name, t) * 0.2

	return base * (1 + wave + jitter)
}
