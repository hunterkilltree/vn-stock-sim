package portfolio

// Portfolio is one named paper-trading account a user owns. A user can
// hold several (Account-Menu.dc.html shows switching between a main
// stock portfolio, a strategy-test portfolio, and a crypto wallet) --
// see phase-b.md decision 1. Currency is "VND" for market: "stock" and
// "USDT" for market: "crypto" (crypto portfolios land in Phase I).
type Portfolio struct {
	ID              string  `json:"id"`
	UserID          string  `json:"userId"`
	Name            string  `json:"name"`
	Market          string  `json:"market"`
	StartingCapital float64 `json:"startingCapital"`
	Currency        string  `json:"currency"`
	CreatedAt       string  `json:"createdAt"`
}

type Position struct {
	Symbol        string  `json:"symbol"`
	Quantity      int64   `json:"quantity"`
	AvgCost       float64 `json:"avgCost"`
	LastPrice     float64 `json:"lastPrice"`
	MarketValue   float64 `json:"marketValue"`
	UnrealizedPnl float64 `json:"unrealizedPnl"`
}

type Summary struct {
	CashBalance          float64 `json:"cashBalance"`
	MarketValue          float64 `json:"marketValue"`
	TotalEquity          float64 `json:"totalEquity"`
	UnrealizedPnl        float64 `json:"unrealizedPnl"`
	UnrealizedPnlPercent float64 `json:"unrealizedPnlPercent"`
}

// StartingCash is the virtual capital every new paper-trading account
// opens with (vn-stock-sim-version-highlights.md: "Starting cash, current
// holdings, market value, and profit/loss — always visible").
const StartingCash = 100_000_000
