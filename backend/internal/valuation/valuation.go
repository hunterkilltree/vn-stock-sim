// Package valuation turns the well-known fair-value rules into buy
// prices: pure functions over a symbol's fundamentals, no I/O, so the
// same code can feed the detail page, a screener field and (later) a
// backtest entry rule. See phase-valuation.md.
package valuation

import (
	"math"
)

// Fundamentals are per-share VND values. In a backtest they must be
// point-in-time (only data published on or before the bar date); for
// today's buy zone the current snapshot is fine. Zero means unknown.
type Fundamentals struct {
	Price     float64
	EPS       float64 // trailing 12-month EPS
	BVPS      float64 // book value per share
	ROE       float64 // percent
	GrowthPct float64 // expected EPS growth, % per year (12 = 12%)
	CashDPS   float64 // trailing 12-month cash dividend per share
	AvgHistPE float64 // e.g. 5-year average P/E
	StdHistPE float64 // standard deviation of that P/E
	HighYield float64 // historical high cash dividend yield (0.06 = 6%)
}

type Method string

const (
	GrahamNumber  Method = "graham_number"
	GrahamFormula Method = "graham_formula"
	LynchFair     Method = "lynch_fair_value"
	Rule1Sticker  Method = "rule1_sticker"
	WeissYield    Method = "weiss_yield"
	PEBand        Method = "pe_band"
)

// Methods is display order.
var Methods = []Method{GrahamNumber, GrahamFormula, LynchFair, Rule1Sticker, WeissYield, PEBand}

// Params are the market-wide inputs. Margin of safety is per method
// (phase-valuation.md decision 2): Rule #1 is defined at 50%, the Graham
// rules at 25-33%, and Lynch / Weiss / P/E band already embed theirs.
type Params struct {
	MarginOfSafety map[Method]float64
	BaseYieldPct   float64 // Graham's 4.4 (AAA yield, 1962)
	BondYieldPct   float64 // VN 10-year government yield today
	MARR           float64 // Rule #1 minimum acceptable return, 0.15
	Years          int     // Rule #1 horizon, 10
	MaxGrowthPct   float64 // cap g so one boom year can't explode the value
}

// BondYieldPct: VN 10-year government bond yield, as of BondYieldAsOf.
// Taken from the user's research note (State Treasury auction); not
// independently verified -- a backtest needs a time series instead.
const (
	BondYieldPct  = 4.43
	BondYieldAsOf = "2026-09-09"
)

func DefaultParams() Params {
	return Params{
		MarginOfSafety: map[Method]float64{
			GrahamNumber:  0.25,
			GrahamFormula: 0.25,
			Rule1Sticker:  0.5,
		},
		BaseYieldPct: 4.4,
		BondYieldPct: BondYieldPct,
		MARR:         0.15,
		Years:        10,
		MaxGrowthPct: 25,
	}
}

// Value is one method's answer. When Applicable is false, Reason says
// why in Vietnamese, for the UI -- the row is shown, not hidden.
type Value struct {
	Method     Method  `json:"method"`
	Applicable bool    `json:"applicable"`
	Reason     string  `json:"reason,omitempty"`
	FairValue  float64 `json:"fairValue,omitempty"`
	BuyPrice   float64 `json:"buyPrice,omitempty"`
	MOS        float64 `json:"marginOfSafety"`
}

func notApplicable(m Method, reason string) Value {
	return Value{Method: m, Reason: reason}
}

// Evaluate computes one method's fair value and buy price (before tick
// rounding).
func Evaluate(m Method, f Fundamentals, p Params) Value {
	g := f.GrowthPct
	if p.MaxGrowthPct > 0 {
		g = math.Min(g, p.MaxGrowthPct)
	}
	mos := p.MarginOfSafety[m]
	var fair float64
	switch m {
	case GrahamNumber:
		if f.EPS <= 0 {
			return notApplicable(m, "Doanh nghiệp không có lãi (EPS ≤ 0)")
		}
		if f.BVPS <= 0 {
			return notApplicable(m, "Thiếu giá trị sổ sách")
		}
		fair = math.Sqrt(22.5 * f.EPS * f.BVPS)
	case GrahamFormula:
		if f.EPS <= 0 {
			return notApplicable(m, "Doanh nghiệp không có lãi (EPS ≤ 0)")
		}
		if f.GrowthPct == 0 {
			return notApplicable(m, "Thiếu dữ liệu tăng trưởng EPS")
		}
		if p.BondYieldPct <= 0 || 8.5+2*g <= 0 {
			return notApplicable(m, "Tăng trưởng âm quá sâu")
		}
		fair = f.EPS * (8.5 + 2*g) * p.BaseYieldPct / p.BondYieldPct
	case LynchFair:
		if f.EPS <= 0 {
			return notApplicable(m, "Doanh nghiệp không có lãi (EPS ≤ 0)")
		}
		if f.GrowthPct == 0 {
			return notApplicable(m, "Thiếu dữ liệu tăng trưởng EPS")
		}
		if g < 5 {
			return notApplicable(m, "Tăng trưởng dưới 5%/năm")
		}
		fair = f.EPS * g // fair P/E = growth rate (PEG = 1)
	case Rule1Sticker:
		if f.EPS <= 0 {
			return notApplicable(m, "Doanh nghiệp không có lãi (EPS ≤ 0)")
		}
		if g <= 0 {
			return notApplicable(m, "Thiếu dữ liệu tăng trưởng EPS")
		}
		futurePE := 2 * g
		if f.AvgHistPE > 0 && f.AvgHistPE < futurePE {
			futurePE = f.AvgHistPE
		}
		n := float64(p.Years)
		fair = f.EPS * math.Pow(1+g/100, n) * futurePE / math.Pow(1+p.MARR, n)
	case WeissYield:
		if f.CashDPS <= 0 {
			return notApplicable(m, "Không trả cổ tức tiền mặt")
		}
		if f.HighYield <= 0 {
			return notApplicable(m, "Cần lịch sử tỷ suất cổ tức")
		}
		// Buy when the yield is within 10% of its historical high.
		fair = f.CashDPS / (0.9 * f.HighYield)
	case PEBand:
		if f.EPS <= 0 {
			return notApplicable(m, "Doanh nghiệp không có lãi (EPS ≤ 0)")
		}
		if f.AvgHistPE <= 0 {
			return notApplicable(m, "Cần lịch sử P/E 5–7 năm")
		}
		fair = f.EPS * math.Max(f.AvgHistPE-f.StdHistPE, 0)
		if fair <= 0 {
			return notApplicable(m, "Biên P/E quá rộng")
		}
	default:
		return notApplicable(m, "Phương pháp không xác định")
	}
	return Value{Method: m, Applicable: true, FairValue: fair, BuyPrice: fair * (1 - mos), MOS: mos}
}

// Quality is the gate applied before any method (phase-valuation.md
// decision 6): every formula here flags a value trap as cheap.
type Quality struct {
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

const minROE = 10.0

func CheckQuality(f Fundamentals) Quality {
	switch {
	case f.EPS <= 0:
		return Quality{Reason: "Doanh nghiệp đang lỗ"}
	case f.ROE > 0 && f.ROE < minROE:
		return Quality{Reason: "ROE dưới 10%"}
	}
	return Quality{Passed: true}
}

// Summary is today's buy zone: every method, plus the conservative pick
// (the lowest applicable buy price, decision 6).
type Summary struct {
	Price        float64 `json:"price"`
	Values       []Value `json:"values"`
	Quality      Quality `json:"quality"`
	BuyPrice     float64 `json:"buyPrice,omitempty"`
	BuyMethod    Method  `json:"buyMethod,omitempty"`
	FairValue    float64 `json:"fairValue,omitempty"`
	InBuyZone    bool    `json:"inBuyZone"`
	UpsidePct    float64 `json:"upsidePct,omitempty"` // fair value vs price
	BondYieldPct float64 `json:"bondYieldPct"`
	BondYieldAt  string  `json:"bondYieldAsOf"`
}

// Summarize evaluates every method and snaps buy prices down onto the
// exchange's price grid with roundDown (symbol.FloorToTick in
// production; passed in so this package stays dependency-free).
func Summarize(f Fundamentals, p Params, roundDown func(float64) float64) Summary {
	s := Summary{Price: f.Price, Quality: CheckQuality(f), BondYieldPct: p.BondYieldPct, BondYieldAt: BondYieldAsOf}
	for _, m := range Methods {
		v := Evaluate(m, f, p)
		if v.Applicable {
			v.FairValue = math.Round(v.FairValue)
			v.BuyPrice = roundDown(v.BuyPrice)
			if s.BuyPrice == 0 || v.BuyPrice < s.BuyPrice {
				s.BuyPrice, s.BuyMethod, s.FairValue = v.BuyPrice, m, v.FairValue
			}
		}
		s.Values = append(s.Values, v)
	}
	if s.BuyPrice > 0 && f.Price > 0 {
		s.InBuyZone = f.Price <= s.BuyPrice
		s.UpsidePct = math.Round((s.FairValue/f.Price-1)*1000) / 10
	}
	return s
}
