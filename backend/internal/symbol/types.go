package symbol

type Symbol struct {
	Symbol      string `json:"symbol"`
	CompanyName string `json:"companyName"`
	Exchange    string `json:"exchange"`
	Sector      string `json:"sector"`
	TickSize    int    `json:"tickSize"`
}

// Detail adds the pricing/fundamentals fields api-spec.md documents for
// GET /symbols/:symbol (the stock detail page).
type Detail struct {
	Symbol
	LastPrice     float64 `json:"lastPrice"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	MarketCap     int64   `json:"marketCap"`
	PERatio       float64 `json:"peRatio"`
	PBRatio       float64 `json:"pbRatio"`
	EPS           float64 `json:"eps"`
	DividendYield float64 `json:"dividendYield"`
}
