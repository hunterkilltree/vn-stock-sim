package market

import "time"

// PriceLevel is one row of the order book (Detail.dc.html's bid/ask
// depth table).
type PriceLevel struct {
	Price  float64 `json:"price"`
	Volume int64   `json:"volume"`
}

// orderBookTick is a hardcoded 100 VND step, not a real per-symbol
// tickSize lookup -- market has no import dependency on symbol (see
// CLAUDE.md's layering rule: a Service depends only on small ports it
// defines itself). Every current mock symbol's tickSize happens to be
// 100, so this is a documented simplification (phase-b.md decision 3)
// rather than a layering violation; revisit if a symbol with a
// different tick size is ever added.
const orderBookTick = 100.0
const orderBookLevels = 6

// GetOrderBook synthesizes a deterministic order book around lastPrice:
// orderBookLevels price levels on each side, spaced by orderBookTick,
// with a pseudo-random but reproducible volume per level. Deterministic
// in (sym, lastPrice, time.Now().Unix()) -- repeated calls within the
// same second return identical levels, matching the same reproducibility
// discipline as closeFor/indexValueFor.
func (p *MockProvider) GetOrderBook(sym string, lastPrice float64) (bids, asks []PriceLevel) {
	t := time.Now().Unix()
	bids = make([]PriceLevel, 0, orderBookLevels)
	asks = make([]PriceLevel, 0, orderBookLevels)

	for i := 1; i <= orderBookLevels; i++ {
		bidPrice := roundToStep(lastPrice-float64(i)*orderBookTick, orderBookTick)
		askPrice := roundToStep(lastPrice+float64(i)*orderBookTick, orderBookTick)
		bids = append(bids, PriceLevel{Price: bidPrice, Volume: levelVolume(sym, "bid", i, t)})
		asks = append(asks, PriceLevel{Price: askPrice, Volume: levelVolume(sym, "ask", i, t)})
	}
	return bids, asks
}

func levelVolume(sym, side string, level int, t int64) int64 {
	seed := seedFromSymbol(sym+side) ^ (t + int64(level)*97)
	seed = (seed*1103515245 + 12345) & 0x7fffffff
	// 1,000-20,000 shares per level -- a plausible depth for a mock book.
	return 1000 + seed%19000
}

func roundToStep(price, step float64) float64 {
	if price < step {
		price = step
	}
	n := float64(int64(price/step + 0.5))
	return round2(n * step)
}
