package order

import (
	"errors"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

type quotes struct{}

func (quotes) Detail(sym string) (symbol.Detail, bool) {
	if sym == "BTCUSDT" {
		return symbol.Detail{Symbol: symbol.Symbol{Symbol: "BTCUSDT", Exchange: symbol.ExchangeCrypto}, LastPrice: 60000}, true
	}
	return symbol.Detail{Symbol: symbol.Symbol{Symbol: sym, Exchange: "HOSE"}, LastPrice: 50000}, true
}

func setup() (*Service, *portfolio.Service, portfolio.Portfolio, portfolio.Portfolio) {
	pfs := portfolio.NewService(portfolio.NewMemoryStore(), quotes{})
	svc := NewService(NewMemoryStore(), pfs, quotes{}, pfs)
	pfs.SetOrdersPort(svc)
	stock := pfs.CreatePortfolio("u1", "Danh mục chính", "stock", 100_000_000, "")
	wallet := pfs.CreatePortfolio("u1", "Ví crypto", "crypto", 0, "")
	return svc, pfs, stock, wallet
}

func TestCryptoWalletDefaults(t *testing.T) {
	_, pfs, stock, wallet := setup()
	if wallet.Currency != "USDT" || wallet.StartingCapital != portfolio.StartingUSDT {
		t.Fatalf("crypto wallet should open with 10,000 USDT: %+v", wallet)
	}
	if pfs.DefaultPortfolioID("u1") != stock.ID {
		t.Fatal("the default portfolio must stay the stock one")
	}
}

func TestMarketsCannotMix(t *testing.T) {
	svc, _, stock, wallet := setup()
	_, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "FPT", Side: "buy", Type: "market", Quantity: 100})
	if !errors.Is(err, ErrWrongMarket) {
		t.Fatalf("stock ticker into a crypto wallet: want ErrWrongMarket, got %v", err)
	}
	_, err = svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 0.01})
	if !errors.Is(err, ErrWrongMarket) {
		t.Fatalf("crypto pair into a stock portfolio: want ErrWrongMarket, got %v", err)
	}
}

func TestQuantityRulesAndCryptoFee(t *testing.T) {
	svc, pfs, stock, wallet := setup()
	if _, err := svc.Create("u1", createRequest{PortfolioID: stock.ID, Symbol: "FPT", Side: "buy", Type: "market", Quantity: 10.5}); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("fractional shares must be rejected, got %v", err)
	}
	o, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 0.025})
	if err != nil {
		t.Fatal(err)
	}
	// 0.025 BTC x 60,000 = 1,500 USDT; fee 0.10% = 1.50 USDT.
	if o.Quantity != 0.025 || o.Fee != 1.5 {
		t.Fatalf("want 0.025 BTC with a 1.50 USDT fee, got %+v", o)
	}
	sum := pfs.Summary(wallet.ID)
	if sum.CashBalance != 8500 || sum.MarketValue != 1500 {
		t.Fatalf("wallet after buy: %+v", sum)
	}
	for _, q := range []float64{0.01, 0.015} {
		if _, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "sell", Type: "market", Quantity: q}); err != nil {
			t.Fatal(err)
		}
	}
	if pos := pfs.Positions(wallet.ID); len(pos) != 0 {
		t.Fatalf("selling 0.01 + 0.015 of 0.025 must close the position, left %+v", pos)
	}
}

func TestOCONeedsBothPrices(t *testing.T) {
	svc, _, _, wallet := setup()
	if _, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "sell", Type: "oco", Quantity: 0.01, Price: 70000}); !errors.Is(err, ErrOCOPrices) {
		t.Fatalf("OCO without a stop price: want ErrOCOPrices, got %v", err)
	}
	o, err := svc.Create("u1", createRequest{PortfolioID: wallet.ID, Symbol: "BTCUSDT", Side: "sell", Type: "oco", Quantity: 0.01, Price: 70000, StopPrice: 55000})
	if err != nil || o.Status != "queued" || o.StopPrice != 55000 {
		t.Fatalf("OCO should be queued with both prices: %+v %v", o, err)
	}
}
