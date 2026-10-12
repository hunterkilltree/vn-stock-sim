package stress

import (
	"math"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

const day = 24 * 60 * 60

// series turns closes into daily bars starting at day `start`.
func series(start int, closes []float64) []market.Bar {
	bars := make([]market.Bar, len(closes))
	for i, c := range closes {
		bars[i] = market.Bar{Time: int64(start+i) * day, Close: c}
	}
	return bars
}

// wave: flat at 100, falls to 70 over 30 days (−30%), recovers to 100.
func crashCloses() []float64 {
	var c []float64
	for i := 0; i < 100; i++ {
		c = append(c, 100+0.1*math.Sin(float64(i))) // tiny noise so beta has variance
	}
	for i := 1; i <= 30; i++ {
		c = append(c, 100-float64(i))
	}
	for i := 1; i <= 30; i++ {
		c = append(c, 70+float64(i))
	}
	return c
}

func TestEpisodesFindsTheCrash(t *testing.T) {
	eps := episodes(series(0, crashCloses()))
	if len(eps) != 1 {
		t.Fatalf("episodes = %+v", eps)
	}
	if math.Abs(eps[0].pct-(-30)) > 0.5 || eps[0].to != 129*day {
		t.Fatalf("episode = %+v", eps[0])
	}
	// A shallow 10% dip is not an episode.
	shallow := []float64{100, 95, 90, 95, 100}
	if got := episodes(series(0, shallow)); len(got) != 0 {
		t.Fatalf("shallow dip = %+v", got)
	}
}

func TestHistoricalScenarioUsesRealPath(t *testing.T) {
	idx := series(0, crashCloses())
	// The stock falls twice as hard as the index through the same crash.
	sc := crashCloses()
	stock := make([]float64, len(sc))
	for i, c := range sc {
		stock[i] = 100 * math.Pow(c/100, 2)
	}
	res := Compute(Input{Cash: 50e6, Index: idx, Holdings: []Holding{{Symbol: "AAA", Value: 50e6, Band: 0.07, Bars: series(0, stock)}}})
	hist := res.Scenarios[0]
	if hist.Kind != KindHistorical {
		t.Fatalf("first scenario = %+v", hist)
	}
	h := hist.Holdings[0]
	if h.Estimated || math.Abs(h.ChangePct-(-51)) > 0.5 { // 0.7^2 = 0.49
		t.Fatalf("holding = %+v", h)
	}
	// Half the account is cash: a 51% stock loss is ~25.5% of equity.
	if math.Abs(hist.LossPct-25.5) > 0.3 {
		t.Fatalf("loss pct = %v", hist.LossPct)
	}
	if b := res.Betas[0]; math.Abs(b.Beta-2) > 0.1 || b.Samples < minBetaSamples {
		t.Fatalf("beta = %+v", b)
	}
}

func TestMissingHistoryFallsBackToBeta(t *testing.T) {
	idx := series(0, crashCloses())
	// A stock listed after the crash: no path for the episode, and too few
	// returns for a beta -> beta 1, estimated.
	late := series(140, []float64{50, 51, 52})
	res := Compute(Input{Index: idx, Holdings: []Holding{{Symbol: "NEW", Value: 10e6, Band: 0.07, Bars: late}}})
	h := res.Scenarios[0].Holdings[0]
	if !h.Estimated || math.Abs(h.ChangePct-(-30)) > 0.5 || res.Betas[0].Samples != 0 {
		t.Fatalf("holding = %+v, beta = %+v", h, res.Betas[0])
	}
}

func TestHypotheticalShocks(t *testing.T) {
	res := Compute(Input{Cash: 0, Holdings: []Holding{{Symbol: "X", Value: 100e6, Band: 0.07}}})
	byKey := map[string]Scenario{}
	for _, s := range res.Scenarios {
		byKey[s.Key] = s
	}
	if s := byKey["shock_20"]; s.LossPct != 20 || s.LossVND != 20e6 {
		t.Fatalf("shock_20 = %+v", s)
	}
	// Beta 1 × −80% = −80%, and never worse than −100%.
	if s := byKey["like_2008"]; s.LossPct != 80 {
		t.Fatalf("like_2008 = %+v", s)
	}
	if got := shocked(3, -80); got != -100 {
		t.Fatalf("shocked(3, -80) = %v", got)
	}
}

func TestFloorSessions(t *testing.T) {
	// HOSE: three straight −7% days, a −2% day, then one more −6.8% day
	// (within tolerance of the floor).
	closes := []float64{100, 93, 86.49, 80.44, 78.83, 73.47}
	days, longest := floorSessions(series(0, closes), -1, 10*day, 0.07)
	if days != 4 || longest != 3 {
		t.Fatalf("days %d longest %d", days, longest)
	}
}

func TestWorstHoldingFirst(t *testing.T) {
	res := Compute(Input{Holdings: []Holding{
		{Symbol: "SMALL", Value: 1e6, Band: 0.07},
		{Symbol: "BIG", Value: 9e6, Band: 0.07},
	}})
	if res.Scenarios[0].Holdings[0].Symbol != "BIG" {
		t.Fatalf("order = %+v", res.Scenarios[0].Holdings)
	}
}

func TestLowBetaIsFlooredInShocks(t *testing.T) {
	// A defensive-looking beta of 0.3 still takes the full index shock.
	if got := shocked(0.3, -40); got != -40 {
		t.Fatalf("shocked(0.3, -40) = %v, want -40", got)
	}
	if got := shocked(1.5, -40); got != -60 {
		t.Fatalf("shocked(1.5, -40) = %v, want -60", got)
	}
}
