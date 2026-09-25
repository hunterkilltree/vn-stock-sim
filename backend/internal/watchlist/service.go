package watchlist

import (
	"errors"
	"strings"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var ErrUnknownSymbol = errors.New("unknown symbol")

// QuotePort is the small interface watchlist depends on for live quote
// fields (lastPrice/change/volume) — satisfied by *symbol.Service. Kept as
// an interface (not a direct *symbol.Service field) so a future dedicated
// quote/streaming service can be swapped in without touching this file.
type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type Service struct {
	store  *MemoryStore
	quotes QuotePort
}

func NewService(store *MemoryStore, quotes QuotePort) *Service {
	return &Service{store: store, quotes: quotes}
}

// Add stores the canonical symbol ("fpt" -> "FPT", "btcusdt" ->
// "BTCUSDT"; pairs by their full name, so a coin never shadows a stock
// ticker) and refuses ones the app has no quote for -- before Phase K any
// string was stored.
func (s *Service) Add(userID, sym string) error {
	d, ok := s.quotes.Detail(strings.ToUpper(strings.TrimSpace(sym)))
	if !ok {
		return ErrUnknownSymbol
	}
	s.store.Add(userID, d.Symbol.Symbol)
	return nil
}

func (s *Service) Remove(userID, sym string) {
	s.store.Remove(userID, strings.ToUpper(sym))
}

func (s *Service) List(userID string) []Item {
	entries := s.store.List(userID)
	items := make([]Item, 0, len(entries))
	for _, e := range entries {
		item := Item{Symbol: e.symbol, AddedAt: e.addedAt.UTC().Format("2006-01-02T15:04:05Z")}
		if d, ok := s.quotes.Detail(e.symbol); ok {
			item.LastPrice = d.LastPrice
			item.Change = d.Change
			item.ChangePercent = d.ChangePercent
			item.CompanyName = d.CompanyName
			item.Exchange = d.Exchange
		}
		items = append(items, item)
	}
	return items
}
