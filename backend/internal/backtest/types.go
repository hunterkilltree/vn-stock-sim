package backtest

type Rule struct {
	Type   string         `json:"type" binding:"required,oneof=ema_crossover rsi_reversion"`
	Params map[string]int `json:"params" binding:"required"`
}

type createRequest struct {
	Symbol          string  `json:"symbol" binding:"required"`
	From            string  `json:"from" binding:"required"`
	To              string  `json:"to" binding:"required"`
	StartingCapital float64 `json:"startingCapital" binding:"required,gt=0"`
	Rule            Rule    `json:"rule" binding:"required"`
}

type Backtest struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
	FinalCapital  float64 `json:"finalCapital,omitempty"`
	ReturnPercent float64 `json:"returnPercent,omitempty"`
	TotalTrades   int     `json:"totalTrades,omitempty"`
	WinRate       float64 `json:"winRate,omitempty"`
	RuleType      string  `json:"ruleType"`
	// Max peak-to-trough equity decline (percent) and gross profit /
	// gross loss over closed trades (0 when there were no losing trades).
	MaxDrawdownPercent float64 `json:"maxDrawdownPercent"`
	ProfitFactor       float64 `json:"profitFactor"`

	// Added for the /backtest results page (phase-k.md decision 13); only
	// the single-backtest endpoints carry Equity and Trades.
	From                   string         `json:"from"`
	To                     string         `json:"to"`
	StartingCapital        float64        `json:"startingCapital"`
	Params                 map[string]int `json:"params"`
	BenchmarkReturnPercent float64        `json:"benchmarkReturnPercent"`
	Equity                 []EquityPoint  `json:"equity,omitempty"`
	Trades                 []Trade        `json:"trades,omitempty"`

	// phase-valuation-backtest.md slice 1. Runs stored before it have
	// none of these (zero / null), and their benchmark ignores fees.
	FeesPaid      float64 `json:"feesPaid"`
	TimeInCashPct float64 `json:"timeInCashPercent"`
	// DaysToFirstFill is calendar days from the first bar to the first
	// buy; null when the rule never bought.
	DaysToFirstFill *int      `json:"daysToFirstFill"`
	LumpSum         *Baseline `json:"lumpSum,omitempty"`
	DCA             *Baseline `json:"dca,omitempty"`
}

// Baseline is a no-rule comparison run on the same bars with the same
// capital, fees and lots: lump sum on the first bar, or monthly DCA.
type Baseline struct {
	FinalCapital       float64 `json:"finalCapital"`
	ReturnPercent      float64 `json:"returnPercent"`
	MaxDrawdownPercent float64 `json:"maxDrawdownPercent"`
	FeesPaid           float64 `json:"feesPaid"`
}

// EquityPoint is the strategy's equity after a bar, next to lump sum
// (Benchmark: buy-and-hold from the first bar) and monthly DCA of the
// same starting capital.
type EquityPoint struct {
	Time      int64   `json:"time"`
	Equity    float64 `json:"equity"`
	Benchmark float64 `json:"benchmark"`
	DCA       float64 `json:"dca"`
}

// Trade is one round trip; Open means still held at the end of the range
// (exit = the last close, not counted in totalTrades or the win rate).
// ReturnPercent is after the fee on both sides.
type Trade struct {
	EntryTime     int64   `json:"entryTime"`
	EntryPrice    float64 `json:"entryPrice"`
	ExitTime      int64   `json:"exitTime"`
	ExitPrice     float64 `json:"exitPrice"`
	ReturnPercent float64 `json:"returnPercent"`
	Open          bool    `json:"open,omitempty"`
}
