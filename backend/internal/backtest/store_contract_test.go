package backtest

import (
	"os"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db/dbtest"
)

func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	t.Run("memory", func(t *testing.T) { run(t, NewMemoryStore()) })
	if os.Getenv("TEST_DATABASE_URL") != "" {
		t.Run("postgres", func(t *testing.T) { run(t, NewPGStore(dbtest.Pool(t, "backtest"))) })
	}
}

func TestStoreContract(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		a := s.Append("u1", Backtest{Symbol: "FPT", RuleType: "rsi_reversion", Params: map[string]int{"entry": 35},
			Equity: []EquityPoint{{Time: 1, Equity: 100, Benchmark: 100}}, Trades: []Trade{{EntryTime: 1, ExitTime: 2, ReturnPercent: 5.5}}})
		b := s.Append("u1", Backtest{Symbol: "VCB", RuleType: "ema_crossover"})
		s.Append("u2", Backtest{Symbol: "HPG"})
		if a.ID == "" || a.ID == b.ID || a.ID[:3] != "bt_" {
			t.Fatalf("ids: %q %q", a.ID, b.ID)
		}
		list := s.List("u1")
		if len(list) != 2 || list[0].Symbol != "FPT" || list[1].Symbol != "VCB" {
			t.Fatalf("list in run order: %+v", list)
		}
		got, ok := s.ByID("u1", a.ID)
		if !ok || got.Params["entry"] != 35 || len(got.Equity) != 1 || got.Trades[0].ReturnPercent != 5.5 {
			t.Fatalf("round trip: %+v %v", got, ok)
		}
		if _, ok := s.ByID("u2", a.ID); ok {
			t.Fatal("another user's backtest must not be found")
		}
	})
}
