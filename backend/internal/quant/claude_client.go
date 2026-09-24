package quant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// claudeClient calls the Claude API through the official Go SDK with the
// user's own key (phase-h.md decision 1). baseURL is only set by tests.
type claudeClient struct {
	baseURL string
}

func (c *claudeClient) sdk(conn Connection) anthropic.Client {
	opts := []option.RequestOption{
		// Never fall back to server-side credentials (ANTHROPIC_API_KEY,
		// ant login profiles): every call must be billed to the key the
		// user supplied, or fail.
		option.WithoutEnvironmentDefaults(),
		option.WithAPIKey(conn.APIKey),
		option.WithMaxRetries(1),
	}
	if c.baseURL != "" {
		opts = append(opts, option.WithBaseURL(c.baseURL))
	}
	return anthropic.NewClient(opts...)
}

// claudeAcceptsTemperature: Claude Opus 5, Sonnet 5, Opus 4.7/4.8 and
// Fable reject sampling parameters with a 400; the 4.6-and-older
// generation (incl. Haiku 4.5) still accepts them (phase-h.md decision 4).
func claudeAcceptsTemperature(model string) bool {
	if strings.HasPrefix(model, "claude-3") {
		return true
	}
	for _, gen := range []string{"-4-6", "-4-5", "-4-1", "-4-0", "-4-2"} {
		if strings.Contains(model, gen) {
			return true
		}
	}
	return model == "claude-opus-4" || model == "claude-sonnet-4"
}

// usesRefusalFallback opts the models that can decline on safety grounds
// into server-side refusal fallbacks (phase-h.md decision 6).
func usesRefusalFallback(model string) bool {
	return model == "claude-opus-5" || strings.HasPrefix(model, "claude-fable-")
}

func (c *claudeClient) Complete(ctx context.Context, conn Connection, req CompletionRequest) (Completion, error) {
	msgs := make([]anthropic.BetaMessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		block := anthropic.NewBetaTextBlock(m.Content)
		if m.Role == "assistant" {
			msgs = append(msgs, anthropic.BetaMessageParam{
				Role:    anthropic.BetaMessageParamRoleAssistant,
				Content: []anthropic.BetaContentBlockParamUnion{block},
			})
		} else {
			msgs = append(msgs, anthropic.NewBetaUserMessage(block))
		}
	}

	params := anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(conn.Model),
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: req.System}},
		Messages:  msgs,
		OutputConfig: anthropic.BetaOutputConfigParam{
			Format: anthropic.BetaJSONOutputFormatParam{Schema: req.Schema},
		},
	}
	if usesRefusalFallback(conn.Model) {
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
		params.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
	}
	tempApplied := false
	if conn.Temperature != nil && claudeAcceptsTemperature(conn.Model) {
		params.Temperature = anthropic.Float(*conn.Temperature)
		tempApplied = true
	}

	client := c.sdk(conn)
	resp, err := client.Beta.Messages.New(ctx, params)
	if err != nil && tempApplied && isTemperatureRejection(err) {
		params.Temperature = param.Opt[float64]{}
		tempApplied = false
		resp, err = client.Beta.Messages.New(ctx, params)
	}
	if err != nil {
		return Completion{}, claudeError(err)
	}

	switch resp.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return Completion{}, ErrRefused
	case anthropic.BetaStopReasonMaxTokens:
		return Completion{}, ErrInvalidOutput
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	return Completion{
		Text:               text.String(),
		Usage:              Usage{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens},
		Model:              string(resp.Model),
		TemperatureApplied: tempApplied,
	}, nil
}

func (c *claudeClient) ListModels(ctx context.Context, conn Connection) ([]string, error) {
	client := c.sdk(conn)
	page, err := client.Models.List(ctx, anthropic.ModelListParams{Limit: anthropic.Int(100)})
	if err != nil {
		return nil, claudeError(err)
	}
	ids := make([]string, 0, len(page.Data))
	for _, m := range page.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

func isTemperatureRejection(err error) bool {
	var apiErr *anthropic.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == 400 &&
		strings.Contains(strings.ToLower(apiErr.RawJSON()), "temperature")
}

// claudeError maps SDK errors to the service's provider-neutral ones,
// keeping the provider's own message (never the request, which holds the
// key).
func claudeError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		msg := apiErr.Type()
		var body struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(apiErr.RawJSON()), &body) == nil && body.Error.Message != "" {
			return &ProviderError{Status: apiErr.StatusCode, Message: body.Error.Message}
		}
		return &ProviderError{Status: apiErr.StatusCode, Message: string(msg)}
	}
	return err
}
