package rating

import (
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

// series builds n daily bars whose close moves by step each session, with
// flat volume.
func series(n int, start, step float64) []market.Bar {
	bars := make([]market.Bar, n)
	for i := range bars {
		c := start + step*float64(i)
		bars[i] = market.Bar{Time: int64(i) * 86400, Open: c, High: c, Low: c, Close: c, Volume: 1000}
	}
	return bars
}

func detail(pe, pb, roe, eps, div float64) symbol.Detail {
	return symbol.Detail{
		Symbol:  symbol.Symbol{Symbol: "TST"},
		PERatio: pe, PBRatio: pb, ROE: roe, EPS: eps, DividendYield: div,
	}
}

func verdictOf(t *testing.T, res Result, label string) string {
	t.Helper()
	for _, s := range res.Signals {
		if s.Label == label {
			return s.Verdict
		}
	}
	t.Fatalf("no signal %q in %+v", label, res.Signals)
	return ""
}

func TestStrongBuy(t *testing.T) {
	// Steady uptrend plus cheap, profitable fundamentals.
	res := Compute(detail(8, 0.9, 20, 3000, 5), series(250, 100, 0.5))
	if res.Overall.Rating != RatingStrongBuy {
		t.Fatalf("rating = %s (score %.2f), want strong_buy", res.Overall.Rating, res.Overall.Score)
	}
	if v := verdictOf(t, res, "SMA 50 so với SMA 200"); v != VerdictBuy {
		t.Errorf("golden cross verdict = %s", v)
	}
	// A straight-line rally has RSI 100: overbought, so that rule sells.
	if v := verdictOf(t, res, "RSI 14"); v != VerdictSell {
		t.Errorf("RSI verdict = %s, want sell", v)
	}
}

func TestStrongSell(t *testing.T) {
	// Steady downtrend, loss-making, expensive on book.
	res := Compute(detail(0, 4, 2, -500, 0), series(250, 300, -0.5))
	if res.Overall.Rating != RatingStrongSell {
		t.Fatalf("rating = %s (score %.2f), want strong_sell", res.Overall.Rating, res.Overall.Score)
	}
	if v := verdictOf(t, res, "P/E"); v != VerdictSell {
		t.Errorf("P/E verdict = %s, want sell for negative EPS", v)
	}
}

func TestFlatIsNeutral(t *testing.T) {
	res := Compute(detail(15, 2, 12, 2000, 2), series(250, 100, 0))
	if res.Overall.Rating != RatingNeutral {
		t.Fatalf("rating = %s (score %.2f), want neutral", res.Overall.Rating, res.Overall.Score)
	}
}

func TestNoFundamentalsUsesTechnicalOnly(t *testing.T) {
	res := Compute(detail(0, 0, 0, 0, 0), series(250, 100, 0.5))
	if res.Fundamental != nil {
		t.Fatalf("fundamental = %+v, want nil", res.Fundamental)
	}
	if res.Overall != res.Technical {
		t.Errorf("overall %+v != technical %+v", res.Overall, res.Technical)
	}
}

func TestShortHistorySkipsLongRules(t *testing.T) {
	res := Compute(detail(0, 0, 0, 0, 0), series(30, 100, 0.5))
	for _, s := range res.Signals {
		if s.Label == "Giá so với SMA 200" || s.Label == "SMA 50 so với SMA 200" || s.Label == "MACD (12, 26, 9)" {
			t.Errorf("unexpected signal %q with 30 bars", s.Label)
		}
	}
	if len(res.Signals) == 0 {
		t.Fatal("want some technical signals with 30 bars")
	}
}

func TestVolumeSpikeConfirmsDirection(t *testing.T) {
	bars := series(30, 100, 0)
	bars[len(bars)-1].Close = 105
	bars[len(bars)-1].Volume = 3000
	if v := verdictOf(t, Compute(detail(0, 0, 0, 0, 0), bars), "Khối lượng"); v != VerdictBuy {
		t.Errorf("up day on 3x volume = %s, want buy", v)
	}
	bars[len(bars)-1].Close = 95
	if v := verdictOf(t, Compute(detail(0, 0, 0, 0, 0), bars), "Khối lượng"); v != VerdictSell {
		t.Errorf("down day on 3x volume = %s, want sell", v)
	}
}

func TestRatingCutoffs(t *testing.T) {
	cases := map[float64]string{
		1: RatingStrongBuy, 0.5: RatingStrongBuy, 0.2: RatingBuy, 0: RatingNeutral,
		-0.2: RatingSell, -0.5: RatingStrongSell, -1: RatingStrongSell,
	}
	for score, want := range cases {
		if got := ratingFor(score); got != want {
			t.Errorf("ratingFor(%v) = %s, want %s", score, got, want)
		}
	}
}
