package symbol

import (
	"errors"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/valuation"
)

// Financials is a company's reported history: every fiscal year and
// quarter on record with its PublishedAt, plus dividend and bonus-issue
// events. valuation.AsOf turns it into point-in-time inputs.
type Financials struct {
	Periods   []valuation.Period
	Dividends []valuation.Dividend
	// IsBank: banks report a different statement layout, and debt/equity
	// means nothing for them (phase-holding-horizon.md H2).
	IsBank bool
}

// ErrNoFinancials: the provider has no history for this symbol (crypto,
// an index, a ticker the vendor doesn't cover).
var ErrNoFinancials = errors.New("symbol: no financials")

// FinancialsProvider is the port for fundamentals history
// (phase-fundamentals-data.md). The VCI adapter goes behind it; until
// then nothing implements it and the rating card keeps using Detail's
// seeded snapshot.
type FinancialsProvider interface {
	History(sym string) (Financials, error)
}
