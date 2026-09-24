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

// NewMockProvider seeds real HOSE/HNX/UPCOM tickers: the ones
// design/screens/Main.dc.html's heatmap and movers show, plus a few large
// caps per sector (phase-i.md decision 14). Sector names are Vietnamese,
// as the design writes them. Company/exchange are real; the fundamentals
// (market cap, P/E, P/B, ROE, EPS, dividend yield) are hand-seeded
// approximations for a simulator, not a data feed -- live prices come
// from the market data provider and override LastPrice/Change in
// Service.Detail.
func NewMockProvider() *MockProvider {
	const (
		bank    = "Ngân hàng"
		broker  = "Chứng khoán"
		realty  = "Bất động sản"
		tech    = "Công nghệ & Viễn thông"
		steel   = "Thép & Vật liệu"
		consume = "Tiêu dùng"
		retail  = "Bán lẻ"
		energy  = "Năng lượng"
		infra   = "Hàng không & Vận tải"
	)
	// capBn is market cap in billion VND; price is a rough seed used only
	// until the quote source answers.
	d := func(sym, name, exchange, sector string, price float64, capBn int64, pe, pb, roe, eps, dy float64) Detail {
		return Detail{
			Symbol:    Symbol{Symbol: sym, CompanyName: name, Exchange: exchange, Sector: sector, TickSize: 100},
			LastPrice: price, MarketCap: capBn * 1_000_000_000,
			PERatio: pe, PBRatio: pb, ROE: roe, EPS: eps, DividendYield: dy,
		}
	}
	seed := []Detail{
		d("VCB", "Joint Stock Commercial Bank for Foreign Trade of Vietnam", "HOSE", bank, 91200, 480500, 14.8, 2.9, 19.7, 6162, 1.2),
		d("BID", "Bank for Investment and Development of Vietnam", "HOSE", bank, 45300, 258000, 12.6, 2.1, 17.4, 3595, 1.1),
		d("CTG", "Vietnam Joint Stock Commercial Bank for Industry and Trade", "HOSE", bank, 38200, 205000, 9.1, 1.5, 17.9, 4198, 1.3),
		d("TCB", "Vietnam Technological and Commercial Joint Stock Bank", "HOSE", bank, 24500, 173000, 8.2, 1.3, 15.2, 2988, 2.0),
		d("MBB", "Military Commercial Joint Stock Bank", "HOSE", bank, 24100, 128000, 6.4, 1.3, 21.8, 3766, 2.4),
		d("VPB", "Vietnam Prosperity Joint Stock Commercial Bank", "HOSE", bank, 19400, 154000, 10.8, 1.1, 10.9, 1796, 2.6),
		d("SHB", "Saigon Hanoi Commercial Bank", "HNX", bank, 12300, 45200, 8.9, 1.1, 12.4, 1382, 3.4),
		d("SSI", "SSI Securities Corporation", "HOSE", broker, 26400, 51900, 17.2, 1.9, 11.4, 1535, 3.8),
		d("VND", "VNDIRECT Securities Corporation", "HOSE", broker, 16800, 25600, 14.6, 1.3, 9.2, 1151, 3.0),
		d("HCM", "Ho Chi Minh City Securities Corporation", "HOSE", broker, 28900, 20800, 19.4, 1.9, 10.6, 1490, 3.5),
		d("VCI", "Vietcap Securities Joint Stock Company", "HOSE", broker, 37100, 26600, 24.1, 2.2, 9.6, 1539, 1.4),
		d("MBS", "MB Securities Joint Stock Company", "HNX", broker, 29600, 16200, 18.8, 2.3, 12.9, 1574, 2.0),
		d("SHS", "Saigon - Hanoi Securities Joint Stock Company", "HNX", broker, 15600, 12700, 14.2, 1.1, 8.3, 1099, 3.2),
		d("VHM", "Vinhomes Joint Stock Company", "HOSE", realty, 41200, 169000, 5.9, 0.9, 16.6, 6983, 0.0),
		d("VIC", "Vingroup Joint Stock Company", "HOSE", realty, 42500, 162000, 28.4, 1.2, 4.3, 1496, 0.0),
		d("NVL", "No Va Land Investment Group Corporation", "HOSE", realty, 12350, 24100, 0, 0.5, -4.2, -310, 0.0),
		d("KDH", "Khang Dien House Trading and Investment JSC", "HOSE", realty, 32800, 33100, 33.6, 1.7, 5.2, 976, 1.5),
		d("DXG", "Dat Xanh Group Joint Stock Company", "HOSE", realty, 16900, 14700, 38.2, 0.9, 2.5, 442, 0.0),
		d("PDR", "Phat Dat Real Estate Development Corporation", "HOSE", realty, 19200, 16800, 31.5, 1.6, 5.1, 610, 0.0),
		d("FPT", "FPT Corporation", "HOSE", tech, 134000, 176300, 22.1, 6.7, 27.9, 6063, 1.1),
		d("CMG", "CMC Corporation", "HOSE", tech, 44900, 9500, 24.8, 2.9, 12.3, 1810, 1.1),
		d("ELC", "Electronics Communications Technology Investment Development Corporation", "HOSE", tech, 24900, 2600, 21.3, 2.4, 11.6, 1169, 2.0),
		d("ITD", "Innovative Technology Development Corporation", "HOSE", tech, 13600, 600, 15.9, 1.2, 7.8, 855, 0.0),
		d("SAM", "SAM Holdings Corporation", "HOSE", tech, 6480, 2500, 60.2, 0.7, 1.1, 108, 0.0),
		d("VGI", "Viettel Global Investment JSC", "UPCOM", tech, 96700, 294000, 41.6, 5.8, 14.2, 2325, 0.0),
		d("HPG", "Hoa Phat Group JSC", "HOSE", steel, 27650, 160900, 11.4, 1.6, 14.1, 2426, 2.9),
		d("HSG", "Hoa Sen Group", "HOSE", steel, 18900, 11700, 16.7, 1.1, 6.8, 1132, 2.6),
		d("NKG", "Nam Kim Steel Joint Stock Company", "HOSE", steel, 17200, 7700, 14.3, 1.0, 7.1, 1203, 0.0),
		d("VNM", "Vietnam Dairy Products JSC", "HOSE", consume, 68500, 143200, 16.2, 4.1, 25.3, 4228, 5.8),
		d("MSN", "Masan Group Corporation", "HOSE", consume, 76800, 110000, 45.2, 2.9, 6.5, 1699, 1.0),
		d("SAB", "Saigon Beer-Alcohol-Beverage Corporation", "HOSE", consume, 52300, 67000, 17.8, 2.9, 17.4, 2938, 5.7),
		d("MWG", "Mobile World Investment Corporation", "HOSE", retail, 61200, 89600, 24.3, 3.1, 13.4, 2519, 0.8),
		d("FRT", "FPT Digital Retail Joint Stock Company", "HOSE", retail, 178400, 24300, 72.6, 9.4, 13.8, 2457, 0.0),
		d("PNJ", "Phu Nhuan Jewelry Joint Stock Company", "HOSE", retail, 88600, 29900, 14.9, 2.7, 19.1, 5946, 2.3),
		d("GAS", "PetroVietnam Gas Joint Stock Corporation", "HOSE", energy, 67800, 158800, 14.8, 2.5, 17.2, 4581, 5.2),
		d("PLX", "Vietnam National Petroleum Group", "HOSE", energy, 37900, 48200, 17.6, 1.6, 9.4, 2153, 3.3),
		d("POW", "PetroVietnam Power Corporation", "HOSE", energy, 12100, 28300, 26.9, 0.9, 3.4, 450, 0.0),
		d("ACV", "Airports Corporation of Vietnam", "UPCOM", infra, 111000, 241600, 27.1, 4.8, 18.5, 4096, 0.6),
		d("VJC", "Vietjet Aviation Joint Stock Company", "HOSE", infra, 101500, 55000, 64.8, 3.5, 5.6, 1566, 0.0),
		d("GMD", "Gemadept Corporation", "HOSE", infra, 60100, 24900, 15.2, 2.1, 14.4, 3954, 3.3),
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
