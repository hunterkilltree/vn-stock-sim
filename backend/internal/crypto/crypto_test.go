package crypto

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeBinance speaks Binance's real public REST formats (strings for
// numbers, klines as arrays) and answers 400/-1121 for unknown symbols.
func fakeBinance(t *testing.T, calls *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		sym := r.URL.Query().Get("symbol")
		if sym != "BTCUSDT" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"code":-1121,"msg":"Invalid symbol."}`)
			return
		}
		switch r.URL.Path {
		case "/api/v3/ticker/24hr":
			_, _ = io.WriteString(w, `{"symbol":"BTCUSDT","priceChange":"1358.20","priceChangePercent":"2.140","lastPrice":"64820.50","highPrice":"65140.00","lowPrice":"62980.00","volume":"438112.10","quoteVolume":"28400000000.00"}`)
		case "/api/v3/klines":
			if r.URL.Query().Get("interval") != "1h" {
				t.Errorf("interval not passed through: %v", r.URL.Query())
			}
			_, _ = io.WriteString(w, `[[1619827200000,"57714.66","58459.07","57419.00","58184.07","1937.20",1619830799999,"112000000",1,"1","1","0"],`+
				`[1619830800000,"58184.07","58300.00","57900.00","57988.70","1402.00",1619834399999,"81000000",1,"1","1","0"]]`)
		case "/api/v3/depth":
			_, _ = io.WriteString(w, `{"lastUpdateId":1,"bids":[["64820.50","1.842"],["64810.00","0.924"]],"asks":[["64830.00","0.474"],["64840.50","0.684"]]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestBinanceAdapterParsesRealFormats(t *testing.T) {
	var calls int32
	srv := fakeBinance(t, &calls)
	defer srv.Close()
	b := &BinanceProvider{baseURL: srv.URL, client: srv.Client()}
	btc, _ := Lookup("BTC/USDT")

	tk, err := b.Ticker(btc)
	if err != nil || tk.LastPrice != 64820.5 || tk.ChangePercent != 2.14 || tk.QuoteVolume24h != 28.4e9 || tk.Source != "binance" {
		t.Fatalf("ticker: %+v %v", tk, err)
	}
	bars, err := b.Klines(btc, "1h", 1619827200, 1619834399)
	if err != nil || len(bars) != 2 || bars[0].Time != 1619827200 || bars[1].Close != 57988.7 || bars[0].Volume != 1937 {
		t.Fatalf("klines: %+v %v", bars, err)
	}
	bids, asks, err := b.Depth(btc, 20)
	if err != nil || bids[0].Price != 64820.5 || bids[0].Size != 1.842 || asks[1].Price != 64840.5 {
		t.Fatalf("depth: %+v %+v %v", bids, asks, err)
	}
	eth, _ := Lookup("ETH")
	if _, err := b.Ticker(eth); err != ErrNoData {
		t.Fatalf("unknown symbol (400/-1121) must be ErrNoData, got %v", err)
	}
}

func TestLiveProviderFallsBackAndMarksSource(t *testing.T) {
	var calls int32
	srv := fakeBinance(t, &calls)
	defer srv.Close()
	live := NewLiveProvider(&BinanceProvider{baseURL: srv.URL, client: srv.Client()})

	btc, _ := Lookup("BTCUSDT")
	if tk := live.Ticker(btc); tk.Source != "binance" || tk.LastPrice != 64820.5 {
		t.Fatalf("listed pair should come from Binance: %+v", tk)
	}
	eth, _ := Lookup("ETHUSDT")
	if tk := live.Ticker(eth); tk.Source != "mock" || tk.LastPrice <= 0 {
		t.Fatalf("a pair Binance rejects must fall back to mock: %+v", tk)
	}
	if live.liveDown() {
		t.Fatal("an unknown-symbol answer must not trip the breaker")
	}
	before := atomic.LoadInt32(&calls)
	okb, _ := Lookup("OKB")
	if tk := live.Ticker(okb); tk.Source != "mock" || atomic.LoadInt32(&calls) != before {
		t.Fatal("unlisted pairs (OKB) must never query Binance")
	}
}

func TestBreakerSkipsLiveAfterConnectionFailure(t *testing.T) {
	live := NewLiveProvider(&BinanceProvider{baseURL: "http://127.0.0.1:1", client: http.DefaultClient})
	btc, _ := Lookup("BTCUSDT")
	if tk := live.Ticker(btc); tk.Source != "mock" {
		t.Fatalf("unreachable Binance must fall back: %+v", tk)
	}
	if !live.liveDown() {
		t.Fatal("a connection failure must trip the breaker")
	}
}

func (l *LiveProvider) liveDown() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.downUntil.After(timeNow())
}

func TestMockIsContinuousAndDeterministic(t *testing.T) {
	btc, _ := Lookup("BTCUSDT")
	var m MockProvider
	// A full week including a weekend: 168 hourly bars, no gaps.
	from := int64(1619827200) // Sat 2021-05-01 00:00 UTC
	bars, _ := m.Klines(btc, "1h", from, from+167*3600)
	if len(bars) != 168 {
		t.Fatalf("want 168 hourly bars over a week incl. weekend, got %d", len(bars))
	}
	for i := 1; i < len(bars); i++ {
		if bars[i].Time-bars[i-1].Time != 3600 {
			t.Fatalf("gap at %d", i)
		}
		if bars[i].Open != bars[i-1].Close {
			t.Fatalf("open must equal previous close at %d", i)
		}
	}
	again, _ := m.Klines(btc, "1h", from+24*3600, from+48*3600)
	if again[0].Close != bars[24].Close {
		t.Fatal("same instant must give the same price regardless of window")
	}
}

func TestOrderBookCumulativeAndHeatmapOrder(t *testing.T) {
	svc := NewService(NewLiveProvider(nil))
	ob, ok := svc.OrderBook("BTCUSDT", 4)
	if !ok || len(ob.Bids) != 4 || len(ob.Asks) != 4 || ob.Source != "mock" {
		t.Fatalf("book: %+v", ob)
	}
	if ob.Bids[3].Cumulative <= ob.Bids[0].Cumulative || ob.Bids[0].Price <= ob.Bids[1].Price || ob.Asks[0].Price >= ob.Asks[1].Price {
		t.Fatalf("levels must run outward from the best price with a running total: %+v", ob)
	}
	groups := svc.Heatmap()
	var names []string
	total := 0
	for _, g := range groups {
		names = append(names, g.Sector)
		total += len(g.Tickers)
	}
	if strings.Join(names, "|") != "Layer 1|DeFi|Sàn & hạ tầng|Dự án Việt Nam" || total != 24 {
		t.Fatalf("design's 4 categories and 24 coins expected, got %v (%d)", names, total)
	}
	o := svc.Overview()
	if o.PairCount != 24 || o.BTCDominancePercent <= 0 || o.BTCDominancePercent >= 100 {
		t.Fatalf("overview: %+v", o)
	}
}

func TestQuoteRouter(t *testing.T) {
	r := QuoteRouter{Crypto: NewService(NewLiveProvider(nil)), Stock: stubStock{}}
	d, ok := r.Detail("BTCUSDT")
	if !ok || d.Exchange != "CRYPTO" || d.Sector != "Layer 1" || d.LastPrice <= 0 {
		t.Fatalf("crypto detail: %+v", d)
	}
	d, ok = r.Detail("FPT")
	if !ok || d.Exchange != "HOSE" {
		t.Fatalf("stock symbols must be delegated: %+v", d)
	}
}
