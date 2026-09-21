package symbol

import "strings"

// Provider is the port a Service depends on for symbol data. V1 ships
// MockProvider (fixture data); swapping to a licensed market-data vendor
// later means writing a new adapter behind this same interface — the
// Service and Handler never change (see CLAUDE.md's dependency-inversion
// rule for market/order/ai, applied the same way here).
type Provider interface {
	Search(query, exchange string) []Symbol
	Detail(sym string) (Detail, bool)
}

type MockProvider struct {
	details map[string]Detail
	order   []string
}

// NewMockProvider seeds a small fixture set of real HOSE/HNX tickers so the
// stock browser, search, and detail page have something concrete to render
// against before a licensed data feed is wired up.
func NewMockProvider() *MockProvider {
	seed := []Detail{
		{Symbol: Symbol{Symbol: "VNM", CompanyName: "Vietnam Dairy Products JSC", Exchange: "HOSE", Sector: "Consumer Staples", TickSize: 100}, LastPrice: 68500, Change: 500, ChangePercent: 0.74, MarketCap: 143200000000000, PERatio: 16.2, PBRatio: 4.1, EPS: 4228, DividendYield: 5.8},
		{Symbol: Symbol{Symbol: "VCB", CompanyName: "Joint Stock Commercial Bank for Foreign Trade of Vietnam", Exchange: "HOSE", Sector: "Financials", TickSize: 100}, LastPrice: 91200, Change: -300, ChangePercent: -0.33, MarketCap: 480500000000000, PERatio: 14.8, PBRatio: 2.9, EPS: 6162, DividendYield: 1.2},
		{Symbol: Symbol{Symbol: "HPG", CompanyName: "Hoa Phat Group JSC", Exchange: "HOSE", Sector: "Materials", TickSize: 100}, LastPrice: 27650, Change: 150, ChangePercent: 0.55, MarketCap: 160900000000000, PERatio: 11.4, PBRatio: 1.6, EPS: 2426, DividendYield: 2.9},
		{Symbol: Symbol{Symbol: "FPT", CompanyName: "FPT Corporation", Exchange: "HOSE", Sector: "Information Technology", TickSize: 100}, LastPrice: 134000, Change: 1200, ChangePercent: 0.9, MarketCap: 176300000000000, PERatio: 22.1, PBRatio: 6.7, EPS: 6063, DividendYield: 1.1},
		{Symbol: Symbol{Symbol: "SHB", CompanyName: "Saigon Hanoi Commercial Bank", Exchange: "HNX", Sector: "Financials", TickSize: 100}, LastPrice: 12300, Change: 0, ChangePercent: 0, MarketCap: 45200000000000, PERatio: 8.9, PBRatio: 1.1, EPS: 1382, DividendYield: 3.4},
	}
	p := &MockProvider{details: make(map[string]Detail, len(seed))}
	for _, d := range seed {
		p.details[d.Symbol.Symbol] = d
		p.order = append(p.order, d.Symbol.Symbol)
	}
	return p
}

func (p *MockProvider) Search(query, exchange string) []Symbol {
	query = strings.ToUpper(strings.TrimSpace(query))
	exchange = strings.ToUpper(strings.TrimSpace(exchange))
	var out []Symbol
	for _, sym := range p.order {
		d := p.details[sym]
		if exchange != "" && d.Exchange != exchange {
			continue
		}
		if query != "" && !strings.Contains(d.Symbol.Symbol, query) && !strings.Contains(strings.ToUpper(d.CompanyName), query) {
			continue
		}
		out = append(out, d.Symbol)
	}
	return out
}

func (p *MockProvider) Detail(sym string) (Detail, bool) {
	d, ok := p.details[strings.ToUpper(sym)]
	return d, ok
}
