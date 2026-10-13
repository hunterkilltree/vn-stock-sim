package luk

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

// series builds daily bars from a close path; each bar's range is
// rangePct wide around the close, volume vol (or volAt(i) when set).
func series(closes []float64, rangePct float64, vol func(i int) int64) []market.Bar {
	out := make([]market.Bar, len(closes))
	for i, c := range closes {
		h := c * (1 + rangePct/200)
		l := c * (1 - rangePct/200)
		out[i] = market.Bar{Time: int64(i) * 86400, Open: c, High: h, Low: l, Close: c, Volume: vol(i)}
	}
	return out
}

func trendPath(n int, start, dailyPct float64) []float64 {
	out := make([]float64, n)
	p := start
	for i := range out {
		out[i] = p
		p *= 1 + dailyPct/100
	}
	return out
}

func constVol(v int64) func(int) int64 { return func(int) int64 { return v } }

func check(t *testing.T, r Result, key string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.Key == key {
			return c
		}
	}
	t.Fatalf("no check %q", key)
	return Check{}
}

// leaderWithPullback: a strong 0.8%/day uptrend that pulls back to the
// 9/21 EMA over the last few sessions on drying volume.
func leaderWithPullback() []market.Bar {
	closes := trendPath(150, 20000, 0.8)
	top := closes[len(closes)-1]
	for i := 0; i < 4; i++ {
		closes = append(closes, top*(1-0.012*float64(i+1)))
	}
	bars := series(closes, 4.4, func(i int) int64 {
		if i >= 150 {
			return 600_000
		}
		return 1_000_000
	})
	// A tight last session, the kind of day Luk's stop-under-the-low suits.
	last := &bars[len(bars)-1]
	last.High, last.Low = last.Close*1.01, last.Close*0.99
	return bars
}

func TestLeaderPullbackIsBuy(t *testing.T) {
	idx := series(trendPath(154, 1200, 0.1), 1.2, constVol(1))
	r := Compute(Input{Symbol: "AAA", Bars: leaderWithPullback(), IndexBars: idx})
	for _, c := range r.Checks {
		if c.Core && c.Status != StatusPass {
			t.Errorf("core check %s = %s (%s, %s)", c.Key, c.Status, c.Value, c.Detail)
		}
	}
	if r.Verdict != VerdictBuy {
		for _, c := range r.Checks {
			t.Logf("%s %s %s", c.Key, c.Status, c.Value)
		}
		t.Fatalf("verdict = %s, want buy", r.Verdict)
	}
	if r.Plan == nil || r.Plan.Stop >= r.Plan.Entry || r.Plan.PositionPct > maxPositionPct {
		t.Fatalf("bad plan %+v", r.Plan)
	}
	want := math.Min(maxPositionPct, riskPct/r.Plan.StopPct*100)
	if math.Abs(r.Plan.PositionPct-round2(want)) > 0.01 {
		t.Errorf("position %v, want %v", r.Plan.PositionPct, want)
	}
}

func TestDowntrendIsAvoid(t *testing.T) {
	bars := series(trendPath(150, 30000, -0.5), 3, constVol(1_000_000))
	idx := series(trendPath(150, 1200, 0.1), 1.2, constVol(1))
	r := Compute(Input{Bars: bars, IndexBars: idx})
	if r.Verdict != VerdictAvoid {
		t.Fatalf("verdict = %s, want avoid", r.Verdict)
	}
	if check(t, r, "trend").Status != StatusFail || check(t, r, "rs").Status != StatusFail {
		t.Error("trend and rs should both fail")
	}
}

func TestExtendedLeaderIsWatch(t *testing.T) {
	// Straight up 3%/day: a leader, but far above the 21 EMA.
	bars := series(trendPath(150, 20000, 3), 4, constVol(1_000_000))
	idx := series(trendPath(150, 1200, 0.1), 1.2, constVol(1))
	r := Compute(Input{Bars: bars, IndexBars: idx})
	if got := check(t, r, "pullback").Status; got != StatusWarn {
		t.Fatalf("pullback = %s, want warn", got)
	}
	if r.Verdict != VerdictWatch {
		t.Fatalf("verdict = %s, want watch", r.Verdict)
	}
}

func TestWeakMarketBlocks(t *testing.T) {
	idx := series(trendPath(154, 1300, -0.4), 1.2, constVol(1))
	r := Compute(Input{Bars: leaderWithPullback(), IndexBars: idx})
	if check(t, r, "market").Status != StatusFail || r.Verdict != VerdictAvoid {
		t.Fatalf("market=%s verdict=%s", check(t, r, "market").Status, r.Verdict)
	}
}

func TestShortHistory(t *testing.T) {
	r := Compute(Input{Bars: series(trendPath(10, 100, 1), 2, constVol(1))})
	if r.Verdict != VerdictAvoid || r.Plan != nil {
		t.Fatalf("got %+v", r)
	}
}

func TestADRAndAVWAP(t *testing.T) {
	bars := series([]float64{100, 100, 100}, 4, constVol(10))
	if got := adrPct(bars, 20); math.Abs(got-4.08) > 0.01 {
		t.Errorf("adr = %v", got)
	}
	if got := avwapFromSwingLow(bars, 63); math.Abs(got-100) > 1e-9 {
		t.Errorf("avwap = %v", got)
	}
}

func TestGeminiReviewerRequest(t *testing.T) {
	var gotKey, gotPath string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotPath = r.Header.Get("x-goog-api-key"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"- Mua"}]}}]}`))
	}))
	defer srv.Close()
	g := NewGeminiReviewer("k123", "gemini-test")
	g.base = srv.URL
	text, err := g.Review(context.Background(), "sys", "prompt")
	if err != nil || text != "- Mua" {
		t.Fatalf("text=%q err=%v", text, err)
	}
	if gotKey != "k123" || gotPath != "/models/gemini-test:generateContent" {
		t.Errorf("key=%q path=%q", gotKey, gotPath)
	}
	if _, ok := body["systemInstruction"]; !ok {
		t.Error("missing systemInstruction")
	}
}

func TestGeminiReviewerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"API key not valid"}}`))
	}))
	defer srv.Close()
	g := NewGeminiReviewer("bad", "m")
	g.base = srv.URL
	_, err := g.Review(context.Background(), "s", "p")
	if err == nil || !strings.Contains(err.Error(), "API key not valid") {
		t.Fatalf("err = %v", err)
	}
	if NewGeminiReviewer("  ", "m") != nil {
		t.Error("empty key should disable the reviewer")
	}
}

func TestGeminiCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/models/good-model" || r.Header.Get("x-goog-api-key") != "k123" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"models/bad-model is not found"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"name":"models/good-model"}`))
	}))
	defer srv.Close()
	g := NewGeminiReviewer("k123", "good-model")
	g.base = srv.URL
	if err := g.Check(context.Background()); err != nil {
		t.Fatalf("good setup: %v", err)
	}
	g.model = "bad-model"
	if err := g.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "is not found") {
		t.Fatalf("bad model: %v", err)
	}
}

func TestGeminiUsageAndEmptyText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[]},"finishReason":"MAX_TOKENS"}],"usageMetadata":{"promptTokenCount":12,"totalTokenCount":4108}}`))
	}))
	defer srv.Close()
	g := NewGeminiReviewer("k", "m")
	g.base = srv.URL
	_, usage, err := g.generate(context.Background(), "s", "p")
	if err == nil || !strings.Contains(err.Error(), "finish=MAX_TOKENS") {
		t.Fatalf("err = %v", err)
	}
	if usage.PromptTokenCount != 12 || usage.TotalTokenCount != 4108 {
		t.Errorf("usage = %+v", usage)
	}
}

func TestMaskKey(t *testing.T) {
	if got := maskKey("AIzaSyABCDEFGH1234"); got != "AIza…1234 (18 chars)" {
		t.Errorf("got %q", got)
	}
	if got := maskKey("short"); strings.Contains(got, "short") {
		t.Errorf("short key leaked: %q", got)
	}
}
