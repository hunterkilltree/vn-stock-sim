package replay

import (
	"errors"
	"fmt"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/order"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
)

var ErrNotFound = errors.New("replay session not found")
var ErrNoData = errors.New("no historical data for that symbol/range")
var ErrSessionDone = errors.New("replay session already completed")

// BarsPort is satisfied by *market.Service unchanged.
type BarsPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

// PortfolioPort is satisfied by *portfolio.Service. See phase-f.md
// decision 4 on why Replay gets its own dedicated portfolio per session
// instead of reusing the user's regular one, and phase-g.md decision 4 on
// why it is created as a "replay"-kind portfolio live orders can't reach.
type PortfolioPort interface {
	CreateReplayPortfolio(userID, name string, startingCapital float64) portfolio.Portfolio
	ApplyFill(portfolioID, sym, side string, quantity int64, price float64) error
	Positions(portfolioID string) []portfolio.Position
	Stats(portfolioID string, startingCapital float64) portfolio.Stats
}

// OrderLog is satisfied by *order.MemoryStore's existing public Append
// -- no new method needed on the order package. Replay fills are
// appended as real order.Order records (with the *simulated* historical
// date, not wall-clock time -- see phase-f.md decision 5) so the
// session's dedicated portfolio's own Stats/journal sees them exactly
// like any other paper trade.
type OrderLog interface {
	Append(userID string, o order.Order) order.Order
}

type Service struct {
	store      *MemoryStore
	bars       BarsPort
	portfolios PortfolioPort
	orders     OrderLog
}

func NewService(store *MemoryStore, bars BarsPort, portfolios PortfolioPort, orders OrderLog) *Service {
	return &Service{store: store, bars: bars, portfolios: portfolios, orders: orders}
}

func (s *Service) Start(userID string, req StartRequest) (SessionView, error) {
	resolution := req.Resolution
	if resolution == "" {
		resolution = "1D"
	}
	totalBars := req.TotalBars
	if totalBars <= 0 {
		totalBars = DefaultTotalBars
	}
	startDate := req.StartDate
	if startDate == "" {
		startDate = DefaultAnchor
	}
	anchor, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return SessionView{}, fmt.Errorf("invalid startDate %q: %w", startDate, err)
	}

	const daySeconds = 24 * 60 * 60
	from := anchor.Unix()
	to := from + int64(totalBars-1)*daySeconds

	bars := s.bars.GetBars(req.Symbol, resolution, from, to)
	if len(bars) == 0 {
		return SessionView{}, ErrNoData
	}
	// GetBars' own from/to grid-alignment can return a slightly
	// different count than requested -- trust what actually came back
	// rather than the request, so CurrentBar/TotalBars never disagree
	// with len(Bars).
	totalBars = len(bars)

	pf := s.portfolios.CreateReplayPortfolio(
		userID,
		fmt.Sprintf("Replay %s %s", req.Symbol, startDate),
		DefaultStartingCapital,
	)

	revealed := InitialRevealed
	if revealed > totalBars {
		revealed = totalBars
	}

	sess := &Session{
		ID:          s.store.nextSessionID(),
		UserID:      userID,
		PortfolioID: pf.ID,
		Symbol:      req.Symbol,
		Resolution:  resolution,
		Bars:        bars,
		TotalBars:   totalBars,
		CurrentBar:  revealed,
		Status:      "active",
		StartedAt:   startDate,
		// Fills starts as an empty (non-nil) slice, not nil -- a nil Go
		// slice marshals to JSON `null`, which crashed the frontend's
		// session.fills.length check on a session with no fills yet.
		Fills: []Fill{},
	}
	s.store.Append(userID, sess)
	return s.view(sess), nil
}

// Advance reveals the next candle and, if it breaches an active stop-
// loss, auto-closes the position at the stop price -- see phase-f.md
// decision 6. Calling Advance past the last bar is a safe no-op (Done
// stays true) rather than an error, so a client can poll it freely.
func (s *Service) Advance(userID, id string) (SessionView, error) {
	sess, ok := s.store.Get(userID, id)
	if !ok {
		return SessionView{}, ErrNotFound
	}
	if sess.Status != "active" {
		return SessionView{}, ErrSessionDone
	}
	if sess.CurrentBar < sess.TotalBars {
		sess.CurrentBar++
	}

	if sess.StopLoss > 0 && sess.CurrentBar > 0 {
		bar := sess.Bars[sess.CurrentBar-1]
		if bar.Low <= sess.StopLoss {
			s.fill(sess, "sell", 0, sess.StopLoss, "Tự động cắt lỗ", 0)
		}
	}

	return s.view(sess), nil
}

// PlaceOrder fills at Bars[CurrentBar-1].Close -- see phase-f.md decision
// 3 on why the request has no price field to validate at all.
func (s *Service) PlaceOrder(userID, id string, req OrderRequest) (SessionView, error) {
	sess, ok := s.store.Get(userID, id)
	if !ok {
		return SessionView{}, ErrNotFound
	}
	if sess.Status != "active" {
		return SessionView{}, ErrSessionDone
	}
	if sess.CurrentBar == 0 {
		return SessionView{}, errors.New("no candle revealed yet")
	}
	price := sess.Bars[sess.CurrentBar-1].Close

	stopToRecord := 0.0
	if req.Side == "buy" {
		stopToRecord = req.StopLoss
	}
	if err := s.fill(sess, req.Side, req.Quantity, price, "", stopToRecord); err != nil {
		return SessionView{}, err
	}
	if req.Side == "buy" && req.StopLoss > 0 {
		sess.StopLoss = req.StopLoss
	}
	return s.view(sess), nil
}

// fill books quantity shares of sess.Symbol against the session's
// dedicated portfolio at price, and logs both a Fill (for the session
// log/skill score) and a real order.Order (for the portfolio's own
// Stats/journal -- phase-f.md decision 5). quantity == 0 means "close
// the whole position" (used by the stop-loss auto-exit). stopSet is
// recorded on the Fill only (see types.go's Fill.StopSet); it does not
// itself set sess.StopLoss -- callers do that themselves once the fill
// succeeds.
func (s *Service) fill(sess *Session, side string, quantity int64, price float64, note string, stopSet float64) error {
	if quantity == 0 {
		for _, p := range s.portfolios.Positions(sess.PortfolioID) {
			if p.Symbol == sess.Symbol {
				quantity = p.Quantity
			}
		}
		if quantity == 0 {
			return nil // nothing to close
		}
	}

	if err := s.portfolios.ApplyFill(sess.PortfolioID, sess.Symbol, side, quantity, price); err != nil {
		return err
	}

	barDate := time.Unix(sess.Bars[sess.CurrentBar-1].Time, 0).UTC().Format("2006-01-02T15:04:05Z")
	fee := round2(price * float64(quantity) * order.FeeRate)
	s.orders.Append(sess.UserID, order.Order{
		PortfolioID: sess.PortfolioID,
		Symbol:      sess.Symbol,
		Side:        side,
		Type:        "market",
		Quantity:    quantity,
		Status:      "filled",
		FilledPrice: price,
		Fee:         fee,
		FilledAt:    barDate,
		CreatedAt:   barDate,
	})

	if side == "sell" {
		sess.StopLoss = 0
	}
	sess.Fills = append(sess.Fills, Fill{
		BarIndex: sess.CurrentBar - 1,
		Date:     barDate,
		Side:     side,
		Quantity: quantity,
		Price:    round2(price),
		StopSet:  stopSet,
		Note:     note,
	})
	return nil
}

// End marks the session completed -- callable at any point (an early
// exit is scored over however many bars were actually played, not just
// at TotalBars). The returned KPIs/skill score are the same shape
// view() always includes, just with Skill.Final now true.
func (s *Service) End(userID, id string) (SessionView, error) {
	sess, ok := s.store.Get(userID, id)
	if !ok {
		return SessionView{}, ErrNotFound
	}
	sess.Status = "completed"
	return s.view(sess), nil
}

func (s *Service) Get(userID, id string) (SessionView, error) {
	sess, ok := s.store.Get(userID, id)
	if !ok {
		return SessionView{}, ErrNotFound
	}
	return s.view(sess), nil
}

// view builds the client-facing snapshot -- Bars/SMA20 are sliced to
// [:CurrentBar] only. This is the actual "future hidden" enforcement
// point: nothing past CurrentBar is ever serialized, regardless of what
// the client asks for.
func (s *Service) view(sess *Session) SessionView {
	revealed := sess.Bars[:sess.CurrentBar]
	closes := make([]float64, len(revealed))
	for i, b := range revealed {
		closes[i] = b.Close
	}
	sma := sma20(closes, revealed)

	var qty int64
	var avgCost float64
	for _, p := range s.portfolios.Positions(sess.PortfolioID) {
		if p.Symbol == sess.Symbol {
			qty = p.Quantity
			avgCost = p.AvgCost
		}
	}

	// Wins/Losses/ProfitFactor/TotalTrades come from portfolio.Stats,
	// which is safe to reuse here -- they're derived purely from FIFO-
	// matched closed trades using each fill's own historical price, never
	// a live quote. NAV/PnlPercent/MaxDrawdownPercent are NOT reused from
	// there: Stats' mark-to-market for an open position values it at
	// today's real, live market quote (via portfolio.Service's QuotePort)
	// -- exactly correct for real paper trading, but wrong here, where it
	// would leak today's live price into a historical session that might
	// be "replaying" a completely different year. replayAccounting below
	// values the open position at the replay's own current revealed
	// close instead, the same way backtest/rule.go's own equity
	// calculation does.
	stats := s.portfolios.Stats(sess.PortfolioID, DefaultStartingCapital)
	nav, pnlPct, maxDD := replayAccounting(sess)
	result := &EndResult{
		NAV:                nav,
		PnlPercent:         pnlPct,
		TotalTrades:        stats.ClosedTradeCount,
		Wins:               stats.Wins,
		Losses:             stats.Losses,
		MaxDrawdownPercent: maxDD,
		ProfitFactor:       stats.ProfitFactor,
		Skill:              computeSkillScore(sess),
	}

	return SessionView{
		ID:          sess.ID,
		Symbol:      sess.Symbol,
		Resolution:  sess.Resolution,
		PortfolioID: sess.PortfolioID,
		CurrentBar:  sess.CurrentBar,
		TotalBars:   sess.TotalBars,
		Done:        sess.CurrentBar >= sess.TotalBars,
		Status:      sess.Status,
		Bars:        revealed,
		SMA20:       sma,
		Fills:       sess.Fills,
		StopLoss:    sess.StopLoss,
		PositionQty: qty,
		AvgCost:     round2(avgCost),
		Result:      result,
	}
}

// sma20 computes a plain 20-period SMA over the revealed closes only --
// deliberately not looking past sess.CurrentBar, even though the fuller
// series is sitting right there in sess.Bars, so the overlay itself
// never leaks unrevealed information.
func sma20(closes []float64, bars []market.Bar) []market.IndicatorPoint {
	const period = 20
	// Non-nil from the start -- see the matching Fills comment in Start:
	// a nil slice marshals to JSON `null`, not `[]`.
	out := []market.IndicatorPoint{}
	if len(closes) < period {
		return out
	}
	var sum float64
	for i, c := range closes {
		sum += c
		if i >= period {
			sum -= closes[i-period]
		}
		if i >= period-1 {
			out = append(out, market.IndicatorPoint{Time: bars[i].Time, Value: round2(sum / period)})
		}
	}
	return out
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}
