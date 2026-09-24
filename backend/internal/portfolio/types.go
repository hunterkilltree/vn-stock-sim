package portfolio

// Portfolio is one named paper-trading account a user owns. A user can
// hold several (Account-Menu.dc.html shows switching between a main
// stock portfolio, a strategy-test portfolio, and a crypto wallet) --
// see phase-b.md decision 1. Currency is "VND" for market: "stock" and
// "USDT" for market: "crypto" (crypto portfolios land in Phase I).
type Portfolio struct {
	ID              string  `json:"id"`
	UserID          string  `json:"userId"`
	Name            string  `json:"name"`
	Market          string  `json:"market"`
	Kind            string  `json:"kind"`
	StartingCapital float64 `json:"startingCapital"`
	Currency        string  `json:"currency"`
	CreatedAt       string  `json:"createdAt"`
}

// A replay portfolio holds one Replay session's historical-price fills
// and must never receive live orders -- see phase-g.md decision 4.
const (
	KindTrading = "trading"
	KindReplay  = "replay"
)

type Position struct {
	Symbol        string  `json:"symbol"`
	Quantity      float64 `json:"quantity"`
	AvgCost       float64 `json:"avgCost"`
	LastPrice     float64 `json:"lastPrice"`
	MarketValue   float64 `json:"marketValue"`
	UnrealizedPnl float64 `json:"unrealizedPnl"`
	// OpenSince is the fill time of the oldest buy lot still open for
	// this symbol (FIFO -- see service.go's reconstructTrades), empty if
	// unknown (no OrdersPort wired, or no matching fill found). Backs
	// Portfolio.dc.html's "Nắm giữ" (days held) column.
	OpenSince string `json:"openSince,omitempty"`
}

type Summary struct {
	CashBalance          float64 `json:"cashBalance"`
	MarketValue          float64 `json:"marketValue"`
	TotalEquity          float64 `json:"totalEquity"`
	UnrealizedPnl        float64 `json:"unrealizedPnl"`
	UnrealizedPnlPercent float64 `json:"unrealizedPnlPercent"`
}

// StartingCash is the virtual capital every new paper-trading account
// opens with (vn-stock-sim-version-highlights.md: "Starting cash, current
// holdings, market value, and profit/loss — always visible").
const StartingCash = 100_000_000

// StartingUSDT is a new crypto wallet's default balance (Crypto-Main.dc.html).
const StartingUSDT = 10_000

// EquityPoint is one (timestamp, NAV) sample of a portfolio's total
// equity over time -- see phase-e.md item 1. V1 appends a real point on
// every fill (not a scheduled snapshot job, see store.go) plus one seed
// point at creation, so a brand-new portfolio's history is never empty.
type EquityPoint struct {
	Timestamp string  `json:"timestamp"`
	NAV       float64 `json:"nav"`
}

// Allocation is one bucket (a sector, or the "Cash" bucket) of a
// portfolio's current NAV -- phase-e.md item 3, derived purely from
// existing positions joined against symbol.Sector, no new storage.
type Allocation struct {
	Sector  string  `json:"sector"`
	Value   float64 `json:"value"`
	Percent float64 `json:"percent"`
}

// OrderRecord is the minimal shape Service needs from a filled order to
// reconstruct realized trades -- satisfied by *order.Service via the
// OrdersPort interface below, converted from order.Order (order already
// depends on this package for its Ledger error types, so this direction
// adds no import cycle).
type OrderRecord struct {
	Symbol      string
	Side        string // "buy" or "sell"
	Quantity    float64
	FilledPrice float64
	FilledAt    string // formatted like order.formatTime: "2006-01-02T15:04:05Z"
}

// OrdersPort supplies a portfolio's filled order history for Stats' FIFO
// trade reconstruction (phase-e.md item 2). Set via Service.SetOrdersPort
// after both services exist (order.Service depends on portfolio.Service
// for order placement, so wiring the reverse dependency at construction
// time would cycle -- see main.go).
type OrdersPort interface {
	FilledOrders(userID, portfolioID string) []OrderRecord
}

// ClosedTrade is one realized round-trip (a sell fill matched against an
// earlier buy fill via FIFO lot matching -- see service.go's
// reconstructTrades). A single sell can produce several ClosedTrades if
// it drains more than one buy lot.
type ClosedTrade struct {
	Symbol      string
	Quantity    float64
	EntryPrice  float64
	ExitPrice   float64
	EntryAt     string
	ExitAt      string
	HoldingDays float64
	PnlAmount   float64
	PnlPercent  float64
}

// ClosedTradeSummary is the trimmed view of a ClosedTrade exposed in
// Stats (BestTrade/WorstTrade) -- the API doesn't need the full lot
// detail, just enough to label it.
type ClosedTradeSummary struct {
	Symbol     string  `json:"symbol"`
	PnlPercent float64 `json:"pnlPercent"`
	PnlAmount  float64 `json:"pnlAmount"`
}

// Stats is a portfolio's real performance KPIs (Portfolio.dc.html's KPI
// row + journal grid), computed from actual closed trades and the real
// equity-history series -- not hardcoded sample numbers. Win rate/profit
// factor/max drawdown are the same KPI classes vn-stock-sim-summary.md's
// backtest section names; here they're computed over real paper-trading
// order history instead of a simulated backtest run (phase-e.md item 2).
type Stats struct {
	TotalEquity float64 `json:"totalEquity"`
	// TotalPnl/TotalPnlPercent are against the portfolio's starting
	// capital, at the current instant (includes unrealized P&L on open
	// positions, not just closed trades).
	TotalPnl        float64 `json:"totalPnl"`
	TotalPnlPercent float64 `json:"totalPnlPercent"`

	// The rest are computed purely from closed (realized) trades.
	ClosedTradeCount int     `json:"closedTradeCount"`
	Wins             int     `json:"wins"`
	Losses           int     `json:"losses"`
	WinRate          float64 `json:"winRate"`
	// ProfitFactor is sum(winning PnL) / abs(sum(losing PnL)). 0 when
	// there are no closed trades yet, or no losing trades to divide by
	// (an honestly-undefined ratio, not a fabricated large number).
	ProfitFactor       float64             `json:"profitFactor"`
	AvgWinPercent      float64             `json:"avgWinPercent"`
	AvgLossPercent     float64             `json:"avgLossPercent"`
	AvgHoldingDays     float64             `json:"avgHoldingDays"`
	MaxDrawdownPercent float64             `json:"maxDrawdownPercent"`
	BestTrade          *ClosedTradeSummary `json:"bestTrade,omitempty"`
	WorstTrade         *ClosedTradeSummary `json:"worstTrade,omitempty"`
}
