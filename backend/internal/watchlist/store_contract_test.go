package watchlist

import (
	"os"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db/dbtest"
)

func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	t.Run("memory", func(t *testing.T) { run(t, NewMemoryStore()) })
	if os.Getenv("TEST_DATABASE_URL") != "" {
		t.Run("postgres", func(t *testing.T) { run(t, NewPGStore(dbtest.Pool(t, "watchlist"))) })
	}
}

func TestStoreContract(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		for _, sym := range []string{"FPT", "BTCUSDT", "FPT", "VCB"} {
			s.Add("u1", sym)
		}
		s.Add("u2", "HPG")
		got := s.List("u1")
		if len(got) != 3 || got[0].symbol != "FPT" || got[1].symbol != "BTCUSDT" || got[2].symbol != "VCB" || got[0].addedAt.IsZero() {
			t.Fatalf("want FPT, BTCUSDT, VCB in insertion order, deduplicated: %+v", got)
		}
		s.Remove("u1", "BTCUSDT")
		s.Remove("u1", "NOPE")
		if got := s.List("u1"); len(got) != 2 || got[1].symbol != "VCB" {
			t.Fatalf("after remove: %+v", got)
		}
		if got := s.List("u2"); len(got) != 1 {
			t.Fatalf("lists are per user: %+v", got)
		}
	})
}
