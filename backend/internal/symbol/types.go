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
	ROE           float64 `json:"roe"`
	EPS           float64 `json:"eps"`
	DividendYield float64 `json:"dividendYield"`
	// Reference (tham chieu), Ceiling (tran), and Floor (san) are the real
	// HOSE/HNX/UPCOM daily price-band rules, derived from the previous
	// close by a fixed percentage band per exchange -- see
	// priceBandPercent in service.go. Populated only when the quote
	// source has data (same condition as LastPrice/Change below); zero
	// otherwise.
	Reference float64 `json:"reference"`
	Ceiling   float64 `json:"ceiling"`
	Floor     float64 `json:"floor"`
}

// ExchangeCrypto marks a Detail that describes a crypto pair (served by
// crypto.QuoteRouter), so order/portfolio can tell the markets apart
// without importing the crypto package (phase-i.md decisions 7-8).
const ExchangeCrypto = "CRYPTO"
