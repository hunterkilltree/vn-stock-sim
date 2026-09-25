package order

import "time"

type Order struct {
	ID          string  `json:"id"`
	PortfolioID string  `json:"portfolioId"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Type        string  `json:"type"`
	Quantity    float64 `json:"quantity"`
	Status      string  `json:"status"`
	// Price is the requested limit/stop trigger price for a non-market
	// order -- zero (omitted) for "market", which fills at whatever the
	// live quote is instead. Added in Phase E: the order ticket
	// (OrderTicket.tsx) always collected this, but it was never sent to
	// or stored by the backend, so a queued order's own Pending panel
	// had no real price to show -- a found gap, not a Phase E feature.
	Price       float64 `json:"price,omitempty"`
	StopPrice   float64 `json:"stopPrice,omitempty"`
	FilledPrice float64 `json:"filledPrice,omitempty"`
	// Fee is the simulated trading fee (FeeRate / CryptoFeeRate), charged
	// only on an actual fill -- a queued order shows fee 0 until the
	// matcher fills it (matcher.go, phase-k.md).
	Fee       float64 `json:"fee"`
	FilledAt  string  `json:"filledAt,omitempty"`
	CreatedAt string  `json:"createdAt"`
	// TriggeredBy says which leg of an OCO filled: "limit" or "stop".
	TriggeredBy string `json:"triggeredBy,omitempty"`
	// RejectReason is set when a triggered order could not be booked
	// (status "rejected": not enough cash or shares at that moment).
	RejectReason string `json:"rejectReason,omitempty"`
}

// FeeRate is the simulated paper-trading fee (Detail.dc.html shows
// 0.15%), applied to the notional value of every actual fill.
const FeeRate = 0.0015

// CryptoFeeRate is Crypto-Detail.dc.html's "Phí mô phỏng (0,10%)".
const CryptoFeeRate = 0.001

type createRequest struct {
	// PortfolioID is optional -- omitted, it resolves to the caller's
	// default portfolio (phase-b.md decision 1), so pre-Phase-B callers
	// that don't yet pick a specific portfolio keep working.
	PortfolioID string `json:"portfolioId"`
	Symbol      string `json:"symbol" binding:"required"`
	Side        string `json:"side" binding:"required,oneof=buy sell"`
	// market == MP, limit == LO, atc/stop are Detail.dc.html's order
	// ticket, oco is Crypto-Detail's. Everything but market is queued and
	// filled by the matcher (matcher.go, phase-k.md decisions 4-10).
	Type     string  `json:"type" binding:"required,oneof=market limit atc stop oco"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	// Price is required for non-market types (see Order.Price above);
	// market orders ignore it and fill at the live quote instead.
	Price float64 `json:"price"`
	// StopPrice is OCO's second leg (Crypto-Detail's order types): the
	// order carries both a limit price and a stop price.
	StopPrice float64 `json:"stopPrice"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
