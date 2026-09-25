package replay

import (
	"os"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db/dbtest"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	t.Run("memory", func(t *testing.T) { run(t, NewMemoryStore()) })
	if os.Getenv("TEST_DATABASE_URL") != "" {
		t.Run("postgres", func(t *testing.T) { run(t, NewPGStore(dbtest.Pool(t, "replay"))) })
	}
}

// A session is saved whole and comes back at the same candle
// (phase-persistence.md decision 8).
func TestStoreContract(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		id := s.nextSessionID()
		if id2 := s.nextSessionID(); id == id2 || id[:3] != "rp_" {
			t.Fatalf("ids: %q %q", id, id2)
		}
		sess := &Session{ID: id, UserID: "u1", Symbol: "HPG", Market: "stock", Capital: 1e8, TotalBars: 3, CurrentBar: 1, Status: "active",
			Bars: []market.Bar{{Time: 1, Close: 10}, {Time: 2, Close: 11}, {Time: 3, Close: 12}}}
		s.Append("u1", sess)

		got, ok := s.Get("u1", id)
		if !ok {
			t.Fatal("not found")
		}
		got.CurrentBar = 2
		got.Fills = append(got.Fills, Fill{BarIndex: 1, Side: "buy", Quantity: 100, Price: 11, StopSet: 10})
		got.StopLoss = 10
		s.Save(got)

		again, ok := s.Get("u1", id)
		if !ok || again.CurrentBar != 2 || len(again.Fills) != 1 || again.StopLoss != 10 || len(again.Bars) != 3 || again.Bars[2].Close != 12 {
			t.Fatalf("after save: %+v", again)
		}
		if _, ok := s.Get("u2", id); ok {
			t.Fatal("another user's session must not be found")
		}
	})
}
