package valuation

import (
	_ "embed"
	"encoding/csv"
	"sort"
	"strconv"
	"strings"
	"time"
)

// bondyield.csv is the VN 10-year government bond yield as a dated
// series, one row per observation, extended by hand
// (phase-fundamentals-data.md "Bond yield"). A backtest uses the row in
// force on its bar date, so the 2008 Graham 1974 value isn't computed at
// today's yield.
//
//go:embed bondyield.csv
var bondYieldCSV string

// BondYield is one observation of the series.
type BondYield struct {
	Date     string // YYYY-MM-DD
	YieldPct float64
}

var bondYields = parseBondYields(bondYieldCSV)

func parseBondYields(src string) []BondYield {
	rows, err := csv.NewReader(strings.NewReader(src)).ReadAll()
	if err != nil {
		panic("valuation: bondyield.csv: " + err.Error())
	}
	var out []BondYield
	for i, r := range rows {
		if i == 0 {
			continue // header
		}
		if _, err := time.Parse(time.DateOnly, r[0]); err != nil {
			panic("valuation: bondyield.csv row " + strconv.Itoa(i) + ": " + err.Error())
		}
		y, err := strconv.ParseFloat(r[1], 64)
		if err != nil || y <= 0 {
			panic("valuation: bondyield.csv row " + strconv.Itoa(i) + ": bad yield " + r[1])
		}
		out = append(out, BondYield{Date: r[0], YieldPct: y})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	if len(out) == 0 {
		panic("valuation: bondyield.csv has no rows")
	}
	return out
}

// BondYieldOn is the latest observation dated on or before t. ok is
// false before the series starts: callers must treat the yield as
// unknown then, not borrow a later one.
func BondYieldOn(t time.Time) (BondYield, bool) {
	day := t.In(vnLoc).Format(time.DateOnly)
	i := sort.Search(len(bondYields), func(i int) bool { return bondYields[i].Date > day })
	if i == 0 {
		return BondYield{}, false
	}
	return bondYields[i-1], true
}

// LatestBondYield is the newest observation, for today's buy zone.
func LatestBondYield() BondYield {
	return bondYields[len(bondYields)-1]
}
