package screener

import (
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

type fakeSymbols struct{ details []symbol.Detail }

func (f fakeSymbols) Search(_, exchange string, _, _ int) ([]symbol.Symbol, int) {
	out := []symbol.Symbol{}
	for _, d := range f.details {
		if exchange == "" || d.Exchange == exchange {
			out = append(out, d.Symbol)
		}
	}
	return out, len(out)
}

func (f fakeSymbols) Detail(sym string) (symbol.Detail, bool) {
	for _, d := range f.details {
		if d.Symbol.Symbol == sym {
			return d, true
		}
	}
	return symbol.Detail{}, false
}

// fakeMarket: every symbol's last 10 daily closes rise from 100 to 110
// except LOSS, which falls from 100 to 90.
type fakeMarket struct{}

func (fakeMarket) LatestQuote(string) (float64, int64, bool) { return 0, 1000, true }
func (fakeMarket) GetBars(sym, _ string, _, _ int64) []market.Bar {
	bars := make([]market.Bar, 11)
	for i := range bars {
		c := 100 + float64(i)
		if sym == "LOSS" {
			c = 100 - float64(i)
		}
		bars[i] = market.Bar{Time: int64(i), Close: c}
	}
	return bars
}

func det(sym, exchange, sector string, pct float64, capBn int64) symbol.Detail {
	return symbol.Detail{Symbol: symbol.Symbol{Symbol: sym, Exchange: exchange, Sector: sector}, ChangePercent: pct, MarketCap: capBn * 1e9}
}

func newSvc() *Service {
	return NewService(fakeSymbols{details: []symbol.Detail{
		det("SMALLBANK", "HOSE", "Ngân hàng", 1, 10),
		det("BIGBANK", "HOSE", "Ngân hàng", -2, 500),
		det("LOSS", "HNX", "Chứng khoán", 0, 20),
	}}, fakeMarket{})
}

func TestHeatmapGroupsOrdersAndCountsBreadth(t *testing.T) {
	out := newSvc().GetSectorHeatmap("1D", "")
	if len(out) != 2 || out[0].Sector != "Ngân hàng" {
		t.Fatalf("sectors must be ordered by total market cap, got %+v", out)
	}
	bank := out[0]
	if bank.Tickers[0].Symbol != "BIGBANK" || bank.MarketCap != 510e9 {
		t.Fatalf("tiles must be ordered by market cap and summed, got %+v", bank)
	}
	if bank.Up != 1 || bank.Down != 1 || bank.AvgChangePercent != -0.5 {
		t.Fatalf("breadth/average wrong: %+v", bank)
	}
	if out[1].Flat != 1 {
		t.Fatalf("0%% change counts as flat: %+v", out[1])
	}
}

func TestHeatmapPeriodUsesDailyBars(t *testing.T) {
	out := newSvc().GetSectorHeatmap("1W", "")
	for _, g := range out {
		for _, tk := range g.Tickers {
			// 5 sessions back: 110 vs 105 (+4.76%), LOSS 90 vs 95 (-5.26%).
			want := 4.76
			if tk.Symbol == "LOSS" {
				want = -5.26
			}
			if tk.ChangePercent != want {
				t.Errorf("%s: want %v, got %v", tk.Symbol, want, tk.ChangePercent)
			}
		}
	}
}

func TestHeatmapExchangeFilter(t *testing.T) {
	out := newSvc().GetSectorHeatmap("1D", "HNX")
	if len(out) != 1 || out[0].Tickers[0].Symbol != "LOSS" {
		t.Fatalf("HNX filter should keep only LOSS, got %+v", out)
	}
}
