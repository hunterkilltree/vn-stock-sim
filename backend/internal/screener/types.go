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
}

// SectorGroup is one sector's heatmap row: its average day change plus
// the individual tickers in it, so the frontend can color-intensity-code
// both the sector row and each ticker tile.
type SectorGroup struct {
	Sector           string         `json:"sector"`
	AvgChangePercent float64        `json:"avgChangePercent"`
	Tickers          []TickerChange `json:"tickers"`
}
