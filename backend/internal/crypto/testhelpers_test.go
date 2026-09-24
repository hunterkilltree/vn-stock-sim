package crypto

import (
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

func timeNow() time.Time { return time.Now() }

type stubStock struct{}

func (stubStock) Detail(sym string) (symbol.Detail, bool) {
	return symbol.Detail{Symbol: symbol.Symbol{Symbol: sym, Exchange: "HOSE"}}, true
}
