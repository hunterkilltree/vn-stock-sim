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
}
