package portfolio

import (
	"errors"
	"math"
	"os"
	"sync"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db/dbtest"
)

// addUser creates the users row a Postgres portfolio references; the
// memory store needs nothing.
type storeCase struct {
	s       Store
	addUser func(id string)
}

func eachStore(t *testing.T, run func(t *testing.T, c storeCase)) {
	t.Run("memory", func(t *testing.T) { run(t, storeCase{s: NewMemoryStore(), addUser: func(string) {}}) })
	if os.Getenv("TEST_DATABASE_URL") != "" {
		t.Run("postgres", func(t *testing.T) {
			pool := dbtest.Pool(t, "portfolio")
			run(t, storeCase{s: NewPGStore(pool), addUser: func(id string) { dbtest.User(t, pool, id) }})
		})
	}
}

func TestStoreContractPortfolios(t *testing.T) {
	eachStore(t, func(t *testing.T, c storeCase) {
		s := c.s
		c.addUser("u1")
		c.addUser("u2")
		wallet := s.Create("u1", "Ví crypto", "crypto", 10_000, "USDT", KindTrading)
		rp := s.Create("u1", "Replay", "stock", 1e8, "VND", KindReplay)
		// Neither a wallet nor a replay portfolio can be the default: the
		// first ask opens "Danh mục chính" (phase-i.md decision 9).
		def := s.DefaultFor("u1")
		if def == wallet.ID || def == rp.ID || def == "" {
			t.Fatalf("default: %q", def)
		}
		if s.DefaultFor("u1") != def {
			t.Fatal("the default is stable")
		}
		second := s.Create("u1", "Thử RSI", "stock", 5e8, "VND", KindTrading)
		if s.DefaultFor("u1") != def {
			t.Fatal("a later stock portfolio must not replace the default")
		}
		list := s.List("u1")
		if len(list) != 4 || list[0].ID != wallet.ID || list[3].ID != second.ID {
			t.Fatalf("list in creation order: %+v", list)
		}
		if p, ok := s.Get("u1", def); !ok || p.Name != "Danh mục chính" || p.StartingCapital != StartingCash {
			t.Fatalf("lazy default: %+v", p)
		}
		if _, ok := s.Get("u2", def); ok {
			t.Fatal("another user's portfolio must not be found")
		}
		if owner, ok := s.OwnerOf(wallet.ID); !ok || owner != "u1" {
			t.Fatalf("owner: %q", owner)
		}
		if h := s.EquityHistory(second.ID); len(h) != 1 || h[0].NAV != 5e8 {
			t.Fatalf("history starts at the starting capital: %+v", h)
		}
	})
}

func TestStoreContractLedger(t *testing.T) {
	eachStore(t, func(t *testing.T, c storeCase) {
		s := c.s
		c.addUser("u1")
		p := s.Create("u1", "Ví", "crypto", 10_000, "USDT", KindTrading)
		if err := s.ApplyFill(p.ID, "BTCUSDT", "buy", 0.1, 60_000, 6); err != nil {
			t.Fatal(err)
		}
		if err := s.ApplyFill(p.ID, "BTCUSDT", "buy", 0.05, 66_000, 3.3); err != nil {
			t.Fatal(err)
		}
		pos := s.Positions(p.ID)["BTCUSDT"]
		if math.Abs(pos.quantity-0.15) > 1e-9 || math.Abs(pos.avgCost-62_000) > 1e-6 {
			t.Fatalf("blended position: %+v", pos)
		}
		if cash := s.Cash(p.ID); math.Abs(cash-(10_000-6_000-6-3_300-3.3)) > 1e-6 {
			t.Fatalf("cash after buys: %v", cash)
		}
		// Value + fee must fit: 690.7 left, 0.011 BTC at 62,800 = 690.8.
		if err := s.ApplyFill(p.ID, "ETHUSDT", "buy", 0.011, 62_800, 0); !errors.Is(err, ErrInsufficientCash) {
			t.Fatalf("want ErrInsufficientCash, got %v", err)
		}
		if err := s.ApplyFill(p.ID, "BTCUSDT", "sell", 0.2, 70_000, 0); !errors.Is(err, ErrInsufficientShares) {
			t.Fatalf("want ErrInsufficientShares, got %v", err)
		}
		// Selling 0.1 + 0.05 closes the position despite float rounding.
		if err := s.ApplyFill(p.ID, "BTCUSDT", "sell", 0.1, 70_000, 7); err != nil {
			t.Fatal(err)
		}
		if err := s.ApplyFill(p.ID, "BTCUSDT", "sell", 0.05, 70_000, 3.5); err != nil {
			t.Fatal(err)
		}
		if len(s.Positions(p.ID)) != 0 {
			t.Fatalf("position should be closed: %+v", s.Positions(p.ID))
		}
		if cash := s.Cash(p.ID); math.Abs(cash-(690.7+7000-7+3500-3.5)) > 1e-6 {
			t.Fatalf("cash after sells: %v", cash)
		}
		s.AppendEquitySnapshot(p.ID, 11_180.2)
		if h := s.EquityHistory(p.ID); len(h) != 2 || h[1].NAV != 11_180.2 {
			t.Fatalf("equity history: %+v", h)
		}
	})
}

// phase-persistence.md decision 7: concurrent fills can't overspend.
// 20 buys of 1,000 each against 10,000 of cash: exactly 10 succeed.
func TestStoreContractConcurrentFills(t *testing.T) {
	eachStore(t, func(t *testing.T, c storeCase) {
		s := c.s
		c.addUser("u1")
		p := s.Create("u1", "Ví", "crypto", 10_000, "USDT", KindTrading)
		var wg sync.WaitGroup
		var mu sync.Mutex
		ok := 0
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if s.ApplyFill(p.ID, "BTCUSDT", "buy", 0.01, 100_000, 0) == nil {
					mu.Lock()
					ok++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		pos := s.Positions(p.ID)["BTCUSDT"]
		if ok != 10 || math.Abs(s.Cash(p.ID)) > 1e-6 || math.Abs(pos.quantity-0.1) > 1e-9 {
			t.Fatalf("want 10 fills, 0 cash, 0.1 BTC; got %d fills, cash %v, %+v", ok, s.Cash(p.ID), pos)
		}
	})
}
