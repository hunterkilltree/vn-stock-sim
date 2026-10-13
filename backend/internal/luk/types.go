// Package luk reviews a stock against Martin Luk's swing-trading rules
// (pullback to the 9/21 EMA and anchored VWAP inside a leader's uptrend,
// a stop under the day's low, 0.5% account risk per trade) -- see
// phase-luk-review.md for where each rule and threshold comes from and how
// it was adapted to HOSE/HNX/UPCOM.
//
// The checklist is deterministic (rules.go). An optional Gemini review
// (gemini.go) reads that checklist and writes a short commentary; it only
// exists when the operator sets GEMINI_API_KEY.
package luk

// Check statuses.
const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// Check groups, in the order a trader would ask them.
const (
	GroupSetup = "setup" // is this a leader worth trading?
	GroupEntry = "entry" // is today a low-risk entry?
	GroupRisk  = "risk"  // can the risk be kept small, and is the market with us?
)

// Verdicts.
const (
	VerdictBuy   = "buy"   // every core rule passes, nothing fails
	VerdictWatch = "watch" // a leader, but not an entry today
	VerdictAvoid = "avoid" // a core rule fails
)

// Check is one rule's outcome. Value is preformatted so the frontend
// doesn't need each rule's units. Core checks decide the verdict.
type Check struct {
	Key    string `json:"key"`
	Group  string `json:"group"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Status string `json:"status"`
	Core   bool   `json:"core"`
	Detail string `json:"detail"`
}

// Plan is the trade the rules imply if the user takes it today. Prices are
// on the exchange's tick grid.
type Plan struct {
	Entry float64 `json:"entry"`
	Stop  float64 `json:"stop"`
	// StopPct is the entry→stop distance in percent.
	StopPct float64 `json:"stopPct"`
	// RiskPct is the share of the account lost if the stop hits.
	RiskPct float64 `json:"riskPct"`
	// PositionPct is RiskPct/StopPct of the account, capped at MaxPositionPct.
	PositionPct float64 `json:"positionPct"`
	// Target is entry + RewardRisk × (entry − stop).
	Target     float64 `json:"target"`
	RewardRisk float64 `json:"rewardRisk"`
	// TrailEMA9 is today's 9 EMA: the trailing exit while the trend holds.
	TrailEMA9 float64 `json:"trailEma9"`
	EMA21     float64 `json:"ema21"`
}

// Result is GET /symbols/:symbol/luk-review.
type Result struct {
	Symbol  string  `json:"symbol"`
	Verdict string  `json:"verdict"`
	Passed  int     `json:"passed"`
	Total   int     `json:"total"`
	Checks  []Check `json:"checks"`
	Plan    *Plan   `json:"plan"`
	// Notes are the caveats that apply to every Vietnamese stock (T+2.5,
	// price bands) -- the rules were written for US stocks.
	Notes []string `json:"notes"`
	// AIAvailable tells the frontend whether POST .../luk-review/ai works
	// (GEMINI_API_KEY is set on the server).
	AIAvailable bool   `json:"aiAvailable"`
	Source      string `json:"source"`
	GeneratedAt string `json:"generatedAt"`
}

// AIReview is POST /symbols/:symbol/luk-review/ai.
type AIReview struct {
	Symbol      string `json:"symbol"`
	Verdict     string `json:"verdict"`
	Text        string `json:"text"`
	Model       string `json:"model"`
	GeneratedAt string `json:"generatedAt"`
}
