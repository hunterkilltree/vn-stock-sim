package order

import (
	"math"
	"testing"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// fakeBars serves fixed 5-minute bars per symbol, whatever the window.
type fakeBars map[string][]market.Bar

func (f fakeBars) IntradayBars(sym string, from, to int64) []market.Bar {
	var out []market.Bar
	for _, b := range f[sym] {
		if b.Time >= from && b.Time <= to {
			out = append(out, b)
		}
	}
	return out
}

// placedAt rewrites an order's creation time so bar times in tests are
// predictable (Create stamps time.Now()).
func placedAt(t *testing.T, svc *Service, id string, at time.Time) {
	t.Helper()
	o, ok := svc.store.ByID("u1", id)
	if !ok {
		t.Fatalf("order %s not found", id)
	}
	o.CreatedAt = formatTime(at)
	svc.store.Replace("u1", o)
}

func bar(t0 time.Time, i int, open, high, low, close float64) market.Bar {
	return market.Bar{Time: t0.Unix() + int64(i*matchStep), Open: open, High: high, Low: low, Close: close}
}

func TestBarTriggerTable(t *testing.T) {
	b := market.Bar{Open: 100, High: 104, Low: 97, Close: 101}
	gapDown := market.Bar{Open: 90, High: 92, Low: 89, Close: 91}
	cases := []struct {
		name  string
		o     Order
		b     market.Bar
		price float64
		leg   string
		hit   bool
	}{
		{"buy limit touched", Order{Side: "buy", Type: "limit", Price: 98}, b, 98, "", true},
		{"buy limit not reached", Order{Side: "buy", Type: "limit", Price: 96}, b, 0, "", false},
		{"buy limit gap fills at open", Order{Side: "buy", Type: "limit", Price: 95}, gapDown, 90, "", true},
		{"sell limit touched", Order{Side: "sell", Type: "limit", Price: 103}, b, 103, "", true},
		{"buy stop", Order{Side: "buy", Type: "stop", Price: 103}, b, 103, "", true},
		{"sell stop", Order{Side: "sell", Type: "stop", Price: 98}, b, 98, "", true},
		{"sell stop gap fills at open", Order{Side: "sell", Type: "stop", Price: 95}, gapDown, 90, "", true},
		{"sell OCO take-profit", Order{Side: "sell", Type: "oco", Price: 103, StopPrice: 90}, b, 103, "limit", true},
		{"sell OCO stop", Order{Side: "sell", Type: "oco", Price: 110, StopPrice: 98}, b, 98, "stop", true},
		{"sell OCO both in one bar -> stop", Order{Side: "sell", Type: "oco", Price: 103, StopPrice: 98}, b, 98, "stop", true},
		{"buy OCO limit leg", Order{Side: "buy", Type: "oco", Price: 98, StopPrice: 110}, b, 98, "limit", true},
	}
	for _, c := range cases {
		price, leg, hit := barTrigger(c.o, c.b)
		if hit != c.hit || price != c.price || leg != c.leg {
			t.Errorf("%s: got (%v, %q, %v), want (%v, %q, %v)", c.name, price, leg, hit, c.price, c.leg, c.hit)
		}
	}
}

func TestQueuedLimitFillsOnLaterBarWithFee(t *testing.T) {
	svc, pfs, stock, _ := setup()
	t0 := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC) // 10:00 in Hanoi
	svc.EnableMatching(fakeBars{"FPT": {
		bar(t0, -1, 49000, 49100, 47000, 49000), // before the order: must be ignored
		bar(t0, 0, 50000, 50200, 49900, 50100),
		bar(t0, 1, 50100, 50100, 49400, 49600), // touches 49,500
		bar(t0, 2, 49600, 49700, 49000, 49100),
	}})
	// Quote is 50,000, so a buy limit at 49,500 waits.
	o, err := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "limit", Quantity: 100, Price: 49500})
	if err != nil || o.Status != "queued" {
		t.Fatalf("limit below the price must queue: %+v %v", o, err)
	}
	placedAt(t, svc, o.ID, t0)

	if n := svc.MatchPending(t0.Add(3 * time.Minute)); n != 0 {
		t.Fatalf("nothing should fill before the touching bar, %d did", n)
	}
	if n := svc.MatchPending(t0.Add(20 * time.Minute)); n != 1 {
		t.Fatalf("want 1 fill, got %d", n)
	}
	got, _ := svc.Get("u1", o.ID)
	wantFee := 49500 * 100 * FeeRate
	if got.Status != "filled" || got.FilledPrice != 49500 || math.Abs(got.Fee-wantFee) > 0.01 {
		t.Fatalf("filled order: %+v", got)
	}
	if got.FilledAt != formatTime(t0.Add(5*time.Minute)) {
		t.Fatalf("filledAt should be the touching bar's time, got %s", got.FilledAt)
	}
	cash := pfs.Summary(stock.ID).CashBalance
	if want := 100_000_000 - 4_950_000 - wantFee; math.Abs(cash-want) > 0.01 {
		t.Fatalf("cash: want %v (value + fee), got %v", want, cash)
	}
	// Filled orders are never matched again.
	if n := svc.MatchPending(t0.Add(time.Hour)); n != 0 {
		t.Fatalf("a filled order matched again")
	}
}

func TestMarketableOnArrivalFillsAtOnce(t *testing.T) {
	svc, _, stock, wallet := setup()
	o, err := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "limit", Quantity: 100, Price: 51000})
	if err != nil || o.Status != "filled" || o.FilledPrice != 50000 {
		t.Fatalf("a buy limit above the price fills at the price: %+v %v", o, err)
	}
	// BTC is at 60,000: a sell OCO whose stop is already above it fires the stop leg.
	if _, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 0.01}); err != nil {
		t.Fatal(err)
	}
	oco, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "sell", Type: "oco", Quantity: 0.01, Price: 70000, StopPrice: 61000})
	if err != nil || oco.Status != "filled" || oco.TriggeredBy != "stop" {
		t.Fatalf("OCO with its stop already crossed fills the stop leg: %+v %v", oco, err)
	}
	if _, err := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "limit", Quantity: 100}); err == nil {
		t.Fatal("a limit order without a price must be refused")
	}
}

func TestOCOFillsOneLegOnly(t *testing.T) {
	svc, _, _, wallet := setup()
	t0 := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	svc.EnableMatching(fakeBars{"BTCUSDT": {
		bar(t0, 0, 60000, 60500, 59800, 60400),
		bar(t0, 1, 60400, 63200, 60300, 63000), // take-profit at 63,000
		bar(t0, 2, 63000, 63100, 50000, 51000), // would also hit the stop later
	}})
	if _, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 0.02}); err != nil {
		t.Fatal(err)
	}
	o, _ := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "sell", Type: "oco", Quantity: 0.02, Price: 63000, StopPrice: 55000})
	placedAt(t, svc, o.ID, t0)
	svc.MatchPending(t0.Add(time.Hour))
	got, _ := svc.Get("u1", o.ID)
	if got.Status != "filled" || got.TriggeredBy != "limit" || got.FilledPrice != 63000 {
		t.Fatalf("OCO should take profit once: %+v", got)
	}
}

func TestRejectedWhenCashRunsOut(t *testing.T) {
	svc, _, stock, _ := setup()
	t0 := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	svc.EnableMatching(fakeBars{"FPT": {bar(t0, 0, 50000, 50000, 40000, 41000)}})
	// 2,000 shares at 45,000 = 90,000,000: affordable when placed...
	o, _ := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "limit", Quantity: 2000, Price: 45000})
	placedAt(t, svc, o.ID, t0)
	// ...but a market buy spends most of the cash first (no reservation, decision 8).
	if _, err := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "market", Quantity: 1000}); err != nil {
		t.Fatal(err)
	}
	svc.MatchPending(t0.Add(10 * time.Minute))
	got, _ := svc.Get("u1", o.ID)
	if got.Status != "rejected" || got.RejectReason == "" {
		t.Fatalf("want rejected with a reason, got %+v", got)
	}
}

func TestCancelledOrderNeverFills(t *testing.T) {
	svc, _, stock, _ := setup()
	t0 := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	svc.EnableMatching(fakeBars{"FPT": {bar(t0, 0, 50000, 50000, 40000, 41000)}})
	o, _ := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "limit", Quantity: 100, Price: 45000})
	placedAt(t, svc, o.ID, t0)
	if _, err := svc.Cancel("u1", o.ID); err != nil {
		t.Fatal(err)
	}
	if n := svc.MatchPending(t0.Add(time.Hour)); n != 0 {
		t.Fatal("a cancelled order filled")
	}
}

func TestATCFillsAtTheClose(t *testing.T) {
	cases := []struct {
		placed, want string // Hanoi local times
	}{
		{"2026-09-24 10:00", "2026-09-24 14:45"}, // Thursday morning -> same day
		{"2026-09-24 15:00", "2026-09-25 14:45"}, // after the close -> Friday
		{"2026-09-25 16:00", "2026-09-28 14:45"}, // Friday evening -> Monday
		{"2026-09-26 09:00", "2026-09-28 14:45"}, // Saturday -> Monday
	}
	for _, c := range cases {
		placed, _ := time.ParseInLocation("2006-01-02 15:04", c.placed, vnZone)
		want, _ := time.ParseInLocation("2006-01-02 15:04", c.want, vnZone)
		if got := atcClose(placed); !got.Equal(want) {
			t.Errorf("ATC placed %s: close %s, want %s", c.placed, got.In(vnZone).Format("2006-01-02 15:04"), c.want)
		}
	}

	svc, _, stock, _ := setup()
	svc.EnableMatching(fakeBars{})
	o, _ := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "atc", Quantity: 100})
	placed, _ := time.ParseInLocation("2006-01-02 15:04", "2026-09-24 10:00", vnZone)
	placedAt(t, svc, o.ID, placed)
	if svc.MatchPending(placed.Add(4*time.Hour)) != 0 {
		t.Fatal("ATC must wait for 14:45")
	}
	svc.MatchPending(placed.Add(5 * time.Hour))
	got, _ := svc.Get("u1", o.ID)
	if got.Status != "filled" || got.FilledPrice != 50000 {
		t.Fatalf("ATC after the close fills at the last price: %+v", got)
	}
}
