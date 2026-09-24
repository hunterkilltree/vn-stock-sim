package quant

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// historyDays covers SMA(200) on trading days (~275 sessions in 400
// calendar days).
const historyDays = 400

// snapshot computes each symbol's metrics at most once per request, for
// both the model's context block and the screen that runs afterwards.
type snapshot struct {
	svc      *Service
	universe []symbol.Symbol
	rows     map[string]*Row
	bars     map[string][]market.Bar
}

func newSnapshot(svc *Service, universe []symbol.Symbol) *snapshot {
	return &snapshot{svc: svc, universe: universe, rows: map[string]*Row{}, bars: map[string][]market.Bar{}}
}

func (s *snapshot) barsFor(sym string) []market.Bar {
	if b, ok := s.bars[sym]; ok {
		return b
	}
	to := s.svc.now().Unix()
	from := to - historyDays*24*60*60
	b := s.svc.bars.GetBars(sym, "1D", from, to)
	s.bars[sym] = b
	return b
}

func (s *snapshot) row(sym string) (Row, bool) {
	if r, ok := s.rows[sym]; ok {
		return *r, r != nil
	}
	d, ok := s.svc.symbols.Detail(sym)
	if !ok {
		s.rows[sym] = nil
		return Row{}, false
	}
	r := &Row{
		Symbol:        d.Symbol.Symbol,
		Exchange:      d.Exchange,
		Sector:        d.Sector,
		Price:         d.LastPrice,
		ChangePercent: d.ChangePercent,
		ROE:           d.ROE,
		PE:            d.PERatio,
		PB:            d.PBRatio,
		EPS:           d.EPS,
		DividendYield: d.DividendYield,
		MarketCap:     float64(d.MarketCap),
	}
	bars := s.barsFor(sym)
	if pts := market.RSI(bars, 14); len(pts) > 0 {
		r.RSI14 = pts[len(pts)-1].Value
	}
	if n := len(bars); n > 0 {
		k := min(20, n)
		var sum float64
		for _, b := range bars[n-k:] {
			sum += float64(b.Volume)
		}
		r.AvgVolume20 = math.Round(sum / float64(k))
	}
	s.rows[sym] = r
	return *r, true
}

func (s *snapshot) sma(sym string, period int) (float64, bool) {
	pts := market.SMA(s.barsFor(sym), period)
	if len(pts) == 0 {
		return 0, false
	}
	return pts[len(pts)-1].Value, true
}

func compare(a float64, op string, b float64) bool {
	switch op {
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	}
	return false
}

// metricValue returns the row's value in the same units the condition
// uses (price in thousand VND, market cap in billion VND).
func metricValue(r Row, field string) float64 {
	switch field {
	case "price":
		return r.Price / 1000
	case "changePercent":
		return r.ChangePercent
	case "rsi14":
		return r.RSI14
	case "roe":
		return r.ROE
	case "pe":
		return r.PE
	case "pb":
		return r.PB
	case "eps":
		return r.EPS
	case "dividendYield":
		return r.DividendYield
	case "marketCap":
		return r.MarketCap / 1e9
	case "avgVolume20":
		return r.AvgVolume20
	}
	return math.NaN()
}

func (s *snapshot) matches(r Row, c Condition) bool {
	if c.Field == "price_vs_sma" {
		v, ok := s.sma(r.Symbol, c.Period)
		return ok && compare(r.Price, c.Op, v)
	}
	return compare(metricValue(r, c.Field), c.Op, c.Value)
}

// screen runs validated conditions against real data -- the model never
// sees or computes the result set itself (phase-h.md ground rules).
func (s *snapshot) screen(spec ScreenSpec) ScreenOutcome {
	out := ScreenOutcome{
		Exchange:   spec.Exchange,
		Conditions: spec.Conditions,
		Labels:     conditionLabels(spec),
		Matches:    []Row{},
		SortedBy:   "symbol",
	}
	for _, sym := range s.universe {
		if spec.Exchange != "ALL" && sym.Exchange != spec.Exchange {
			continue
		}
		out.Universe++
		r, ok := s.row(sym.Symbol)
		if !ok {
			continue
		}
		pass := true
		for _, c := range spec.Conditions {
			if !s.matches(r, c) {
				pass = false
				break
			}
		}
		if pass {
			out.Matches = append(out.Matches, r)
		}
	}
	for _, c := range spec.Conditions {
		if c.Field == "rsi14" {
			out.SortedBy = "rsi14"
		}
	}
	sort.SliceStable(out.Matches, func(i, j int) bool {
		if out.SortedBy == "rsi14" {
			return out.Matches[i].RSI14 < out.Matches[j].RSI14
		}
		return out.Matches[i].Symbol < out.Matches[j].Symbol
	})
	return out
}

// Allowed value range per field; a condition outside it is dropped rather
// than clamped, since clamping would silently change what was asked.
var fieldRange = map[string][2]float64{
	"price":         {0, 10_000},
	"changePercent": {-100, 100},
	"rsi14":         {0, 100},
	"roe":           {-100, 1000},
	"pe":            {-1000, 10_000},
	"pb":            {-100, 1000},
	"eps":           {-1_000_000, 1_000_000},
	"dividendYield": {0, 100},
	"marketCap":     {0, 100_000_000},
	"avgVolume20":   {0, 10_000_000_000},
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// validatePlan keeps only what the app can actually run and reports the
// rest (phase-h.md decision 1).
func validatePlan(p Plan, universe []symbol.Symbol) (Plan, []string) {
	dropped := []string{}
	out := Plan{Reply: strings.TrimSpace(p.Reply)}
	if len([]rune(out.Reply)) > 2000 {
		out.Reply = string([]rune(out.Reply)[:2000])
	}

	ex := strings.ToUpper(strings.TrimSpace(p.Screen.Exchange))
	if !contains([]string{"ALL", "HOSE", "HNX", "UPCOM"}, ex) {
		if ex != "" {
			dropped = append(dropped, fmt.Sprintf("sàn %q không hợp lệ, dùng tất cả các sàn", p.Screen.Exchange))
		}
		ex = "ALL"
	}
	out.Screen = ScreenSpec{Exchange: ex, Conditions: []Condition{}}
	for _, c := range p.Screen.Conditions {
		if len(out.Screen.Conditions) >= 8 {
			dropped = append(dropped, "chỉ áp dụng tối đa 8 điều kiện")
			break
		}
		if !contains(conditionFields, c.Field) || !contains(conditionOps, c.Op) {
			dropped = append(dropped, fmt.Sprintf("điều kiện không hỗ trợ: %s %s", c.Field, c.Op))
			continue
		}
		if c.Field == "price_vs_sma" {
			if c.Period < 2 || c.Period > 250 {
				dropped = append(dropped, fmt.Sprintf("SMA(%d) nằm ngoài khoảng 2–250", c.Period))
				continue
			}
			c.Value = 0
		} else {
			rg := fieldRange[c.Field]
			if math.IsNaN(c.Value) || c.Value < rg[0] || c.Value > rg[1] {
				dropped = append(dropped, fmt.Sprintf("giá trị %v ngoài khoảng hợp lệ cho %s", c.Value, c.Field))
				continue
			}
			c.Period = 0
		}
		out.Screen.Conditions = append(out.Screen.Conditions, c)
	}

	st := p.Strategy
	st.Name = strings.TrimSpace(st.Name)
	if len([]rune(st.Name)) > 80 {
		st.Name = string([]rune(st.Name)[:80])
	}
	st.Symbol = strings.ToUpper(strings.TrimSpace(st.Symbol))
	if st.Symbol != "" {
		known := false
		for _, u := range universe {
			if u.Symbol == st.Symbol {
				known = true
			}
		}
		if !known {
			dropped = append(dropped, fmt.Sprintf("mã %s không có trong dữ liệu của ứng dụng", st.Symbol))
			st.Symbol = ""
		}
	}
	switch st.Kind {
	case "ema_crossover":
		if st.Fast < 2 || st.Fast > 100 || st.Slow <= st.Fast || st.Slow > 250 {
			dropped = append(dropped, fmt.Sprintf("chu kỳ EMA %d/%d không hợp lệ", st.Fast, st.Slow))
			st = StrategySpec{Kind: "none"}
		} else {
			st.RSIEntry, st.RSIExit, st.StopLossPercent, st.TrendSMA = 0, 0, 0, 0
		}
	case "rsi_reversion":
		validTrend := st.TrendSMA == 0 || (st.TrendSMA >= 2 && st.TrendSMA <= 250)
		if st.RSIEntry < 5 || st.RSIEntry > 60 || st.RSIExit < 40 || st.RSIExit > 95 || st.RSIExit <= st.RSIEntry ||
			st.StopLossPercent < 0 || st.StopLossPercent > 50 || !validTrend {
			dropped = append(dropped, "tham số chiến lược RSI không hợp lệ")
			st = StrategySpec{Kind: "none"}
		} else {
			st.Fast, st.Slow = 0, 0
		}
	case "none", "":
		st = StrategySpec{Kind: "none"}
	default:
		dropped = append(dropped, fmt.Sprintf("loại chiến lược không hỗ trợ: %s", st.Kind))
		st = StrategySpec{Kind: "none"}
	}
	out.Strategy = st

	if out.Reply == "" {
		out.Reply = "Đây là những gì tôi hiểu từ yêu cầu của bạn."
	}
	return out, dropped
}

var fieldLabel = map[string]string{
	"price":         "Giá",
	"changePercent": "% thay đổi",
	"rsi14":         "RSI(14)",
	"roe":           "ROE",
	"pe":            "P/E",
	"pb":            "P/B",
	"eps":           "EPS",
	"dividendYield": "Cổ tức",
	"marketCap":     "Vốn hoá",
	"avgVolume20":   "KLTB20",
}

// conditionLabels renders Quant.dc.html's condition chips, e.g.
// "sàn = HOSE", "RSI(14) < 35", "ROE > 15%", "KLTB20 > 1 tr",
// "giá > SMA(200)".
func conditionLabels(spec ScreenSpec) []string {
	labels := []string{}
	if spec.Exchange != "" && spec.Exchange != "ALL" {
		labels = append(labels, "sàn = "+spec.Exchange)
	}
	for _, c := range spec.Conditions {
		if c.Field == "price_vs_sma" {
			labels = append(labels, fmt.Sprintf("giá %s SMA(%d)", c.Op, c.Period))
			continue
		}
		var v string
		switch c.Field {
		case "roe", "dividendYield", "changePercent":
			v = formatVN(c.Value, 2) + "%"
		case "avgVolume20":
			v = formatVN(c.Value/1_000_000, 2) + " tr"
		case "price":
			v = formatVN(c.Value, 2) + " nghìn ₫"
		case "marketCap":
			v = formatVN(c.Value, 0) + " tỷ ₫"
		default:
			v = formatVN(c.Value, 2)
		}
		labels = append(labels, fmt.Sprintf("%s %s %s", fieldLabel[c.Field], c.Op, v))
	}
	return labels
}

// formatVN formats with "." thousands and "," decimals, trimming trailing
// zeros ("15" not "15,00"; "1,5" not "1,50").
func formatVN(v float64, decimals int) string {
	s := fmt.Sprintf("%.*f", decimals, math.Abs(v))
	intPart, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	out := b.String()
	if frac != "" {
		out += "," + frac
	}
	if v < 0 {
		out = "−" + out
	}
	return out
}
