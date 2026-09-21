package watchlist

type Item struct {
	Symbol        string  `json:"symbol"`
	AddedAt       string  `json:"addedAt"`
	LastPrice     float64 `json:"lastPrice"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Volume        int64   `json:"volume"`
}

type addRequest struct {
	Symbol string `json:"symbol" binding:"required"`
}
