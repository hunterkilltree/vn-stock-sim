package quant

import "sort"

// Providers the user can bring a key for (phase-h.md). "custom" is any
// OpenAI-compatible endpoint the user runs themselves.
const (
	ProviderClaude = "claude"
	ProviderOpenAI = "openai"
	ProviderCustom = "custom"
)

// Connection is the per-request model configuration. The API key arrives
// with every request and is never stored or logged (phase-h.md ground
// rules).
type Connection struct {
	Provider       string   `json:"provider" binding:"required,oneof=claude openai custom"`
	APIKey         string   `json:"apiKey" binding:"max=500"`
	BaseURL        string   `json:"baseUrl" binding:"max=500"`
	Model          string   `json:"model" binding:"required,max=200"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
	Temperature    *float64 `json:"temperature"`
}

type ChatMessage struct {
	Role    string `json:"role" binding:"required,oneof=user assistant"`
	Content string `json:"content" binding:"required,max=8000"`
}

// DataScope mirrors Settings-AI.dc.html's "Dữ liệu gửi cho mô hình"
// toggles; each optional context block is only built when its flag is on.
type DataScope struct {
	Prices     bool `json:"prices"`
	Indicators bool `json:"indicators"`
	Watchlist  bool `json:"watchlist"`
	Positions  bool `json:"positions"`
}

type ChatRequest struct {
	Connection
	Messages    []ChatMessage `json:"messages" binding:"required,min=1,max=20,dive"`
	Scope       DataScope     `json:"scope"`
	PortfolioID string        `json:"portfolioId"`
	// DefaultExchange is Quant.dc.html's "Sàn: …" chip under the input:
	// used when the question doesn't name an exchange.
	DefaultExchange string `json:"defaultExchange" binding:"omitempty,oneof=ALL HOSE HNX UPCOM"`
}

type TestRequest struct {
	Connection
}

// Plan is the one JSON object the model must return. No field is
// nullable -- "nothing" is an empty condition list or strategy kind
// "none" -- so a single schema works across providers (phase-h.md
// decision 1).
type Plan struct {
	Reply    string       `json:"reply"`
	Screen   ScreenSpec   `json:"screen"`
	Strategy StrategySpec `json:"strategy"`
}

type ScreenSpec struct {
	Exchange   string      `json:"exchange"`
	Conditions []Condition `json:"conditions"`
}

// Condition is `field op value`, except field "price_vs_sma", which means
// `price op SMA(period)` (value unused). Units: price in thousand VND,
// marketCap in billion VND, avgVolume20 in shares, percentages as percent.
type Condition struct {
	Field  string  `json:"field"`
	Op     string  `json:"op"`
	Value  float64 `json:"value"`
	Period int     `json:"period"`
}

type StrategySpec struct {
	Kind            string  `json:"kind"`
	Name            string  `json:"name"`
	Symbol          string  `json:"symbol"`
	Fast            int     `json:"fast"`
	Slow            int     `json:"slow"`
	RSIEntry        float64 `json:"rsiEntry"`
	RSIExit         float64 `json:"rsiExit"`
	StopLossPercent float64 `json:"stopLossPercent"`
	TrendSMA        int     `json:"trendSma"`
}

var conditionFields = []string{
	"price", "changePercent", "rsi14", "roe", "pe", "pb", "eps",
	"dividendYield", "marketCap", "avgVolume20", "price_vs_sma",
}

var conditionOps = []string{"<", "<=", ">", ">="}

// planSchema is the JSON Schema both structured-output providers enforce.
// Every object lists all its properties as required and forbids extras,
// which strict mode on both Claude and OpenAI requires.
func planSchema() map[string]any {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	integer := map[string]any{"type": "integer"}
	obj := func(props map[string]any) map[string]any {
		required := make([]string, 0, len(props))
		for k := range props {
			required = append(required, k)
		}
		sort.Strings(required)
		return map[string]any{
			"type":                 "object",
			"properties":           props,
			"required":             required,
			"additionalProperties": false,
		}
	}
	condition := obj(map[string]any{
		"field":  map[string]any{"type": "string", "enum": conditionFields},
		"op":     map[string]any{"type": "string", "enum": conditionOps},
		"value":  num,
		"period": integer,
	})
	return obj(map[string]any{
		"reply": str,
		"screen": obj(map[string]any{
			"exchange":   map[string]any{"type": "string", "enum": []string{"ALL", "HOSE", "HNX", "UPCOM"}},
			"conditions": map[string]any{"type": "array", "items": condition},
		}),
		"strategy": obj(map[string]any{
			"kind":            map[string]any{"type": "string", "enum": []string{"none", "ema_crossover", "rsi_reversion"}},
			"name":            str,
			"symbol":          str,
			"fast":            integer,
			"slow":            integer,
			"rsiEntry":        num,
			"rsiExit":         num,
			"stopLossPercent": num,
			"trendSma":        integer,
		}),
	})
}

// Row is one symbol's computed metrics in a screen result.
type Row struct {
	Symbol        string  `json:"symbol"`
	Exchange      string  `json:"exchange"`
	Sector        string  `json:"sector"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"changePercent"`
	RSI14         float64 `json:"rsi14"`
	ROE           float64 `json:"roe"`
	PE            float64 `json:"pe"`
	PB            float64 `json:"pb"`
	EPS           float64 `json:"eps"`
	DividendYield float64 `json:"dividendYield"`
	MarketCap     float64 `json:"marketCap"`
	AvgVolume20   float64 `json:"avgVolume20"`
}

type ScreenOutcome struct {
	Exchange   string      `json:"exchange"`
	Conditions []Condition `json:"conditions"`
	Labels     []string    `json:"labels"`
	Universe   int         `json:"universe"`
	Matches    []Row       `json:"matches"`
	SortedBy   string      `json:"sortedBy"`
}

type Usage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

type ChatResponse struct {
	Reply    string         `json:"reply"`
	Screen   *ScreenOutcome `json:"screen"`
	Strategy *StrategySpec  `json:"strategy"`
	// Dropped lists parts of the model's output the backend refused to
	// run (unknown field, out-of-range value...), so the UI can say so.
	Dropped []string `json:"dropped"`
	// Plan is the validated plan, echoed so the client can send it back
	// as the assistant turn when the user refines the request.
	Plan               Plan   `json:"plan"`
	Usage              Usage  `json:"usage"`
	LatencyMs          int64  `json:"latencyMs"`
	Model              string `json:"model"`
	TemperatureApplied bool   `json:"temperatureApplied"`
}

type TestResponse struct {
	OK                 bool     `json:"ok"`
	LatencyMs          int64    `json:"latencyMs"`
	Usage              Usage    `json:"usage"`
	Model              string   `json:"model"`
	SampleQuestion     string   `json:"sampleQuestion"`
	Understood         int      `json:"understood"`
	Expected           int      `json:"expected"`
	Labels             []string `json:"labels"`
	TemperatureApplied bool     `json:"temperatureApplied"`
	Models             []string `json:"models"`
	ModelsError        string   `json:"modelsError,omitempty"`
}
