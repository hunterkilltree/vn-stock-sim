package order

import (
	"errors"
	"testing"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/riskrules"
)

// The quotes stub prices every stock at 50,000; the portfolio has 100m.
func buy(qty, stop, target float64) createRequest {
	return createRequest{Symbol: "FPT", Side: "buy", Type: "market", Quantity: qty, StopLoss: stop, Target: target, Setup: "Pullback"}
}

func TestStockBuysAreWholeLots(t *testing.T) {
	svc, _, stock, _ := setup()
	r := buy(150, 48_000, 56_000)
	r.PortfolioID = stock.ID
	if _, err := svc.Create("u1", r); !errors.Is(err, ErrOddLot) {
		t.Fatalf("150 shares: want ErrOddLot, got %v", err)
	}
}

func TestPlanOpensAddsAndClosesWithR(t *testing.T) {
	svc, pfs, stock, _ := setup()
	r := buy(200, 48_000, 56_000)
	r.PortfolioID = stock.ID
	o, err := svc.Create("u1", r)
	if err != nil || o.PlanID == "" || len(o.Violations) != 0 {
		t.Fatalf("first buy: %+v %v", o, err)
	}
	// Adding without a stop joins the plan and inherits its stop: no
	// missing_stop warning.
	add := buy(100, 0, 0)
	add.PortfolioID = stock.ID
	o2, err := svc.Create("u1", add)
	if err != nil || o2.PlanID != o.PlanID || len(o2.Violations) != 0 {
		t.Fatalf("add: %+v %v", o2, err)
	}
	// A partial sell keeps the plan open; selling the rest closes it.
	for _, q := range []float64{100, 200} {
		if _, err := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "sell", Type: "market", Quantity: q}); err != nil {
			t.Fatal(err)
		}
		if _, open := svc.store.OpenPlan(stock.ID, "FPT"); open != (q == 100) {
			t.Fatalf("after selling %v: open = %v", q, open)
		}
	}
	plans := svc.Plans("u1")
	// Exit at 50,000 = the entry: 0 R. Initial stop kept.
	if len(plans) != 1 || plans[0].InitialStop != 48_000 || plans[0].ClosedAt == "" || plans[0].RMultiple != 0 || plans[0].ExitPrice != 50_000 {
		t.Fatalf("plans: %+v", plans)
	}
	if len(pfs.Positions(stock.ID)) != 0 {
		t.Fatal("position should be flat")
	}
}

func TestCoachRecordsViolationsStrictRefuses(t *testing.T) {
	svc, pfs, stock, _ := setup()
	noStop := buy(100, 0, 0)
	noStop.PortfolioID = stock.ID
	o, err := svc.Create("u1", noStop)
	if err != nil || o.PlanID != "" || len(o.Violations) != 2 {
		t.Fatalf("coach: want a fill with missing_stop and missing_target recorded, got %+v %v", o, err)
	}

	if err := pfs.SetRiskSettings(stock.ID, riskrules.Settings{Mode: riskrules.Strict, RiskPct: 1, MaxRiskPct: 2, MinRewardRisk: 2}); err != nil {
		t.Fatal(err)
	}
	other := noStop
	other.Symbol = "HPG"
	var rv *RuleViolation
	if _, err := svc.Create("u1", other); !errors.As(err, &rv) || rv.Findings[0].Code != "missing_stop" {
		t.Fatalf("strict without a stop: want RuleViolation, got %v", err)
	}
	// 1,000 shares risking 2,000 each = 2.0m + fees > 2% of ~100m.
	big := buy(1_000, 48_000, 60_000)
	big.PortfolioID, big.Symbol = stock.ID, "HPG"
	if _, err := svc.Create("u1", big); !errors.As(err, &rv) {
		t.Fatalf("strict over 2%%: want RuleViolation, got %v", err)
	}
	// The dry run reports the same thing, plus the size that fits 1%.
	a, err := svc.Check("u1", big)
	if err != nil || !a.Blocked || a.SuggestedQty != 400 {
		t.Fatalf("check: %+v %v", a, err)
	}
}

// A queued limit buy opens its plan when the matcher fills it.
func TestMatcherFillOpensPlan(t *testing.T) {
	svc, _, stock, _ := setup()
	r := createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "limit", Quantity: 100, Price: 49_000, StopLoss: 46_000, Target: 56_000}
	o, err := svc.Create("u1", r)
	if err != nil || o.Status != "queued" || o.PlanID != "" {
		t.Fatalf("queued: %+v %v", o, err)
	}
	svc.mu.Lock()
	detail, _ := quotes{}.Detail("FPT")
	svc.book("u1", &o, detail, 49_000, "", time.Date(2026, 9, 24, 3, 5, 0, 0, time.UTC))
	svc.mu.Unlock()
	plan, ok := svc.store.OpenPlan(stock.ID, "FPT")
	if !ok || o.PlanID != plan.ID || plan.PlannedEntry != 49_000 || plan.InitialStop != 46_000 {
		t.Fatalf("plan after fill: %+v (order %+v)", plan, o)
	}
}
