package stress

// Scenario kinds: "historical" replays a real VN-Index fall found in the
// data with each holding's real path; "hypothetical" applies an index
// shock through each holding's beta (phase-market-risk.md feature 4).
const (
	KindHistorical   = "historical"
	KindHypothetical = "hypothetical"
)

type HoldingBeta struct {
	Symbol  string  `json:"symbol"`
	Value   float64 `json:"value"`
	Beta    float64 `json:"beta"`
	Samples int     `json:"samples"` // daily returns behind the beta; 0 = defaulted to 1
	// CrisisBeta is what the shocks use: max(Beta, 1).
	CrisisBeta float64 `json:"crisisBeta"`
}

// HoldingImpact is one holding under one scenario. Estimated means the
// stock has no price history for that period, so beta × index move was
// used instead of its real path.
type HoldingImpact struct {
	Symbol        string  `json:"symbol"`
	ChangePct     float64 `json:"changePct"`
	LossVND       float64 `json:"lossVnd"`
	Estimated     bool    `json:"estimated"`
	FloorDays     int     `json:"floorDays"`     // sessions closing at (about) the floor price
	LongestFloors int     `json:"longestFloors"` // longest run of consecutive floor sessions
}

type Scenario struct {
	Key            string          `json:"key"`
	Label          string          `json:"label"`
	Kind           string          `json:"kind"`
	From           int64           `json:"from,omitempty"`
	To             int64           `json:"to,omitempty"`
	IndexChangePct float64         `json:"indexChangePct"`
	LossVND        float64         `json:"lossVnd"`
	LossPct        float64         `json:"lossPct"` // of total equity (cash included)
	Holdings       []HoldingImpact `json:"holdings"`
	Note           string          `json:"note,omitempty"`
}

type Result struct {
	Equity     float64       `json:"equity"`
	StockValue float64       `json:"stockValue"`
	Cash       float64       `json:"cash"`
	Betas      []HoldingBeta `json:"betas"`
	Scenarios  []Scenario    `json:"scenarios"`
	Source     string        `json:"source"`
}
