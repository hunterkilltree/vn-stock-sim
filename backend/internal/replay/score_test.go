package replay

import (
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// Replay's session NAV pays the same fees the portfolio ledger charges
// (phase-k.md decision 3): 0.15% for stocks, 0.10% for crypto, per fill.
func TestReplayAccountingChargesFees(t *testing.T) {
	bars := []market.Bar{{Time: 1, Close: 100}, {Time: 2, Close: 110}, {Time: 3, Close: 120}}
	sess := &Session{
		Capital:    1_000_000,
		Market:     "stock",
		Bars:       bars,
		CurrentBar: 3,
		Fills: []Fill{
			{BarIndex: 0, Side: "buy", Quantity: 1000, Price: 100},  // 100,000 + fee 150
			{BarIndex: 2, Side: "sell", Quantity: 1000, Price: 120}, // 120,000 - fee 180
		},
	}
	nav, pnl, _ := replayAccounting(sess)
	if nav != 1_019_670 {
		t.Fatalf("stock NAV: want 1,019,670 (20,000 gain - 330 fees), got %v", nav)
	}
	if pnl != 1.96 {
		t.Fatalf("stock P&L %%: want 1.96, got %v", pnl)
	}

	sess.Market = "crypto"
	sess.Capital = 10_000
	sess.Fills = []Fill{{BarIndex: 0, Side: "buy", Quantity: 10, Price: 100}} // 1,000 + fee 1
	nav, _, _ = replayAccounting(sess)
	if nav != 10_199 { // 9,000 - 1 cash + 10 x 120
		t.Fatalf("crypto NAV: want 10,199, got %v", nav)
	}
}
