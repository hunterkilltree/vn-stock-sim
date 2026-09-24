// Package screener backs the Main screen's sector heatmap and top-movers
// tables (Main.dc.html), and later the standalone screener (Phase K).
package screener

// TickerChange is one symbol's live day change, used both inside a
// SectorGroup's tile list (Price/Volume left zero there -- the heatmap
// tiles only ever show symbol+percent) and in the top-movers tables
// (design/screens/Main.dc.html's Ma/Gia/+-/KL columns, which need both).
type TickerChange struct {
	Symbol        string  `json:"symbol"`
	ChangePercent float64 `json:"changePercent"`
	Price         float64 `json:"price,omitempty"`
	Volume        int64   `json:"volume,omitempty"`
	// Set on heatmap tiles (phase-i.md decision 15): the full heatmap
	// sizes tiles by market cap and shows name/exchange on hover.
	CompanyName string `json:"companyName,omitempty"`
	Exchange    string `json:"exchange,omitempty"`
	MarketCap   int64  `json:"marketCap,omitempty"`
}

// SectorGroup is one sector's heatmap row: its average day change plus
// the individual tickers in it, so the frontend can color-intensity-code
// both the sector row and each ticker tile.
type SectorGroup struct {
	Sector           string         `json:"sector"`
	AvgChangePercent float64        `json:"avgChangePercent"`
	Tickers          []TickerChange `json:"tickers"`
	MarketCap        int64          `json:"marketCap"`
	Up               int            `json:"up"`
	Down             int            `json:"down"`
	Flat             int            `json:"flat"`
}

// Periods the heatmap can colour by, as sessions back from the latest
// daily close ("1D" uses the quote's own day change).
var PeriodSessions = map[string]int{"1D": 1, "1W": 5, "1M": 21, "3M": 63}
