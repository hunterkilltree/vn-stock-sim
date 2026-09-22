// Package screener backs the Main screen's sector heatmap and top-movers
// tables (Main.dc.html), and later the standalone screener (Phase K).
package screener

// TickerChange is one symbol's live day change, used both inside a
// SectorGroup's tile list and in the top-movers tables.
type TickerChange struct {
	Symbol        string  `json:"symbol"`
	ChangePercent float64 `json:"changePercent"`
}

// SectorGroup is one sector's heatmap row: its average day change plus
// the individual tickers in it, so the frontend can color-intensity-code
// both the sector row and each ticker tile.
type SectorGroup struct {
	Sector           string         `json:"sector"`
	AvgChangePercent float64        `json:"avgChangePercent"`
	Tickers          []TickerChange `json:"tickers"`
}
