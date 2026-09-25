package watchlist

import (
	"errors"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

type quotes struct{}

func (quotes) Detail(sym string) (symbol.Detail, bool) {
	switch sym {
	case "FPT":
		return symbol.Detail{Symbol: symbol.Symbol{Symbol: "FPT", CompanyName: "FPT Corporation", Exchange: "HOSE"}, LastPrice: 52260, ChangePercent: 1.37}, true
	case "BTCUSDT":
		return symbol.Detail{Symbol: symbol.Symbol{Symbol: "BTCUSDT", CompanyName: "Bitcoin", Exchange: symbol.ExchangeCrypto}, LastPrice: 60000}, true
	}
	return symbol.Detail{}, false
}

// phase-k.md decision 11: canonical symbols, names on each item, unknown
// symbols refused, crypto pairs allowed.
func TestAddListRemove(t *testing.T) {
	svc := NewService(NewMemoryStore(), quotes{})
	if err := svc.Add("u1", "nope"); !errors.Is(err, ErrUnknownSymbol) {
		t.Fatalf("unknown symbol: want ErrUnknownSymbol, got %v", err)
	}
	for _, s := range []string{" fpt ", "FPT", "btcusdt"} {
		if err := svc.Add("u1", s); err != nil {
			t.Fatal(err)
		}
	}
	items := svc.List("u1")
	if len(items) != 2 || items[0].Symbol != "FPT" || items[0].CompanyName != "FPT Corporation" || items[0].Exchange != "HOSE" || items[1].Symbol != "BTCUSDT" {
		t.Fatalf("want FPT then BTCUSDT with names, deduplicated: %+v", items)
	}
	svc.Remove("u1", "fpt")
	if items := svc.List("u1"); len(items) != 1 || items[0].Symbol != "BTCUSDT" {
		t.Fatalf("after removing fpt: %+v", items)
	}
}
