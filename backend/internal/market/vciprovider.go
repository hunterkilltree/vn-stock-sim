package market

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// VCIProvider is a MarketDataProvider adapter against Vietcap Securities
// (VCI)'s own trading-platform API -- a real, licensed Vietnamese
// brokerage's internal endpoint, not an officially published third-
// party developer API (no API key; same endpoint their own web app
// calls). This is a deliberate, user-approved exception to api-spec.md's
// "no scraped data" rule (see phase-vci-market-data.md) -- kept behind
// the same MarketDataProvider port as MockProvider precisely so it can
// be swapped for a licensed vendor (e.g. SSI FastConnect) later without
// touching any caller.
//
// Concrete request/response shape verified with real curl calls before
// writing this file (see phase-vci-market-data.md), not assumed from
// the vnstock Python source it was read from.
type VCIProvider struct {
	httpClient *http.Client
	baseURL    string
	fallback   *MockProvider // order book only -- see phase-vci-market-data.md decision 2
}

func NewVCIProvider() *VCIProvider {
	return &VCIProvider{
		httpClient: &http.Client{Timeout: 8 * time.Second},
		baseURL:    "https://trading.vietcap.com.vn/api/",
		fallback:   NewMockProvider(),
	}
}

// vciIndexSymbols translates this app's display names to VCI's own
// index codes (from vnstock's _VCI_INDEX_MAPPING, cross-checked live).
var vciIndexSymbols = map[string]string{
	"VN-Index":    "VNINDEX",
	"VN30":        "VN30",
	"HNX-Index":   "HNXIndex",
	"UPCOM-Index": "HNXUpcomIndex",
}

// vciTimeFrame maps this app's resolution strings to VCI's timeFrame
// values. VCI collapses 1/5/15-minute requests to the same real
// ONE_MINUTE granularity server-side (verified in the vnstock source),
// so "5"/"15" here come back as real 1-minute bars, not degraded data.
func vciTimeFrame(resolution string) string {
	switch resolution {
	case "1", "5", "15":
		return "ONE_MINUTE"
	case "60":
		return "ONE_HOUR"
	default:
		return "ONE_DAY"
	}
}

type vciChartResponse struct {
	Symbol string    `json:"symbol"`
	Open   []float64 `json:"o"`
	High   []float64 `json:"h"`
	Low    []float64 `json:"l"`
	Close  []float64 `json:"c"`
	Volume []float64 `json:"v"`
	// VCI returns these as JSON strings (confirmed by the live test in
	// phase-vci-market-data.md: "t":["1789516800", ...]), not numbers.
	Time []string `json:"t"`
}

// fetchOHLC calls VCI's real chart endpoint and returns bars covering
// (from, to]. Unexported so LiveProvider (same package) can see the
// error and decide to fall back; GetBars (below) is the public,
// interface-satisfying method that swallows the error into an empty
// slice per MarketDataProvider's contract.
func (p *VCIProvider) fetchOHLC(vciSymbol, timeFrame string, from, to int64) ([]Bar, error) {
	countBack := estimateCountBack(timeFrame, from, to)

	body, err := json.Marshal(map[string]any{
		"timeFrame": timeFrame,
		"symbols":   []string{vciSymbol},
		"to":        to,
		"countBack": countBack,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"chart/OHLCChart/gap-chart", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://trading.vietcap.com.vn/")
	req.Header.Set("Origin", "https://trading.vietcap.com.vn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vci: unexpected status %d", resp.StatusCode)
	}

	var parsed []vciChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed) == 0 {
		return nil, errors.New("vci: empty response")
	}
	row := parsed[0]
	if len(row.Time) == 0 || len(row.Time) != len(row.Close) {
		return nil, errors.New("vci: malformed response shape")
	}

	bars := make([]Bar, 0, len(row.Time))
	for i, ts := range row.Time {
		t, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			continue
		}
		if t < from {
			continue // VCI's countBack can overshoot the window; trim to what was asked for
		}
		bars = append(bars, Bar{
			Time:   t,
			Open:   round2(row.Open[i]),
			High:   round2(row.High[i]),
			Low:    round2(row.Low[i]),
			Close:  round2(row.Close[i]),
			Volume: int64(row.Volume[i]),
		})
	}
	return bars, nil
}

// estimateCountBack turns a (from, to) window into VCI's bar-count
// parameter, capped so a single request can't ask for an unbounded
// number of bars against someone else's unofficial endpoint (phase-vci-
// market-data.md decisions 5 and 8).
func estimateCountBack(timeFrame string, from, to int64) int {
	const maxCountBack = 2000
	span := to - from
	if span <= 0 {
		span = 24 * 60 * 60
	}
	var n int64
	switch timeFrame {
	case "ONE_MINUTE":
		n = span/60 + 10
	case "ONE_HOUR":
		n = span/3600 + 5
	default: // ONE_DAY
		n = span*5/(7*24*60*60) + 5 // business days, generously
	}
	if n > maxCountBack {
		n = maxCountBack
	}
	if n < 5 {
		n = 5
	}
	return int(n)
}

func (p *VCIProvider) GetBars(sym, resolution string, from, to int64) []Bar {
	vciSymbol := sym
	if mapped, ok := vciIndexSymbols[sym]; ok {
		vciSymbol = mapped
	}
	bars, err := p.fetchOHLC(vciSymbol, vciTimeFrame(resolution), from, to)
	if err != nil {
		return nil
	}
	return bars
}

// GetIndex builds a real IndexSnapshot from a real 21-daily-bar VCI
// fetch (20 for the sparkline + 1 to compute the most recent day's
// change against).
func (p *VCIProvider) GetIndex(name string) IndexSnapshot {
	vciSymbol, ok := vciIndexSymbols[name]
	if !ok {
		return IndexSnapshot{Name: name}
	}
	now := time.Now().Unix()
	from := now - 40*24*60*60 // comfortably more than 21 trading days
	bars, err := p.fetchOHLC(vciSymbol, "ONE_DAY", from, now)
	if err != nil || len(bars) < 2 {
		return IndexSnapshot{Name: name}
	}
	if len(bars) > sparklinePoints {
		bars = bars[len(bars)-sparklinePoints:]
	}

	sparkline := make([]float64, len(bars))
	for i, b := range bars {
		sparkline[i] = b.Close
	}
	last := bars[len(bars)-1]
	prev := bars[len(bars)-2]
	change := last.Close - prev.Close
	var changePercent float64
	if prev.Close != 0 {
		changePercent = change / prev.Close * 100
	}
	return IndexSnapshot{
		Name:          name,
		Value:         last.Close,
		Change:        round2(change),
		ChangePercent: round2(changePercent),
		Sparkline:     sparkline,
	}
}

// GetOrderBook stays synthetic -- see phase-vci-market-data.md decision 2.
func (p *VCIProvider) GetOrderBook(sym string, lastPrice float64) (bids, asks []PriceLevel) {
	return p.fallback.GetOrderBook(sym, lastPrice)
}
