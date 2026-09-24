package quant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/watchlist"
)

// --- fakes ---------------------------------------------------------------

type fakeSymbols struct{ details map[string]symbol.Detail }

func (f fakeSymbols) Search(_, _ string, _, _ int) ([]symbol.Symbol, int) {
	out := []symbol.Symbol{}
	for _, k := range []string{"AAA", "BBB", "CCC"} {
		if d, ok := f.details[k]; ok {
			out = append(out, d.Symbol)
		}
	}
	return out, len(out)
}

func (f fakeSymbols) Detail(sym string) (symbol.Detail, bool) {
	d, ok := f.details[sym]
	return d, ok
}

// fakeBars: AAA slides (low RSI), BBB climbs (high RSI), CCC is flat.
type fakeBars struct{}

func (fakeBars) GetBars(sym, _ string, _, _ int64) []market.Bar {
	out := make([]market.Bar, 60)
	p := 50_000.0
	for i := range out {
		switch sym {
		case "AAA":
			p -= 300
		case "BBB":
			p += 300
		}
		out[i] = market.Bar{Time: int64(i) * 86400, Open: p, High: p, Low: p, Close: p, Volume: 2_000_000}
	}
	return out
}

type fakeWatchlist struct{}

func (fakeWatchlist) List(string) []watchlist.Item { return []watchlist.Item{{Symbol: "BBB"}} }

type fakePortfolios struct{}

func (fakePortfolios) GetPortfolio(_, id string) (portfolio.Portfolio, bool) {
	return portfolio.Portfolio{ID: id, Kind: portfolio.KindTrading}, true
}
func (fakePortfolios) DefaultPortfolioID(string) string { return "pf_1" }
func (fakePortfolios) Positions(string) []portfolio.Position {
	return []portfolio.Position{{Symbol: "AAA", Quantity: 100, AvgCost: 40_000}}
}

func detail(sym, exchange string, roe float64) symbol.Detail {
	return symbol.Detail{
		Symbol:    symbol.Symbol{Symbol: sym, Exchange: exchange, Sector: "Test"},
		LastPrice: 40_000, ROE: roe, PERatio: 10, MarketCap: 50_000_000_000_000,
	}
}

func testSymbols() fakeSymbols {
	return fakeSymbols{details: map[string]symbol.Detail{
		"AAA": detail("AAA", "HOSE", 20),
		"BBB": detail("BBB", "HOSE", 25),
		"CCC": detail("CCC", "HNX", 30),
	}}
}

// scriptedClient returns a fixed completion and records what it was sent.
type scriptedClient struct {
	text string
	got  CompletionRequest
}

func (c *scriptedClient) Complete(_ context.Context, _ Connection, req CompletionRequest) (Completion, error) {
	c.got = req
	return Completion{Text: c.text, Usage: Usage{InputTokens: 10, OutputTokens: 5}, Model: "scripted"}, nil
}
func (c *scriptedClient) ListModels(context.Context, Connection) ([]string, error) {
	return []string{"m1"}, nil
}

func newTestService(client ModelClient) *Service {
	return NewService(map[string]ModelClient{ProviderClaude: client, ProviderCustom: client},
		testSymbols(), fakeBars{}, fakeWatchlist{}, fakePortfolios{})
}

// --- validation ------------------------------------------------------------

func TestValidatePlanDropsWhatCannotRun(t *testing.T) {
	universe, _ := testSymbols().Search("", "", 1, 100)
	in := Plan{
		Reply: "ok",
		Screen: ScreenSpec{Exchange: "NYSE", Conditions: []Condition{
			{Field: "rsi14", Op: "<", Value: 35, Period: 9},
			{Field: "sentiment", Op: ">", Value: 1},
			{Field: "rsi14", Op: "==", Value: 1},
			{Field: "rsi14", Op: ">", Value: 150},
			{Field: "price_vs_sma", Op: ">", Value: 7, Period: 200},
			{Field: "price_vs_sma", Op: ">", Period: 1000},
		}},
		Strategy: StrategySpec{Kind: "rsi_reversion", Symbol: "zzz", RSIEntry: 35, RSIExit: 70, StopLossPercent: 7, Fast: 5},
	}
	out, dropped := validatePlan(in, universe)
	if out.Screen.Exchange != "ALL" {
		t.Errorf("unknown exchange should become ALL, got %q", out.Screen.Exchange)
	}
	if len(out.Screen.Conditions) != 2 {
		t.Fatalf("want 2 runnable conditions, got %+v", out.Screen.Conditions)
	}
	if out.Screen.Conditions[0].Period != 0 || out.Screen.Conditions[1].Value != 0 {
		t.Errorf("unused period/value must be zeroed: %+v", out.Screen.Conditions)
	}
	if out.Strategy.Symbol != "" || out.Strategy.Fast != 0 {
		t.Errorf("unknown symbol and unrelated params must be cleared: %+v", out.Strategy)
	}
	if len(dropped) != 6 { // exchange, 4 conditions, symbol
		t.Errorf("want 6 dropped notes, got %d: %v", len(dropped), dropped)
	}
}

func TestValidatePlanRejectsBadStrategy(t *testing.T) {
	out, dropped := validatePlan(Plan{Strategy: StrategySpec{Kind: "rsi_reversion", RSIEntry: 70, RSIExit: 30}}, nil)
	if out.Strategy.Kind != "none" || len(dropped) != 1 {
		t.Fatalf("exit below entry must drop the strategy, got %+v %v", out.Strategy, dropped)
	}
	out, _ = validatePlan(Plan{Strategy: StrategySpec{Kind: "martingale"}}, nil)
	if out.Strategy.Kind != "none" {
		t.Fatalf("unknown kind must become none")
	}
}

func TestConditionLabels(t *testing.T) {
	got := conditionLabels(ScreenSpec{Exchange: "HOSE", Conditions: []Condition{
		{Field: "rsi14", Op: "<", Value: 35},
		{Field: "roe", Op: ">", Value: 15},
		{Field: "avgVolume20", Op: ">", Value: 1_000_000},
		{Field: "price_vs_sma", Op: ">", Period: 200},
		{Field: "marketCap", Op: ">", Value: 100000},
	}})
	want := []string{"sàn = HOSE", "RSI(14) < 35", "ROE > 15%", "KLTB20 > 1 tr", "giá > SMA(200)", "Vốn hoá > 100.000 tỷ ₫"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("labels\n got %v\nwant %v", got, want)
	}
}

// --- screen + chat -----------------------------------------------------------

func TestChatRunsScreenOnRealDataNotModelOutput(t *testing.T) {
	client := &scriptedClient{text: `{"reply":"Đã lọc.","screen":{"exchange":"HOSE","conditions":[` +
		`{"field":"rsi14","op":"<","value":35,"period":0},{"field":"roe","op":">","value":15,"period":0}]},` +
		`"strategy":{"kind":"none","name":"","symbol":"","fast":0,"slow":0,"rsiEntry":0,"rsiExit":0,"stopLossPercent":0,"trendSma":0}}`}
	svc := newTestService(client)
	resp, err := svc.Chat(context.Background(), "u1", ChatRequest{
		Connection: Connection{Provider: ProviderClaude, APIKey: "k", Model: "m"},
		Messages:   []ChatMessage{{Role: "user", Content: "lọc"}},
		Scope:      DataScope{Prices: true, Indicators: true, Watchlist: true, Positions: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Screen == nil || resp.Screen.Universe != 2 {
		t.Fatalf("HOSE universe should be 2 symbols, got %+v", resp.Screen)
	}
	if len(resp.Screen.Matches) != 1 || resp.Screen.Matches[0].Symbol != "AAA" {
		t.Fatalf("only the sliding HOSE stock has RSI < 35, got %+v", resp.Screen.Matches)
	}
	if resp.Strategy != nil {
		t.Fatalf("kind none must not produce a strategy")
	}
	last := client.got.Messages[len(client.got.Messages)-1].Content
	for _, want := range []string{"<app_data>", "AAA", "RSI14", "User's watchlist: BBB", "Paper position: AAA 100 shares"} {
		if !strings.Contains(last, want) {
			t.Errorf("context block missing %q:\n%s", want, last)
		}
	}
}

func TestChatSendsNoContextWhenScopeIsOff(t *testing.T) {
	client := &scriptedClient{text: `{"reply":"MACD là...","screen":{"exchange":"ALL","conditions":[]},"strategy":{"kind":"none","name":"","symbol":"","fast":0,"slow":0,"rsiEntry":0,"rsiExit":0,"stopLossPercent":0,"trendSma":0}}`}
	svc := newTestService(client)
	resp, err := svc.Chat(context.Background(), "u1", ChatRequest{
		Connection: Connection{Provider: ProviderClaude, APIKey: "k", Model: "m"},
		Messages:   []ChatMessage{{Role: "user", Content: "Giải thích MACD"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := client.got.Messages[0].Content; got != "Giải thích MACD" {
		t.Fatalf("no data may be sent with every toggle off, got %q", got)
	}
	if resp.Screen != nil {
		t.Fatalf("no conditions -> no screen")
	}
}

func TestTestScoresSampleQuestion(t *testing.T) {
	client := &scriptedClient{text: "Sure:\n```json\n" + `{"reply":"ok","screen":{"exchange":"HOSE","conditions":[` +
		`{"field":"rsi14","op":"<","value":35,"period":0},{"field":"roe","op":">","value":15,"period":0}]},` +
		`"strategy":{"kind":"none","name":"","symbol":"","fast":0,"slow":0,"rsiEntry":0,"rsiExit":0,"stopLossPercent":0,"trendSma":0}}` + "\n```"}
	svc := newTestService(client)
	resp, err := svc.Test(context.Background(), TestRequest{Connection{Provider: ProviderCustom, BaseURL: "http://x", Model: "m"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Understood != 3 || resp.Expected != 3 || len(resp.Models) != 1 {
		t.Fatalf("want 3/3 understood and the model list, got %+v", resp)
	}
}

func TestMissingKeyIsRejectedBeforeAnyCall(t *testing.T) {
	client := &scriptedClient{}
	svc := newTestService(client)
	_, err := svc.Test(context.Background(), TestRequest{Connection{Provider: ProviderClaude, Model: "m"}})
	if !errors.Is(err, ErrMissingKey) || client.got.System != "" {
		t.Fatalf("want ErrMissingKey and no model call, got %v", err)
	}
}

// --- SSRF guard --------------------------------------------------------------

func TestBlockedIP(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "192.168.1.1", "172.16.0.5", "169.254.169.254", "::1", "0.0.0.0", "fd00::1"} {
		if !blockedIP(net.ParseIP(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "2606:4700::1111"} {
		if blockedIP(net.ParseIP(s)) {
			t.Errorf("%s should be allowed", s)
		}
	}
}

func TestValidateBaseURL(t *testing.T) {
	ctx := context.Background()
	for _, raw := range []string{"http://127.0.0.1:11434/v1", "http://localhost/v1", "ftp://example.com", "http://user:pw@example.com", "not a url"} {
		if _, err := validateBaseURL(ctx, raw, false); !errors.Is(err, ErrEndpointNotAllowed) {
			t.Errorf("%q should be rejected, got %v", raw, err)
		}
	}
	if _, err := validateBaseURL(ctx, "http://127.0.0.1:11434/v1", true); err != nil {
		t.Errorf("private endpoints must be allowed when opted in: %v", err)
	}
}

func TestGuardedClientRefusesPrivateDial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()
	oa := &openAIClient{allowPrivate: false}
	_, err := oa.Complete(context.Background(), Connection{Provider: ProviderCustom, BaseURL: srv.URL, Model: "m"}, CompletionRequest{})
	if !errors.Is(err, ErrEndpointNotAllowed) {
		t.Fatalf("a loopback endpoint must be refused, got %v", err)
	}
}

// --- adapters against fake servers -------------------------------------------

func TestClaudeAdapterUsesOnlyTheUserKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "server-key-must-not-be-used")
	var body map[string]any
	var headers http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5",`+
			`"content":[{"type":"text","text":"{\"reply\":\"hi\"}"}],"stop_reason":"end_turn","stop_sequence":null,`+
			`"usage":{"input_tokens":123,"output_tokens":45}}`)
	}))
	defer srv.Close()

	temp := 0.2
	c := &claudeClient{baseURL: srv.URL}
	comp, err := c.Complete(context.Background(),
		Connection{Provider: ProviderClaude, APIKey: "user-key", Model: "claude-opus-5", Temperature: &temp},
		CompletionRequest{System: "sys", Messages: []ChatMessage{{Role: "user", Content: "q"}}, Schema: planSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if headers.Get("X-Api-Key") != "user-key" {
		t.Fatalf("must send the user's key, got %q", headers.Get("X-Api-Key"))
	}
	if _, ok := body["temperature"]; ok || comp.TemperatureApplied {
		t.Fatalf("claude-opus-5 rejects temperature; it must not be sent")
	}
	oc, _ := body["output_config"].(map[string]any)
	if f, _ := oc["format"].(map[string]any); f["type"] != "json_schema" {
		t.Fatalf("structured output format missing: %v", body["output_config"])
	}
	if body["fallbacks"] != "default" || !strings.Contains(headers.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Fatalf("claude-opus-5 should opt into refusal fallbacks: %v / %v", body["fallbacks"], headers.Get("Anthropic-Beta"))
	}
	if comp.Text != `{"reply":"hi"}` || comp.Usage.InputTokens != 123 || comp.Usage.OutputTokens != 45 {
		t.Fatalf("unexpected completion %+v", comp)
	}
}

func TestClaudeAdapterMapsProviderErrorsAndRefusals(t *testing.T) {
	status := http.StatusUnauthorized
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"id":"m","type":"message","role":"assistant","model":"claude-opus-5","content":[],`+
			`"stop_reason":"refusal","stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":0}}`)
	}))
	defer srv.Close()
	c := &claudeClient{baseURL: srv.URL}
	conn := Connection{Provider: ProviderClaude, APIKey: "bad", Model: "claude-opus-5"}

	_, err := c.Complete(context.Background(), conn, CompletionRequest{Messages: []ChatMessage{{Role: "user", Content: "q"}}, Schema: planSchema()})
	var pe *ProviderError
	if !errors.As(err, &pe) || pe.Status != 401 || pe.Message != "invalid x-api-key" {
		t.Fatalf("want ProviderError 401 'invalid x-api-key', got %v", err)
	}
	status = http.StatusOK
	_, err = c.Complete(context.Background(), conn, CompletionRequest{Messages: []ChatMessage{{Role: "user", Content: "q"}}, Schema: planSchema()})
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("want ErrRefused, got %v", err)
	}
}

func TestOpenAIAdapterStrictSchemaAndTemperatureRetry(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-user" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		bodies = append(bodies, b)
		if _, ok := b["temperature"]; ok {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"Unsupported parameter: 'temperature' is not supported with this model."}}`)
			return
		}
		_, _ = io.WriteString(w, `{"model":"gpt-x","choices":[{"message":{"content":"{}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":3}}`)
	}))
	defer srv.Close()

	temp := 0.5
	oa := &openAIClient{openAIBase: srv.URL}
	comp, err := oa.Complete(context.Background(),
		Connection{Provider: ProviderOpenAI, APIKey: "sk-user", Model: "gpt-x", Temperature: &temp},
		CompletionRequest{System: "s", Messages: []ChatMessage{{Role: "user", Content: "q"}}, Schema: planSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || comp.TemperatureApplied {
		t.Fatalf("want one retry without temperature, got %d calls, applied=%v", len(bodies), comp.TemperatureApplied)
	}
	rf, _ := bodies[1]["response_format"].(map[string]any)
	if rf["type"] != "json_schema" {
		t.Fatalf("OpenAI must request strict json_schema output, got %v", bodies[1]["response_format"])
	}
	if comp.Usage.InputTokens != 7 || comp.Model != "gpt-x" {
		t.Fatalf("unexpected completion %+v", comp)
	}
}

func TestCustomEndpointNoResponseFormatAndModelList(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"llama-local"},{"id":"qwen"}]}`)
		case "/v1/chat/completions":
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{}"}}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oa := &openAIClient{allowPrivate: true}
	conn := Connection{Provider: ProviderCustom, BaseURL: srv.URL + "/v1/", Model: "llama-local"}
	if _, err := oa.Complete(context.Background(), conn, CompletionRequest{Schema: planSchema()}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["response_format"]; ok {
		t.Fatalf("self-hosted endpoints must not get response_format")
	}
	models, err := oa.ListModels(context.Background(), conn)
	if err != nil || strings.Join(models, ",") != "llama-local,qwen" {
		t.Fatalf("model list: %v %v", models, err)
	}
}

func TestRequestTimeoutClamp(t *testing.T) {
	if requestTimeout(Connection{}) != defaultTimeout ||
		requestTimeout(Connection{TimeoutSeconds: 1}) != minTimeout ||
		requestTimeout(Connection{TimeoutSeconds: 999}) != maxTimeout ||
		requestTimeout(Connection{TimeoutSeconds: 30}) != 30*time.Second {
		t.Fatal("timeout clamp is wrong")
	}
}
