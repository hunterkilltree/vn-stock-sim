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

	// The buy's trade plan (phase-risk-rules.md rule 1): protective stop,
	// target and setup. StopLoss is not StopPrice, which is an OCO leg.
	StopLoss float64 `json:"stopLoss,omitempty"`
	Target   float64 `json:"target,omitempty"`
	Setup    string  `json:"setup,omitempty"`
	// PlanID is the TradePlan this fill opened or added to.
	PlanID string `json:"planId,omitempty"`
	// Violations are the rule findings a coach-mode portfolio let through
	// (riskrules finding codes).
	Violations []string `json:"violations,omitempty"`
}

// TradePlan is one position's plan, opened by the first filled buy that
// carries a stop and closed when the position returns to zero
// (phase-risk-rules.md design item 2). InitialStop never changes after
// creation: R is measured against it, whatever happens to CurrentStop.
type TradePlan struct {
	ID           string  `json:"id"`
	PortfolioID  string  `json:"portfolioId"`
	Symbol       string  `json:"symbol"`
	Setup        string  `json:"setup,omitempty"`
	PlannedEntry float64 `json:"plannedEntry"`
	InitialStop  float64 `json:"initialStop"`
	CurrentStop  float64 `json:"currentStop"`
	Target       float64 `json:"target,omitempty"`
	OpenedAt     string  `json:"openedAt"`
	ClosedAt     string  `json:"closedAt,omitempty"`
	ExitPrice    float64 `json:"exitPrice,omitempty"`
	// RMultiple = (exit − planned entry) ÷ (planned entry − initial stop),
	// set when the plan closes.
	RMultiple float64 `json:"rMultiple,omitempty"`
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
	// A buy's trade plan (phase-risk-rules.md slice 1). Adding to an open
	// plan may omit StopLoss: the plan's current stop applies.
	StopLoss float64 `json:"stopLoss" binding:"gte=0"`
	Target   float64 `json:"target" binding:"gte=0"`
	Setup    string  `json:"setup" binding:"max=200"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
