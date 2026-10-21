// Package riskrules checks an order against the user's risk-management
// rules before it is booked (phase-risk-rules.md design item 1). Pure
// functions, no I/O: order.Service calls Check before every buy, and the
// ticket calls the same Check through a dry-run endpoint, so the
// warnings the user sees before clicking Mua are the ones the server
// applies.
//
// Slice 1 (`claude/risk-trade-plan-sizing`) covers rules 1-3: a plan with
// a stop on every buy, size from risk, and reward-to-risk.
package riskrules

import (
	"fmt"
	"math"
	"strings"
)

// Mode is how a portfolio enforces the rules. Coach warns and records
// the violation on the order; strict rejects an order with a blocking
// finding.
type Mode string

const (
	Coach  Mode = "coach"
	Strict Mode = "strict"
)

// Settings are one portfolio's thresholds (Settings → Tài khoản giấy).
type Settings struct {
	Mode Mode `json:"mode"`
	// RiskPct is the planned risk per trade and MaxRiskPct its hard cap,
	// both in percent of equity (rule 2: 1%, capped at Elder's 2%).
	RiskPct    float64 `json:"riskPercent"`
	MaxRiskPct float64 `json:"maxRiskPercent"`
	// MinRewardRisk is the lowest acceptable (target − entry) ÷ (entry −
	// stop) (rule 3).
	MinRewardRisk float64 `json:"minRewardRisk"`
}

// Defaults: coach mode, so nothing that worked before starts failing
// (phase-risk-rules.md open question 1, the recommended answer).
func Defaults() Settings {
	return Settings{Mode: Coach, RiskPct: 1, MaxRiskPct: 2, MinRewardRisk: 2}
}

// Validate rejects settings the rules can't work with.
func (s Settings) Validate() error {
	switch {
	case s.Mode != Coach && s.Mode != Strict:
		return fmt.Errorf("mode must be %q or %q", Coach, Strict)
	case s.RiskPct <= 0 || s.RiskPct > 10:
		return fmt.Errorf("riskPercent must be in (0, 10]")
	case s.MaxRiskPct < s.RiskPct || s.MaxRiskPct > 10:
		return fmt.Errorf("maxRiskPercent must be between riskPercent and 10")
	case s.MinRewardRisk < 0 || s.MinRewardRisk > 10:
		return fmt.Errorf("minRewardRisk must be in [0, 10]")
	}
	return nil
}

// Intent is the order being checked. Entry is the expected fill price
// (the limit price, or the last price for a market order); Stop and
// Target are the trade plan's, 0 when not set.
type Intent struct {
	Side     string
	Quantity float64
	Entry    float64
	Stop     float64
	Target   float64
	FeeRate  float64
	// Lot is the board lot sizes round down to: 100 for VN stocks, 0 for
	// crypto (any quantity).
	Lot float64
}

// Severity: a warning never stops an order; a block does in strict mode,
// and an invalid plan (Invalid) is refused in both modes.
type Severity string

const (
	Warn    Severity = "warn"
	Block   Severity = "block"
	Invalid Severity = "invalid"
)

// Finding is one rule's verdict, with a Vietnamese message for the UI.
type Finding struct {
	Rule     int      `json:"rule"`
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

// Assessment is Check's answer. RiskAmount is what the position loses if
// the stop fills, fees on both sides included; SuggestedQty is the
// quantity that risks exactly Settings.RiskPct.
type Assessment struct {
	Findings     []Finding `json:"findings"`
	RiskAmount   float64   `json:"riskAmount"`
	RiskPct      float64   `json:"riskPercent"`
	RewardRisk   float64   `json:"rewardRisk"`
	SuggestedQty float64   `json:"suggestedQuantity"`
	// Blocked: the order must not be placed in this mode.
	Blocked bool `json:"blocked"`
}

// Check applies rules 1-3 to a buy. Sells exit a plan and aren't
// checked here (stop management is slice 4).
func Check(in Intent, equity float64, s Settings) Assessment {
	a := Assessment{Findings: []Finding{}}
	if in.Side != "buy" {
		return a
	}
	// A rule's "block" severity only blocks in strict mode.
	enforce := func(sev Severity) Severity {
		if sev == Block && s.Mode != Strict {
			return Warn
		}
		return sev
	}
	add := func(rule int, code string, sev Severity, msg string) {
		sev = enforce(sev)
		a.Findings = append(a.Findings, Finding{Rule: rule, Code: code, Severity: sev, Message: msg})
		if sev != Warn {
			a.Blocked = true
		}
	}

	switch {
	case in.Stop <= 0:
		add(1, "missing_stop", Block, "Chưa có điểm cắt lỗ: hãy đặt điểm thoát trước khi vào lệnh.")
	case in.Stop >= in.Entry:
		add(1, "stop_not_below_entry", Invalid, "Điểm cắt lỗ phải thấp hơn giá mua.")
	default:
		perShare := riskPerShare(in.Entry, in.Stop, in.FeeRate)
		a.RiskAmount = in.Quantity * perShare
		a.SuggestedQty = SizeFor(equity, s.RiskPct, perShare, in.Lot)
		if equity > 0 {
			a.RiskPct = a.RiskAmount / equity * 100
			switch {
			case a.RiskPct > s.MaxRiskPct:
				add(2, "risk_over_max", Block, fmt.Sprintf("Rủi ro %s%% vốn, vượt mức tối đa %s%%.", vn(a.RiskPct, 2), vn(s.MaxRiskPct, 1)))
			case a.RiskPct > s.RiskPct:
				add(2, "risk_over_target", Warn, fmt.Sprintf("Rủi ro %s%% vốn, cao hơn mức dự kiến %s%%.", vn(a.RiskPct, 2), vn(s.RiskPct, 1)))
			}
		}
	}

	switch {
	case in.Target <= 0:
		add(3, "missing_target", Warn, "Chưa có mục tiêu chốt lời: không tính được tỷ lệ lời/lỗ.")
	case in.Target <= in.Entry:
		add(3, "target_not_above_entry", Invalid, "Mục tiêu chốt lời phải cao hơn giá mua.")
	case in.Stop > 0 && in.Stop < in.Entry:
		a.RewardRisk = (in.Target - in.Entry) / (in.Entry - in.Stop)
		if a.RewardRisk < s.MinRewardRisk {
			add(3, "low_reward_risk", Warn, fmt.Sprintf("Tỷ lệ lời/lỗ %s, thấp hơn %s.", vn(a.RewardRisk, 1), vn(s.MinRewardRisk, 1)))
		}
	}
	return a
}

// vn formats a number the Vietnamese way: decimal comma.
func vn(v float64, decimals int) string {
	return strings.Replace(fmt.Sprintf("%.*f", decimals, v), ".", ",", 1)
}

// riskPerShare is what one share loses if bought at entry and sold at
// stop, with the fee on both fills.
func riskPerShare(entry, stop, feeRate float64) float64 {
	return entry - stop + feeRate*(entry+stop)
}

// SizeFor is rule 2's quantity: equity × riskPct ÷ risk per share,
// rounded down to the lot (whole units of 1e-8 when lot is 0).
func SizeFor(equity, riskPct, perShare, lot float64) float64 {
	if equity <= 0 || riskPct <= 0 || perShare <= 0 {
		return 0
	}
	q := equity * riskPct / 100 / perShare
	if lot > 0 {
		return math.Floor(q/lot) * lot
	}
	return math.Floor(q*1e8) / 1e8
}
