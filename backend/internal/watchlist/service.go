package watchlist

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"

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

func (s *Service) Add(userID, sym string) {
	s.store.Add(userID, sym)
}

func (s *Service) Remove(userID, sym string) {
	s.store.Remove(userID, sym)
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
		}
		items = append(items, item)
	}
	return items
}
