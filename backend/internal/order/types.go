package order

import "time"

type Order struct {
	ID          string `json:"id"`
	PortfolioID string `json:"portfolioId"`
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Type        string `json:"type"`
	Quantity    int64  `json:"quantity"`
	Status      string `json:"status"`
	// Price is the requested limit/stop trigger price for a non-market
	// order -- zero (omitted) for "market", which fills at whatever the
	// live quote is instead. Added in Phase E: the order ticket
	// (OrderTicket.tsx) always collected this, but it was never sent to
	// or stored by the backend, so a queued order's own Pending panel
	// had no real price to show -- a found gap, not a Phase E feature.
	Price       float64 `json:"price,omitempty"`
	FilledPrice float64 `json:"filledPrice,omitempty"`
	// Fee is the simulated 0.15% trading fee (FeeRate), charged only on an
	// actual fill -- a queued order (limit/atc/stop, none of which fill
	// immediately in V1, see service.go) shows fee 0 until a real matching
	// engine later fills it.
	Fee       float64 `json:"fee"`
	FilledAt  string  `json:"filledAt,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

// FeeRate is the simulated paper-trading fee (Detail.dc.html shows
// 0.15%), applied to the notional value of every actual fill.
const FeeRate = 0.0015

type createRequest struct {
	// PortfolioID is optional -- omitted, it resolves to the caller's
	// default portfolio (phase-b.md decision 1), so pre-Phase-B callers
	// that don't yet pick a specific portfolio keep working.
	PortfolioID string `json:"portfolioId"`
	Symbol      string `json:"symbol" binding:"required"`
	Side        string `json:"side" binding:"required,oneof=buy sell"`
	// market == MP, limit == LO, atc/stop are new (Detail.dc.html's order
	// ticket). atc/stop are accepted and queued like limit today -- V1 has
	// no real matching engine yet (see RESUME.md future work).
	Type     string `json:"type" binding:"required,oneof=market limit atc stop"`
	Quantity int64  `json:"quantity" binding:"required,gt=0"`
	// Price is required for non-market types (see Order.Price above);
	// market orders ignore it and fill at the live quote instead.
	Price float64 `json:"price"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
