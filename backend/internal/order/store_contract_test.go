package order

import (
	"os"
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
		if !ok || got != q {
			t.Fatalf("replace round trip: %+v", got)
		}
		if _, ok := s.ByID("u2", q.ID); ok {
			t.Fatal("another user's order must not be found")
		}
		if len(s.Queued()) != 1 {
			t.Fatal("a filled order leaves the queue")
		}
	})
}
