package portfolio

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
