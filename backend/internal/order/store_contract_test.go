package order

import (
	"os"
	"reflect"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db/dbtest"
)

func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	t.Run("memory", func(t *testing.T) { run(t, NewMemoryStore()) })
	if os.Getenv("TEST_DATABASE_URL") != "" {
		t.Run("postgres", func(t *testing.T) { run(t, NewPGStore(dbtest.Pool(t, "order"))) })
	}
}

func TestStoreContract(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		m := s.Append("u1", Order{PortfolioID: "pf_1", Symbol: "FPT", Side: "buy", Type: "market", Quantity: 100, Status: "filled", FilledPrice: 52000, Fee: 7.8, FilledAt: "2026-09-25T01:00:00Z", CreatedAt: "2026-09-25T01:00:00Z"})
		q := s.Append("u1", Order{PortfolioID: "pf_1", Symbol: "FPT", Side: "buy", Type: "oco", Quantity: 0.025, Status: "queued", Price: 51000, StopPrice: 53000, CreatedAt: "2026-09-25T02:00:00Z"})
		q2 := s.Append("u2", Order{PortfolioID: "pf_2", Symbol: "VCB", Side: "sell", Type: "limit", Quantity: 10, Status: "queued", Price: 90000, CreatedAt: "2026-09-25T01:30:00Z"})
		if m.ID == q.ID || m.ID[:4] != "ord_" {
			t.Fatalf("ids: %q %q", m.ID, q.ID)
		}
		list := s.List("u1")
		if len(list) != 2 || list[0].ID != m.ID || list[1].Quantity != 0.025 || list[1].StopPrice != 53000 {
			t.Fatalf("list: %+v", list)
		}
		queued := s.Queued()
		if len(queued) != 2 || queued[0].order.ID != q2.ID || queued[0].userID != "u2" || queued[1].order.ID != q.ID {
			t.Fatalf("queued across users, oldest first: %+v", queued)
		}
		q.Status, q.FilledPrice, q.Fee, q.FilledAt, q.TriggeredBy = "filled", 53000, 1.3, "2026-09-25T02:05:00Z", "stop"
		s.Replace("u1", q)
		got, ok := s.ByID("u1", q.ID)
		if !ok || !reflect.DeepEqual(got, q) {
			t.Fatalf("replace round trip: %+v", got)
		}
		if _, ok := s.ByID("u2", q.ID); ok {
			t.Fatal("another user's order must not be found")
		}
		if len(s.Queued()) != 1 {
			t.Fatal("a filled order leaves the queue")
		}

		// A buy with its plan and coach-mode violations round-trips.
		withPlan := s.Append("u1", Order{PortfolioID: "pf_1", Symbol: "HPG", Side: "buy", Type: "limit", Quantity: 200, Status: "queued",
			Price: 27000, StopLoss: 25500, Target: 30500, Setup: "Breakout nền giá", Violations: []string{"risk_over_target"},
			CreatedAt: "2026-09-25T03:00:00Z"})
		withPlan.Status, withPlan.PlanID = "filled", "tp_1"
		s.Replace("u1", withPlan)
		if got, _ := s.ByID("u1", withPlan.ID); !reflect.DeepEqual(got, withPlan) {
			t.Fatalf("plan fields round trip: %+v", got)
		}
	})
}

func TestStoreContractTradePlans(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		if _, ok := s.OpenPlan("pf_1", "FPT"); ok {
			t.Fatal("no plans yet")
		}
		p := s.AppendPlan("u1", TradePlan{PortfolioID: "pf_1", Symbol: "FPT", Setup: "Pullback", PlannedEntry: 100_000,
			InitialStop: 94_000, CurrentStop: 94_000, Target: 112_000, OpenedAt: "2026-09-25T01:00:00Z"})
		other := s.AppendPlan("u2", TradePlan{PortfolioID: "pf_2", Symbol: "FPT", PlannedEntry: 1, InitialStop: 0.5, CurrentStop: 0.5, OpenedAt: "x"})
		if p.ID == other.ID || p.ID[:3] != "tp_" {
			t.Fatalf("ids: %q %q", p.ID, other.ID)
		}
		got, ok := s.OpenPlan("pf_1", "FPT")
		if !ok || got != p {
			t.Fatalf("open plan: %+v", got)
		}

		// The initial stop is immutable: an update can't move it.
		p.CurrentStop, p.InitialStop = 97_000, 1
		s.UpdatePlan(p)
		got, _ = s.OpenPlan("pf_1", "FPT")
		if got.InitialStop != 94_000 || got.CurrentStop != 97_000 {
			t.Fatalf("after update: %+v", got)
		}

		got.ClosedAt, got.ExitPrice, got.RMultiple = "2026-10-01T02:00:00Z", 112_000, 2
		s.UpdatePlan(got)
		if _, ok := s.OpenPlan("pf_1", "FPT"); ok {
			t.Fatal("a closed plan is not open")
		}
		plans := s.Plans("u1")
		if len(plans) != 1 || plans[0] != got {
			t.Fatalf("u1's plans: %+v", plans)
		}
	})
}
