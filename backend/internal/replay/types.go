// Package replay backs Replay Mode (design/screens/Replay.dc.html):
// candle-by-candle historical replay with the future genuinely withheld
// server-side (not just visually masked) and a heuristic skill score
// computed once a session ends. See phase-f.md for the full design.
package replay

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/market"

// DefaultAnchor is the historical date a new session starts from when
// the request doesn't specify one -- matches
// design/screens/Replay.dc.html's own caption ("Phiên mô phỏng bắt đầu
// 04/01/2021") exactly, so a visual check against the mockup is a fair
// comparison. "2021-01-04" parses with time.Parse("2006-01-02", ...).
const DefaultAnchor = "2021-01-04"

// DefaultTotalBars matches the design's own "Nến 44/120" caption.
const DefaultTotalBars = 120

// DefaultStartingCapital matches portfolio.StartingCash -- not imported
// directly to avoid a needless import just for one constant; kept equal
// deliberately (see service.go).
const DefaultStartingCapital = 100_000_000

// Crypto sessions (phase-i.md decision 11): Crypto-Replay.dc.html's May
// 2021 crash on 1-hour candles, with a 10,000 USDT replay wallet.
const DefaultCryptoAnchor = "2021-05-01"
const DefaultCryptoStartingCapital = 10_000

// InitialRevealed is how many candles a fresh session shows immediately
// (not zero -- an empty chart on session start would be a strange first
// screen, and SMA(20) needs at least 20 points to plot at all).
const InitialRevealed = 20

// Session is one Replay run: a single symbol, a fixed historical bar
// range fetched once at Start (see service.go) and progressively
// revealed via Advance, backed by its own dedicated portfolio (see
// phase-f.md decision 4 for why it isn't the user's regular one).
type Session struct {
	ID          string
	UserID      string
	PortfolioID string
	Symbol      string
	Market      string  // "stock" | "crypto"
	Capital     float64 // starting capital of the session's portfolio
	Resolution  string
	// Bars is the FULL historical series, fetched once at Start -- only
	// Bars[:CurrentBar] is ever exposed in a SessionView. Scoring at End
	// is the one place allowed to look past CurrentBar, since by then the
	// session is over and nothing more can be traded on that knowledge.
	Bars       []market.Bar
	TotalBars  int
	CurrentBar int
	Status     string // "active" | "completed"
	Fills      []Fill
	StopLoss   float64 // 0 = no active stop for Symbol
	StartedAt  string
}

// Fill is one Replay order, always executed at Bars[BarIndex].Close --
// see phase-f.md decision 3 on why there's no client-suppliable price at
// all, not just a validated one.
type Fill struct {
	BarIndex int     `json:"barIndex"`
	Date     string  `json:"date"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	// StopSet is the stop-loss price active immediately after this fill
	// (0 = none), recorded on the fill itself rather than only on the
	// session's mutable current StopLoss, so the skill score can later
	// tell "a stop was set at the time of this buy" from "no stop was
	// ever set" -- see score.go's stopDiscipline.
	StopSet float64 `json:"stopSet,omitempty"`
	Note    string  `json:"note,omitempty"`
}

type StartRequest struct {
	Symbol     string `json:"symbol" binding:"required"`
	Market     string `json:"market" binding:"omitempty,oneof=stock crypto"`
	Resolution string `json:"resolution"`
	TotalBars  int    `json:"totalBars"`
	StartDate  string `json:"startDate"`
}

type OrderRequest struct {
	Side     string  `json:"side" binding:"required,oneof=buy sell"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	StopLoss float64 `json:"stopLoss"`
}

// SessionView is the one response shape Start/Advance/PlaceOrder/Get all
// return -- the client always gets the full current state back rather
// than a bespoke diff per endpoint, since the UI (chart markers,
// position line, fill log) needs to redraw all of it after any action.
type SessionView struct {
	ID          string                  `json:"id"`
	Symbol      string                  `json:"symbol"`
	Market      string                  `json:"market"`
	Capital     float64                 `json:"startingCapital"`
	Resolution  string                  `json:"resolution"`
	PortfolioID string                  `json:"portfolioId"`
	CurrentBar  int                     `json:"currentBar"`
	TotalBars   int                     `json:"totalBars"`
	Done        bool                    `json:"done"`
	Status      string                  `json:"status"`
	Bars        []market.Bar            `json:"bars"`
	SMA20       []market.IndicatorPoint `json:"sma20"`
	Fills       []Fill                  `json:"fills"`
	StopLoss    float64                 `json:"stopLoss,omitempty"`
	PositionQty float64                 `json:"positionQty"`
	AvgCost     float64                 `json:"avgCost,omitempty"`
	// Result is always present, not just after End -- KPIs and the skill
	// score are real numbers throughout the session (matching the
	// design's "Điểm kỹ năng tạm tính" = "provisional skill score" panel,
	// visible mid-session), Skill.Final just flips true once the session
	// is actually completed.
	Result *EndResult `json:"result"`
}

// SkillScore is explicitly a heuristic (Source is always "heuristic"),
// never presented as a validated skill assessment -- see phase-f.md's
// formula writeup in score.go.
type SkillScore struct {
	Source         string  `json:"source"`
	Final          bool    `json:"final"`
	Overall        float64 `json:"overall"`
	EntryQuality   float64 `json:"entryQuality"`
	ExitQuality    float64 `json:"exitQuality"`
	StopDiscipline float64 `json:"stopDiscipline"`
	PositionSizing float64 `json:"positionSizing"`
}

type EndResult struct {
	NAV                float64    `json:"nav"`
	PnlPercent         float64    `json:"pnlPercent"`
	TotalTrades        int        `json:"totalTrades"`
	Wins               int        `json:"wins"`
	Losses             int        `json:"losses"`
	MaxDrawdownPercent float64    `json:"maxDrawdownPercent"`
	ProfitFactor       float64    `json:"profitFactor"`
	Skill              SkillScore `json:"skill"`
}
