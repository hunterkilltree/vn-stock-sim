package crypto

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// BinanceProvider reads Binance's public spot market data -- no API key.
// The default base URL is data-api.binance.vision, Binance's
// market-data-only host (phase-i.md decision 1). Response formats:
// https://developers.binance.com/docs/binance-spot-api-docs/rest-api/market-data-endpoints
type BinanceProvider struct {
	baseURL string
	client  *http.Client
}

func NewBinanceProvider() *BinanceProvider {
	return &BinanceProvider{baseURL: "https://data-api.binance.vision", client: &http.Client{Timeout: 8 * time.Second}}
}

// get decodes a successful response into out. A 400 is Binance's answer
// for an unknown symbol (code -1121), which is ErrNoData; any other
// failure is returned as-is (and trips LiveProvider's breaker).
func (b *BinanceProvider) get(path string, q url.Values, out any) error {
	resp, err := b.client.Get(b.baseURL + path + "?" + q.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		return ErrNoData
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func num(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (b *BinanceProvider) Ticker(p Pair) (Ticker, error) {
	var r struct {
		PriceChange        string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
		LastPrice          string `json:"lastPrice"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
	}
	if err := b.get("/api/v3/ticker/24hr", url.Values{"symbol": {p.Symbol}}, &r); err != nil {
		return Ticker{}, err
	}
	t := Ticker{
		LastPrice: num(r.LastPrice), Change: num(r.PriceChange), ChangePercent: num(r.PriceChangePercent),
		High24h: num(r.HighPrice), Low24h: num(r.LowPrice), Volume24h: num(r.Volume), QuoteVolume24h: num(r.QuoteVolume),
		Source: "binance",
	}
	if t.LastPrice <= 0 {
		return Ticker{}, ErrNoData
	}
	return t, nil
}

// Klines pages through /api/v3/klines (max 1000 bars per call). Each row
// is [openTime(ms), open, high, low, close, volume, closeTime, ...] with
// prices as strings.
func (b *BinanceProvider) Klines(p Pair, interval string, from, to int64) ([]market.Bar, error) {
	if !validInterval(interval) {
		return nil, ErrNoData
	}
	var out []market.Bar
	start := from * 1000
	for page := 0; page < 5; page++ {
		var rows [][]json.RawMessage
		q := url.Values{
			"symbol": {p.Symbol}, "interval": {interval}, "limit": {"1000"},
			"startTime": {strconv.FormatInt(start, 10)}, "endTime": {strconv.FormatInt(to*1000, 10)},
		}
		if err := b.get("/api/v3/klines", q, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			if len(row) < 6 {
				continue
			}
			var openMs int64
			var o, h, l, c, v string
			if json.Unmarshal(row[0], &openMs) != nil || json.Unmarshal(row[1], &o) != nil || json.Unmarshal(row[2], &h) != nil ||
				json.Unmarshal(row[3], &l) != nil || json.Unmarshal(row[4], &c) != nil || json.Unmarshal(row[5], &v) != nil {
				continue
			}
			out = append(out, market.Bar{Time: openMs / 1000, Open: num(o), High: num(h), Low: num(l), Close: num(c), Volume: int64(num(v))})
		}
		if len(rows) < 1000 {
			break
		}
		start = out[len(out)-1].Time*1000 + 1
	}
	if len(out) == 0 {
		return nil, ErrNoData
	}
	return out, nil
}

func (b *BinanceProvider) Depth(p Pair, limit int) ([]DepthLevel, []DepthLevel, error) {
	var r struct {
		Bids [][2]string `json:"bids"`
		Asks [][2]string `json:"asks"`
	}
	if err := b.get("/api/v3/depth", url.Values{"symbol": {p.Symbol}, "limit": {strconv.Itoa(limit)}}, &r); err != nil {
		return nil, nil, err
	}
	conv := func(in [][2]string) []DepthLevel {
		out := make([]DepthLevel, 0, len(in))
		for _, lv := range in {
			out = append(out, DepthLevel{Price: num(lv[0]), Size: num(lv[1])})
		}
		return out
	}
	bids, asks := conv(r.Bids), conv(r.Asks)
	if len(bids) == 0 || len(asks) == 0 {
		return nil, nil, ErrNoData
	}
	return bids, asks, nil
}
