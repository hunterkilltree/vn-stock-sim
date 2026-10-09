package rating

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/valuation"

// Verdict is one rule's vote, and Rating the five-step scale the votes
// roll up into -- the same five steps (Strong Sell .. Strong Buy) as
// TradingView's Technical Ratings and most broker research, so the badge
// reads the way a user already expects (phase-rating.md decision 1).
const (
	VerdictBuy     = "buy"
	VerdictNeutral = "neutral"
	VerdictSell    = "sell"

	RatingStrongBuy  = "strong_buy"
	RatingBuy        = "buy"
	RatingNeutral    = "neutral"
	RatingSell       = "sell"
	RatingStrongSell = "strong_sell"

	GroupTechnical   = "technical"
	GroupFundamental = "fundamental"
)

// Signal is one rule's input value and vote. Value is preformatted so the
// frontend doesn't need to know each rule's units.
type Signal struct {
	Group   string `json:"group"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Verdict string `json:"verdict"`
	Detail  string `json:"detail"`
}

// Summary is one group's (or the overall) roll-up. Score is the mean vote
// in [-1, 1]; Buy/Neutral/Sell count the votes behind it.
type Summary struct {
	Rating  string  `json:"rating"`
	Score   float64 `json:"score"`
	Buy     int     `json:"buy"`
	Neutral int     `json:"neutral"`
	Sell    int     `json:"sell"`
}

// Result is GET /symbols/:symbol/rating. Fundamental is nil when the
// symbol has no fundamentals (e.g. crypto), and Overall is then the
// technical summary alone.
type Result struct {
	Symbol      string   `json:"symbol"`
	Overall     Summary  `json:"overall"`
	Technical   Summary  `json:"technical"`
	Fundamental *Summary `json:"fundamental"`
	Signals     []Signal `json:"signals"`
	// Horizons and Valuation are nil when there isn't enough data
	// (phase-holding-horizon.md, phase-valuation.md); crypto never has them.
	Horizons    *Horizons          `json:"horizons"`
	Valuation   *valuation.Summary `json:"valuation"`
	Source      string             `json:"source"`
	GeneratedAt string             `json:"generatedAt"`
}
