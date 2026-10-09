package rating

// Horizon verdicts (phase-holding-horizon.md decision 1).
const (
	VerdictSuitable = "suitable" // Phù hợp nắm giữ
	VerdictWatch    = "watch"    // Cần theo dõi
	VerdictAvoid    = "avoid"    // Chưa phù hợp
)

// Factor is one scored input to a horizon verdict. Available is false
// when the data behind it doesn't exist yet (the H2 fields, or too
// little history); it then has no weight in the score.
type Factor struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Value     string  `json:"value"`
	Score     float64 `json:"score"`
	Weight    float64 `json:"weight"`
	Available bool    `json:"available"`
	Why       string  `json:"why"`
}

// Risk is what holding has felt like: annualised volatility, the worst
// peak-to-trough fall over the horizon's lookback, and (medium only) a
// reference stop-loss.
type Risk struct {
	VolatilityPct  float64 `json:"volatilityPct"`
	MaxDrawdownPct float64 `json:"maxDrawdownPct"`
	DrawdownWindow string  `json:"drawdownWindow"`
	StopLoss       float64 `json:"stopLoss,omitempty"`
	StopPct        float64 `json:"stopPct,omitempty"`
}

// HoldStats: if you'd bought on any past day and held Label long.
// Returns are percent; Index is VN-Index over the same windows.
type HoldStats struct {
	Label   string     `json:"label"`
	Days    int        `json:"days"`
	Samples int        `json:"samples"`
	WinRate float64    `json:"winRate"`
	Median  float64    `json:"median"`
	P10     float64    `json:"p10"`
	P90     float64    `json:"p90"`
	Index   *HoldStats `json:"index,omitempty"`
}

type Horizon struct {
	Verdict  string      `json:"verdict"`
	Score    float64     `json:"score"`
	Knockout string      `json:"knockout,omitempty"`
	Partial  bool        `json:"partial"`
	Factors  []Factor    `json:"factors"`
	Risk     Risk        `json:"risk"`
	History  []HoldStats `json:"history"`
}

type Horizons struct {
	Medium Horizon `json:"medium"`
	Long   Horizon `json:"long"`
}
