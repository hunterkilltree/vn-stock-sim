package watchlist

type Item struct {
	Symbol  string `json:"symbol"`
	AddedAt string `json:"addedAt"`
	// CompanyName/Exchange let a list render without a second lookup per
	// row (phase-k.md decision 11); Exchange is "CRYPTO" for pairs.
	CompanyName   string  `json:"companyName"`
	Exchange      string  `json:"exchange"`
	LastPrice     float64 `json:"lastPrice"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Volume        int64   `json:"volume"`
}

type addRequest struct {
	Symbol string `json:"symbol" binding:"required"`
}
