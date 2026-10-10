package regime

// Levels of the market risk regime, "Nhiệt kế thị trường"
// (phase-market-risk.md feature 1).
const (
	LevelNormal   = "normal"    // Bình thường
	LevelCaution  = "caution"   // Thận trọng
	LevelHighRisk = "high_risk" // Rủi ro cao

	ZoneOK      = "ok"
	ZoneCaution = "caution"
	ZoneRisk    = "risk"
	ZoneNoData  = "nodata" // shown, not counted
)

// Signal is one measured condition, with the threshold it was judged by
// and why it matters, so the card teaches rather than just alarms.
type Signal struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	Zone  string `json:"zone"`
	Rule  string `json:"rule"`
	Why   string `json:"why"`
}

// Regime is GET /market/regime. Score sums the zones (caution 1, risk 2)
// over signals that have data.
type Regime struct {
	Level       string   `json:"level"`
	Score       int      `json:"score"`
	MaxScore    int      `json:"maxScore"`
	Knockout    string   `json:"knockout,omitempty"`
	Signals     []Signal `json:"signals"`
	IndexName   string   `json:"indexName"`
	IndexValue  float64  `json:"indexValue"`
	AsOf        int64    `json:"asOf"`
	Universe    int      `json:"universe"`
	Source      string   `json:"source"`
	GeneratedAt string   `json:"generatedAt"`
}
