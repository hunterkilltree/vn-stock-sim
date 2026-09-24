// Package crypto is the crypto market (Phase I): the pair universe, live
// market data from Binance's public API with a deterministic fallback,
// and the overview/heatmap/movers/order-book views the Crypto screens
// draw. It is deliberately separate from the stock symbol/market
// packages (FULL-APP-PLAN.md 2.3) -- see phase-i.md.
package crypto

// Pair is one tradable USDT pair. Symbol is the exchange form
// ("BTCUSDT"), which is also how positions and orders store it, so it
// can never collide with a VN stock ticker (phase-i.md decision 6).
type Pair struct {
	Symbol   string `json:"symbol"`
	Base     string `json:"base"`
	Quote    string `json:"quote"`
	Name     string `json:"name"`
	Category string `json:"category"`
	// Hand-seeded facts Binance's market-data API doesn't provide
	// (phase-i.md decision 3). MaxSupply 0 means no fixed cap.
	CirculatingSupply float64 `json:"circulatingSupply"`
	MaxSupply         float64 `json:"maxSupply"`
	// listed is false for pairs Binance doesn't trade (OKB, VNDC): those
	// are always served by the mock generator and marked source "mock".
	listed    bool
	seedPrice float64
	seedATH   float64
}

// Ticker is a pair's rolling 24-hour market data.
type Ticker struct {
	LastPrice      float64 `json:"lastPrice"`
	Change         float64 `json:"change"`
	ChangePercent  float64 `json:"changePercent"`
	High24h        float64 `json:"high24h"`
	Low24h         float64 `json:"low24h"`
	Volume24h      float64 `json:"volume24h"`
	QuoteVolume24h float64 `json:"quoteVolume24h"`
	// Source is "binance" or "mock" -- never mixed silently.
	Source string `json:"source"`
}

type PairQuote struct {
	Pair
	Ticker
	MarketCap float64 `json:"marketCap"`
}

// PairDetail adds Crypto-Detail.dc.html's coin facts.
type PairDetail struct {
	PairQuote
	AllTimeHigh             float64 `json:"allTimeHigh"`
	DistanceFromATHPercent  float64 `json:"distanceFromAthPercent"`
	Volatility30dPercentDay float64 `json:"volatility30dPercentDay"`
}

// DepthLevel is one order-book level with the running total from the
// best price outward (Crypto-Detail's "Cộng dồn" column).
type DepthLevel struct {
	Price      float64 `json:"price"`
	Size       float64 `json:"size"`
	Cumulative float64 `json:"cumulative"`
}

type OrderBook struct {
	Bids   []DepthLevel `json:"bids"`
	Asks   []DepthLevel `json:"asks"`
	Source string       `json:"source"`
}

// HeatTile / HeatGroup use the same JSON field names as the stock
// heatmap (screener.SectorGroup) so the frontend heatmap renders both.
type HeatTile struct {
	Symbol        string  `json:"symbol"`
	CompanyName   string  `json:"companyName"`
	Exchange      string  `json:"exchange"`
	ChangePercent float64 `json:"changePercent"`
	Price         float64 `json:"price"`
	Volume        int64   `json:"volume"`
	MarketCap     int64   `json:"marketCap"`
}

type HeatGroup struct {
	Sector           string     `json:"sector"`
	AvgChangePercent float64    `json:"avgChangePercent"`
	Tickers          []HeatTile `json:"tickers"`
	MarketCap        int64      `json:"marketCap"`
	Up               int        `json:"up"`
	Down             int        `json:"down"`
	Flat             int        `json:"flat"`
}

// Overview backs Crypto-Main's four cards. The total/dominance figures
// are over the app's own pair universe, not the global market
// (phase-i.md decision 4) -- PairCount says how many.
type Overview struct {
	BTC                     PairQuote `json:"btc"`
	ETH                     PairQuote `json:"eth"`
	BTCSparkline            []float64 `json:"btcSparkline"`
	ETHSparkline            []float64 `json:"ethSparkline"`
	TotalMarketCap          float64   `json:"totalMarketCap"`
	TotalMarketCapChangePct float64   `json:"totalMarketCapChangePercent"`
	BTCDominancePercent     float64   `json:"btcDominancePercent"`
	ETHDominancePercent     float64   `json:"ethDominancePercent"`
	PairCount               int       `json:"pairCount"`
	TotalQuoteVolume24h     float64   `json:"totalQuoteVolume24h"`
	CategoriesInOrder       []string  `json:"categories"`
}
