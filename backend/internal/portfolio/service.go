package portfolio

import (
	"math"
	"sort"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type Service struct {
	store  Store
	quotes QuotePort
	// orders is set after order.Service exists (main.go calls
	// SetOrdersPort) -- order.Service depends on *Service for order
	// placement, so this reverse dependency can't be a constructor arg
	// without cycling. Stats/Positions degrade gracefully (empty trade
	// history, no OpenSince) if it's never set.
	orders OrdersPort
}

func NewService(store Store, quotes QuotePort) *Service {
	return &Service{store: store, quotes: quotes}
}

// SetOrdersPort wires the order-history dependency Stats/Positions need
// for FIFO trade reconstruction -- see phase-e.md item 2 and the field
// comment above.
func (s *Service) SetOrdersPort(orders OrdersPort) {
	s.orders = orders
}

// ApplyFill books a fill against portfolioID's ledger (delegating to the
// store) and, on success, records a real mark-to-market equity snapshot
// -- this is what main.go wires as order.Service's Ledger dependency
// (instead of the bare store), so every fill grows the equity-history
// series Stats' max-drawdown and the equity-curve chart both read (see
// phase-e.md item 1).
func (s *Service) ApplyFill(portfolioID, sym, side string, quantity, price, fee float64) error {
	if err := s.store.ApplyFill(portfolioID, sym, side, quantity, price, fee); err != nil {
		return err
	}
	s.store.AppendEquitySnapshot(portfolioID, s.Summary(portfolioID).TotalEquity)
	return nil
}

func (s *Service) EquityHistory(portfolioID string) []EquityPoint {
	return s.store.EquityHistory(portfolioID)
}

// DefaultPortfolioID returns userID's lazily-created default portfolio,
// so callers that don't yet pick a specific portfolio (pre-Phase-B
// frontend, order requests with no portfolioId) keep working -- see
// phase-b.md decision 1.
func (s *Service) DefaultPortfolioID(userID string) string {
	return s.store.DefaultFor(userID)
}

func (s *Service) CreatePortfolio(userID, name, market string, startingCapital float64, currency string) Portfolio {
	if name == "" {
		name = "Danh muc moi"
	}
	if market != "crypto" {
		market = "stock"
	}
	// Currency follows the market: VND for stocks, USDT for crypto
	// wallets (Crypto-Main's "Ví giấy ... 10.000 USDT ban đầu").
	currency = "VND"
	if market == "crypto" {
		currency = "USDT"
	}
	if startingCapital <= 0 {
		startingCapital = StartingCash
		if market == "crypto" {
			startingCapital = StartingUSDT
		}
	}
	return s.store.Create(userID, name, market, startingCapital, currency, KindTrading)
}

func (s *Service) CreateReplayPortfolio(userID, name, market string, startingCapital float64) Portfolio {
	currency := "VND"
	if market == "crypto" {
		currency = "USDT"
	} else {
		market = "stock"
	}
	return s.store.Create(userID, name, market, startingCapital, currency, KindReplay)
}

func (s *Service) ListPortfolios(userID string) []Portfolio {
	// DefaultFor ensures every user has at least one portfolio to list,
	// even if they've never explicitly created one.
	s.store.DefaultFor(userID)
	return s.store.List(userID)
}

func (s *Service) GetPortfolio(userID, id string) (Portfolio, bool) {
	return s.store.Get(userID, id)
}

func (s *Service) Summary(portfolioID string) Summary {
	cash := s.store.Cash(portfolioID)
	positions := s.valuedPositions(portfolioID)

	var marketValue, costBasis float64
	for _, p := range positions {
		marketValue += p.MarketValue
		costBasis += p.AvgCost * float64(p.Quantity)
	}
	unrealized := marketValue - costBasis
	var unrealizedPct float64
	if costBasis > 0 {
		unrealizedPct = unrealized / costBasis * 100
	}
	return Summary{
		CashBalance:          round2(cash),
		MarketValue:          round2(marketValue),
		TotalEquity:          round2(cash + marketValue),
		UnrealizedPnl:        round2(unrealized),
		UnrealizedPnlPercent: round2(unrealizedPct),
	}
}

func (s *Service) Positions(portfolioID string) []Position {
	return s.valuedPositions(portfolioID)
}

func (s *Service) valuedPositions(portfolioID string) []Position {
	raw := s.store.Positions(portfolioID)
	_, openLots := s.reconstructTrades(portfolioID)
	out := make([]Position, 0, len(raw))
	for sym, p := range raw {
		lastPrice := p.avgCost
		if d, ok := s.quotes.Detail(sym); ok {
			lastPrice = d.LastPrice
		}
		marketValue := lastPrice * float64(p.quantity)
		pos := Position{
			Symbol:        sym,
			Quantity:      p.quantity,
			AvgCost:       roundPrice(p.avgCost),
			LastPrice:     roundPrice(lastPrice),
			MarketValue:   round2(marketValue),
			UnrealizedPnl: round2(marketValue - p.avgCost*float64(p.quantity)),
		}
		if lots := openLots[sym]; len(lots) > 0 {
			pos.OpenSince = lots[0].at.Format("2006-01-02T15:04:05Z")
		}
		out = append(out, pos)
	}
	return out
}

// Allocation groups a portfolio's current NAV by sector (via QuotePort's
// existing Detail, which already carries symbol.Sector -- see phase-e.md
// item 3) plus a trailing "Tiền mặt" (cash) bucket. No new storage.
func (s *Service) Allocation(portfolioID string) []Allocation {
	positions := s.valuedPositions(portfolioID)
	cash := s.store.Cash(portfolioID)

	bySector := make(map[string]float64)
	var order []string
	for _, p := range positions {
		if d, ok := s.quotes.Detail(p.Symbol); ok {
			if _, seen := bySector[d.Sector]; !seen {
				order = append(order, d.Sector)
			}
			bySector[d.Sector] += p.MarketValue
		}
	}

	var total float64
	for _, v := range bySector {
		total += v
	}
	total += cash

	out := make([]Allocation, 0, len(order)+1)
	for _, sector := range order {
		v := bySector[sector]
		pct := 0.0
		if total > 0 {
			pct = v / total * 100
		}
		out = append(out, Allocation{Sector: sector, Value: round2(v), Percent: round2(pct)})
	}
	cashPct := 0.0
	if total > 0 {
		cashPct = cash / total * 100
	}
	out = append(out, Allocation{Sector: "Tiền mặt", Value: round2(cash), Percent: round2(cashPct)})
	return out
}

type lot struct {
	quantity float64
	price    float64
	at       time.Time
}

// reconstructTrades replays portfolioID's filled orders (oldest first,
// as OrdersPort.FilledOrders naturally returns them -- see order.go's
// Append-ordered store) using FIFO lot matching to produce every
// realized closed trade plus each symbol's still-open lots -- see
// phase-e.md item 2. Returns (nil, nil) if OrdersPort was never wired
// (SetOrdersPort not called), so callers degrade gracefully instead of
// panicking.
func (s *Service) reconstructTrades(portfolioID string) ([]ClosedTrade, map[string][]lot) {
	if s.orders == nil {
		return nil, nil
	}
	userID, ok := s.store.OwnerOf(portfolioID)
	if !ok {
		return nil, nil
	}
	fills := s.orders.FilledOrders(userID, portfolioID)
	// FIFO matching below assumes fills arrive oldest-first. That's true
	// by construction for ordinary real-time trading (orders are
	// appended in the order they happen), but Phase F's Replay sessions
	// append fills carrying a *simulated* historical FilledAt at real
	// (later) wall-clock append time -- so append order and chronological
	// order can disagree once Replay exists. Sorting defensively here
	// costs nothing for the already-sorted real-trading case.
	sort.SliceStable(fills, func(i, j int) bool { return fills[i].FilledAt < fills[j].FilledAt })

	queues := make(map[string][]lot)
	var closed []ClosedTrade
	for _, f := range fills {
		at, err := time.Parse("2006-01-02T15:04:05Z", f.FilledAt)
		if err != nil {
			continue
		}
		switch f.Side {
		case "buy":
			queues[f.Symbol] = append(queues[f.Symbol], lot{quantity: f.Quantity, price: f.FilledPrice, at: at})
		case "sell":
			remaining := f.Quantity
			q := queues[f.Symbol]
			for remaining > qtyEpsilon && len(q) > 0 {
				head := &q[0]
				matched := head.quantity
				if matched > remaining {
					matched = remaining
				}
				pnlPercent := 0.0
				if head.price > 0 {
					pnlPercent = (f.FilledPrice - head.price) / head.price * 100
				}
				closed = append(closed, ClosedTrade{
					Symbol:      f.Symbol,
					Quantity:    matched,
					EntryPrice:  head.price,
					ExitPrice:   f.FilledPrice,
					EntryAt:     head.at.Format("2006-01-02T15:04:05Z"),
					ExitAt:      f.FilledAt,
					HoldingDays: round2(at.Sub(head.at).Hours() / 24),
					PnlAmount:   round2((f.FilledPrice - head.price) * float64(matched)),
					PnlPercent:  round2(pnlPercent),
				})
				head.quantity -= matched
				remaining -= matched
				if head.quantity <= qtyEpsilon {
					q = q[1:]
				}
			}
			queues[f.Symbol] = q
		}
	}
	return closed, queues
}

// Stats computes Portfolio.dc.html's KPI row + journal grid from real
// closed trades (via reconstructTrades) and the real equity-history
// series (via EquityHistory) -- not hardcoded sample numbers. See
// phase-e.md item 2.
func (s *Service) Stats(portfolioID string, startingCapital float64) Stats {
	summary := s.Summary(portfolioID)
	stats := Stats{
		TotalEquity: summary.TotalEquity,
	}
	if startingCapital > 0 {
		stats.TotalPnl = round2(summary.TotalEquity - startingCapital)
		stats.TotalPnlPercent = round2(stats.TotalPnl / startingCapital * 100)
	}

	closed, _ := s.reconstructTrades(portfolioID)
	stats.ClosedTradeCount = len(closed)

	var sumWinAmount, sumLossAmount, sumWinPercent, sumLossPercent, sumHoldingDays float64
	var best, worst *ClosedTrade
	for i := range closed {
		t := &closed[i]
		sumHoldingDays += t.HoldingDays
		if t.PnlAmount >= 0 {
			stats.Wins++
			sumWinAmount += t.PnlAmount
			sumWinPercent += t.PnlPercent
		} else {
			stats.Losses++
			sumLossAmount += -t.PnlAmount
			sumLossPercent += t.PnlPercent
		}
		if best == nil || t.PnlPercent > best.PnlPercent {
			best = t
		}
		if worst == nil || t.PnlPercent < worst.PnlPercent {
			worst = t
		}
	}
	if stats.ClosedTradeCount > 0 {
		stats.WinRate = round2(float64(stats.Wins) / float64(stats.ClosedTradeCount) * 100)
		stats.AvgHoldingDays = round2(sumHoldingDays / float64(stats.ClosedTradeCount))
	}
	if stats.Wins > 0 {
		stats.AvgWinPercent = round2(sumWinPercent / float64(stats.Wins))
	}
	if stats.Losses > 0 {
		stats.AvgLossPercent = round2(sumLossPercent / float64(stats.Losses))
	}
	if sumLossAmount > 0 {
		stats.ProfitFactor = round2(sumWinAmount / sumLossAmount)
	}
	if best != nil {
		stats.BestTrade = &ClosedTradeSummary{Symbol: best.Symbol, PnlPercent: best.PnlPercent, PnlAmount: best.PnlAmount}
	}
	if worst != nil {
		stats.WorstTrade = &ClosedTradeSummary{Symbol: worst.Symbol, PnlPercent: worst.PnlPercent, PnlAmount: worst.PnlAmount}
	}

	stats.MaxDrawdownPercent = maxDrawdown(s.store.EquityHistory(portfolioID))
	return stats
}

// maxDrawdown returns the largest peak-to-trough decline (as a positive
// percentage) across an equity-history series -- phase-e.md item 2's
// formula, section 2.5's same heuristic style as Replay's planned skill
// score.
func maxDrawdown(points []EquityPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	sorted := make([]EquityPoint, len(points))
	copy(sorted, points)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Timestamp < sorted[j].Timestamp })

	peak := sorted[0].NAV
	var maxDD float64
	for _, p := range sorted {
		if p.NAV > peak {
			peak = p.NAV
		}
		if peak > 0 {
			dd := (peak - p.NAV) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return round2(maxDD)
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}

// roundPrice keeps 2 decimals for normal prices but 8 below 1, so
// sub-cent coins (SLP at 0.00412 USDT) don't round to zero.
func roundPrice(v float64) float64 {
	if v >= 1 || v <= -1 {
		return round2(v)
	}
	return math.Round(v*1e8) / 1e8
}
