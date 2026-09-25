package order

import (
	"context"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// BarsPort gives the matcher 5-minute bars for a stock ticker or a
// crypto pair -- the same bars the Detail chart's "1 ngày"/"5 phút" view
// draws, so every fill can be pointed to on the chart (phase-k.md
// decision 4). Satisfied by crypto.BarsRouter.
type BarsPort interface {
	IntradayBars(sym string, from, to int64) []market.Bar
}

const matchStep = 5 * 60

// vnZone is Asia/Ho_Chi_Minh (UTC+7, no DST) -- fixed so the ATC close
// doesn't depend on the host's tzdata.
var vnZone = time.FixedZone("ICT", 7*60*60)

// EnableMatching turns on the matcher for queued orders. Without it
// (tests of Create alone, or no bar source) orders stay queued.
func (s *Service) EnableMatching(bars BarsPort) {
	s.bars = bars
	s.checked = make(map[string]int64)
}

// RunMatcher calls MatchPending every `every` until ctx is done
// (phase-k.md decision 9).
func (s *Service) RunMatcher(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.MatchPending(now)
		}
	}
}

// MatchPending checks every queued order against what the market did
// since it was placed and fills, or rejects, the ones that triggered.
// It returns how many orders changed state.
func (s *Service) MatchPending(now time.Time) int {
	if s.bars == nil {
		return 0
	}
	queued := s.store.Queued()
	changed := 0
	for _, q := range queued {
		if s.matchOne(q.userID, q.order.ID, now) {
			changed++
		}
	}
	return changed
}

// matchOne re-reads the order under the service lock (it may have been
// cancelled since the scan) and fills it if its trigger was hit.
func (s *Service) matchOne(userID, orderID string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.store.ByID(userID, orderID)
	if !ok || o.Status != "queued" {
		return false
	}
	detail, ok := s.quotes.Detail(o.Symbol)
	if !ok {
		return false
	}
	created, err := time.Parse("2006-01-02T15:04:05Z", o.CreatedAt)
	if err != nil {
		return false
	}

	if o.Type == "atc" {
		closeAt := atcClose(created)
		if now.Before(closeAt) {
			return false
		}
		s.book(userID, &o, detail, detail.LastPrice, "", closeAt)
		return true
	}

	// Only bars that start at or after the order's creation count, and
	// only once each (the per-order watermark).
	from := created.Unix()
	if w, ok := s.checked[o.ID]; ok && w >= from {
		from = w + 1
	}
	bars := s.bars.IntradayBars(o.Symbol, from-from%matchStep, now.Unix())
	for _, b := range bars {
		if b.Time < created.Unix() || b.Time <= s.checked[o.ID] || b.Time > now.Unix() {
			continue
		}
		s.checked[o.ID] = b.Time
		if price, leg, hit := barTrigger(o, b); hit {
			s.book(userID, &o, detail, price, leg, time.Unix(b.Time, 0))
			delete(s.checked, o.ID)
			return true
		}
	}
	return false
}

// book applies a triggered order to the ledger with its fee, or marks it
// rejected when the portfolio can't cover it at that moment (phase-k.md
// decision 8).
func (s *Service) book(userID string, o *Order, detail symbol.Detail, price float64, leg string, at time.Time) {
	fee := round2(price * o.Quantity * feeRateFor(detail))
	if err := s.ledger.ApplyFill(o.PortfolioID, o.Symbol, o.Side, o.Quantity, price, fee); err != nil {
		o.Status = "rejected"
		o.RejectReason = err.Error()
		s.store.Replace(userID, *o)
		return
	}
	o.Status = "filled"
	o.FilledPrice = price
	o.Fee = fee
	o.FilledAt = formatTime(at)
	o.TriggeredBy = leg
	s.store.Replace(userID, *o)
}

func feeRateFor(d symbol.Detail) float64 {
	if d.Exchange == symbol.ExchangeCrypto {
		return CryptoFeeRate
	}
	return FeeRate
}

// barTrigger applies phase-k.md decision 5's table to one bar: the fill
// price (a gap fills at the bar's open), the OCO leg that fired, and
// whether anything fired at all. When both OCO legs fall in the same bar
// the stop leg wins -- the conservative assumption.
func barTrigger(o Order, b market.Bar) (price float64, leg string, hit bool) {
	buy := o.Side == "buy"
	limitHit := func(p float64) (float64, bool) {
		if buy && b.Low <= p {
			return min(p, b.Open), true
		}
		if !buy && b.High >= p {
			return max(p, b.Open), true
		}
		return 0, false
	}
	stopHit := func(p float64) (float64, bool) {
		if buy && b.High >= p {
			return max(p, b.Open), true
		}
		if !buy && b.Low <= p {
			return min(p, b.Open), true
		}
		return 0, false
	}
	switch o.Type {
	case "limit":
		price, hit = limitHit(o.Price)
	case "stop":
		price, hit = stopHit(o.Price)
	case "oco":
		if p, ok := stopHit(o.StopPrice); ok {
			return p, "stop", true
		}
		if p, ok := limitHit(o.Price); ok {
			return p, "limit", true
		}
	}
	return price, "", hit
}

// triggeredNow is decision 6: an order that is already marketable at the
// ticket's current price fills at once, like a market order.
func triggeredNow(o Order, last float64) (leg string, hit bool) {
	buy := o.Side == "buy"
	limitNow := func(p float64) bool { return (buy && last <= p) || (!buy && last >= p) }
	stopNow := func(p float64) bool { return (buy && last >= p) || (!buy && last <= p) }
	switch o.Type {
	case "limit":
		return "", limitNow(o.Price)
	case "stop":
		return "", stopNow(o.Price)
	case "oco":
		if stopNow(o.StopPrice) {
			return "stop", true
		}
		if limitNow(o.Price) {
			return "limit", true
		}
	}
	return "", false
}

// atcClose is the 14:45 (Asia/Ho_Chi_Minh) close an ATC order placed at
// `created` fills at: the same weekday if placed before it, otherwise
// the next weekday's (phase-k.md decision 7).
func atcClose(created time.Time) time.Time {
	local := created.In(vnZone)
	c := time.Date(local.Year(), local.Month(), local.Day(), 14, 45, 0, 0, vnZone)
	if !local.Before(c) {
		c = c.AddDate(0, 0, 1)
	}
	for c.Weekday() == time.Saturday || c.Weekday() == time.Sunday {
		c = c.AddDate(0, 0, 1)
	}
	return c
}
