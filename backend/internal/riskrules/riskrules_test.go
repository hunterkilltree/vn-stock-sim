package riskrules

import (
	"math"
	"testing"
)

func codes(a Assessment) map[string]Severity {
	out := map[string]Severity{}
	for _, f := range a.Findings {
		out[f.Code] = f.Severity
	}
	return out
}

func strict() Settings {
	s := Defaults()
	s.Mode = Strict
	return s
}

// The spec's sizing: qty = equity × 1% ÷ (entry − stop), down to 100.
// Without fees: 100,000,000 × 1% ÷ (50,000 − 46,000) = 250 → 200.
func TestSizeForSpecExample(t *testing.T) {
	if got := SizeFor(100_000_000, 1, 4_000, 100); got != 200 {
		t.Fatalf("SizeFor = %v, want 200", got)
	}
	if got := SizeFor(100_000_000, 1, 3_900, 100); got != 200 { // 256.4
		t.Fatalf("SizeFor = %v, want 200", got)
	}
	if got := SizeFor(10_000, 1, 300, 0); got != 0.33333333 {
		t.Fatalf("crypto SizeFor = %v, want 0.33333333", got)
	}
}

func TestCheckRiskIncludesFeesBothSides(t *testing.T) {
	in := Intent{Side: "buy", Quantity: 200, Entry: 50_000, Stop: 46_000, Target: 58_000, FeeRate: 0.0015, Lot: 100}
	a := Check(in, 100_000_000, Defaults())
	// Per share: 4,000 + 0.0015 × 96,000 = 4,144; × 200 = 828,800.
	if math.Abs(a.RiskAmount-828_800) > 1e-6 || math.Abs(a.RiskPct-0.8288) > 1e-9 {
		t.Fatalf("risk = %v (%v%%)", a.RiskAmount, a.RiskPct)
	}
	// 1,000,000 ÷ 4,144 = 241.3 → 200.
	if a.SuggestedQty != 200 {
		t.Fatalf("suggested = %v", a.SuggestedQty)
	}
	// R:R = 8,000 ÷ 4,000 = 2: exactly the minimum, no finding.
	if a.RewardRisk != 2 || len(a.Findings) != 0 || a.Blocked {
		t.Fatalf("assessment = %+v", a)
	}
}

func TestMissingStopWarnsInCoachBlocksInStrict(t *testing.T) {
	in := Intent{Side: "buy", Quantity: 100, Entry: 50_000, Target: 60_000, Lot: 100}
	if a := Check(in, 1e8, Defaults()); a.Blocked || codes(a)["missing_stop"] != Warn {
		t.Fatalf("coach: %+v", a)
	}
	if a := Check(in, 1e8, strict()); !a.Blocked || codes(a)["missing_stop"] != Block {
		t.Fatalf("strict: %+v", a)
	}
}

func TestRiskOverMax(t *testing.T) {
	// 1,000 shares × 4,000 = 4,000,000 = 4% of 100m (no fees here).
	in := Intent{Side: "buy", Quantity: 1_000, Entry: 50_000, Stop: 46_000, Target: 60_000, Lot: 100}
	if a := Check(in, 1e8, Defaults()); a.Blocked || codes(a)["risk_over_max"] != Warn {
		t.Fatalf("coach: %+v", a)
	}
	if a := Check(in, 1e8, strict()); !a.Blocked || codes(a)["risk_over_max"] != Block {
		t.Fatalf("strict: %+v", a)
	}
	// 1.5%: above the 1% plan, under the 2% cap -- a warning in both modes.
	in.Quantity = 375
	if a := Check(in, 1e8, strict()); a.Blocked || codes(a)["risk_over_target"] != Warn {
		t.Fatalf("1.5%%: %+v", a)
	}
}

// Elder's 6% rule in numbers: three full 2% losses use up the month.
func TestThreeTwoPercentLossesReachSixPercent(t *testing.T) {
	equity := 100_000_000.0
	var lost float64
	for i := 0; i < 3; i++ {
		qty := SizeFor(equity, 2, 1_000, 0)
		lost += qty * 1_000
		equity -= qty * 1_000
	}
	if pct := lost / 100_000_000 * 100; pct < 5.88 || pct > 6 {
		t.Fatalf("three 2%% losses = %.2f%%", pct)
	}
}

func TestInvalidPlansBlockInBothModes(t *testing.T) {
	for _, s := range []Settings{Defaults(), strict()} {
		a := Check(Intent{Side: "buy", Quantity: 100, Entry: 50_000, Stop: 51_000, Target: 60_000}, 1e8, s)
		if !a.Blocked || codes(a)["stop_not_below_entry"] != Invalid {
			t.Fatalf("%s stop above entry: %+v", s.Mode, a)
		}
		a = Check(Intent{Side: "buy", Quantity: 100, Entry: 50_000, Stop: 46_000, Target: 49_000}, 1e8, s)
		if !a.Blocked || codes(a)["target_not_above_entry"] != Invalid {
			t.Fatalf("%s target below entry: %+v", s.Mode, a)
		}
	}
}

func TestLowRewardRiskOnlyWarns(t *testing.T) {
	in := Intent{Side: "buy", Quantity: 100, Entry: 50_000, Stop: 46_000, Target: 54_000, Lot: 100}
	a := Check(in, 1e8, strict())
	if a.Blocked || codes(a)["low_reward_risk"] != Warn || a.RewardRisk != 1 {
		t.Fatalf("%+v", a)
	}
	if a := Check(Intent{Side: "buy", Quantity: 100, Entry: 50_000, Stop: 46_000, Lot: 100}, 1e8, strict()); codes(a)["missing_target"] != Warn || a.Blocked {
		t.Fatalf("missing target: %+v", a)
	}
}

func TestSellsAreNotChecked(t *testing.T) {
	if a := Check(Intent{Side: "sell", Quantity: 100, Entry: 50_000}, 1e8, strict()); a.Blocked || len(a.Findings) != 0 {
		t.Fatalf("%+v", a)
	}
}

func TestSettingsValidate(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []Settings{
		{Mode: "lax", RiskPct: 1, MaxRiskPct: 2, MinRewardRisk: 2},
		{Mode: Coach, RiskPct: 0, MaxRiskPct: 2, MinRewardRisk: 2},
		{Mode: Coach, RiskPct: 3, MaxRiskPct: 2, MinRewardRisk: 2},
		{Mode: Coach, RiskPct: 1, MaxRiskPct: 2, MinRewardRisk: -1},
	}
	for _, s := range bad {
		if s.Validate() == nil {
			t.Errorf("%+v should be invalid", s)
		}
	}
}
