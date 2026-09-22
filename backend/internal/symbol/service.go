package symbol

import "math"

// QuotePort is the small interface Service depends on to derive a live
// last price/change instead of carrying its own separately-seeded
// static value. Satisfied by *market.Service. Kept as primitive types
// (not market.Bar) so this package has no import dependency on market --
// only main.go wires the two together.
type QuotePort interface {
	// LatestClose returns the most recent two closes for sym; ok is
	// false if the quote source doesn't have enough data.
	LatestClose(sym string) (today, yesterday float64, ok bool)
}

type Service struct {
	provider Provider
	quotes   QuotePort
}

func NewService(provider Provider, quotes QuotePort) *Service {
	return &Service{provider: provider, quotes: quotes}
}

func (s *Service) Search(query, exchange string, page, pageSize int) ([]Symbol, int) {
	all := s.provider.Search(query, exchange)
	total := len(all)
	start := (page - 1) * pageSize
	if start >= total {
		return []Symbol{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total
}

// Detail returns fundamentals from the provider (company/exchange/sector/
// tickSize/marketCap/peRatio/pbRatio/eps/dividendYield) with the price
// fields (lastPrice/change/changePercent) overridden from the quote
// source when available. Without this override those two were entirely
// independent mock values -- the provider's seeded lastPrice had no
// relationship to what the chart (backed by market data) actually
// showed as the latest close, which is the bug this fixes. The
// provider's seeded price fields remain as a fallback if the quote
// source has no data for this symbol.
func (s *Service) Detail(sym string) (Detail, bool) {
	detail, ok := s.provider.Detail(sym)
	if !ok {
		return Detail{}, false
	}
	if today, yesterday, ok := s.quotes.LatestClose(detail.Symbol.Symbol); ok {
		detail.LastPrice = today
		detail.Change = today - yesterday
		if yesterday != 0 {
			detail.ChangePercent = detail.Change / yesterday * 100
		}
		detail.Reference = yesterday
		band := priceBandPercent(detail.Exchange)
		detail.Ceiling = roundToTick(yesterday*(1+band), detail.TickSize)
		detail.Floor = roundToTick(yesterday*(1-band), detail.TickSize)
	}
	return detail, true
}

// priceBandPercent is the real daily price-band rule for each Vietnamese
// exchange -- ceiling/floor are this percent above/below the reference
// (previous close), not a design choice. Unknown exchanges fall back to
// HOSE's band rather than 0, so ceiling/floor never silently collapse to
// the reference price.
func priceBandPercent(exchange string) float64 {
	switch exchange {
	case "HOSE":
		return 0.07
	case "HNX":
		return 0.10
	case "UPCOM":
		return 0.15
	default:
		return 0.07
	}
}

func roundToTick(price float64, tickSize int) float64 {
	if tickSize <= 0 {
		tickSize = 100
	}
	t := float64(tickSize)
	return math.Round(price/t) * t
}
