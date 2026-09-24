package quant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openAIClient speaks the Chat Completions wire format: OpenAI itself
// (fixed base URL, strict json_schema output) and any self-hosted
// OpenAI-compatible server (user-supplied base URL behind the SSRF guard,
// no response_format since support varies) -- phase-h.md decisions 1, 11.
type openAIClient struct {
	openAIBase   string
	allowPrivate bool
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *openAIClient) target(ctx context.Context, conn Connection) (string, *http.Client, error) {
	if conn.Provider == ProviderOpenAI {
		return strings.TrimRight(c.openAIBase, "/"), &http.Client{}, nil
	}
	u, err := validateBaseURL(ctx, strings.TrimSpace(conn.BaseURL), c.allowPrivate)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimRight(u.String(), "/"), guardedClient(u, 0, c.allowPrivate), nil
}

func (c *openAIClient) do(ctx context.Context, client *http.Client, method, url, key string, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, 0, ErrTimeout
		}
		if errors.Is(err, ErrEndpointNotAllowed) {
			return nil, 0, ErrEndpointNotAllowed
		}
		return nil, 0, fmt.Errorf("could not reach the model endpoint")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return data, resp.StatusCode, err
}

func oaErrorMessage(data []byte) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}
	s := strings.TrimSpace(string(data))
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}

func (c *openAIClient) Complete(ctx context.Context, conn Connection, req CompletionRequest) (Completion, error) {
	base, client, err := c.target(ctx, conn)
	if err != nil {
		return Completion{}, err
	}
	msgs := []oaMessage{{Role: "system", Content: req.System}}
	for _, m := range req.Messages {
		msgs = append(msgs, oaMessage{Role: m.Role, Content: m.Content})
	}
	body := map[string]any{"model": conn.Model, "messages": msgs}
	if conn.Provider == ProviderOpenAI {
		body["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": "quant_plan", "strict": true, "schema": req.Schema},
		}
	}
	tempApplied := conn.Temperature != nil
	if tempApplied {
		body["temperature"] = *conn.Temperature
	}

	data, status, err := c.do(ctx, client, http.MethodPost, base+"/chat/completions", conn.APIKey, body)
	// Reasoning models reject sampling parameters; retry once without it
	// and report that the setting was ignored (phase-h.md decision 4).
	if err == nil && status == http.StatusBadRequest && tempApplied &&
		strings.Contains(strings.ToLower(oaErrorMessage(data)), "temperature") {
		delete(body, "temperature")
		tempApplied = false
		data, status, err = c.do(ctx, client, http.MethodPost, base+"/chat/completions", conn.APIKey, body)
	}
	if err != nil {
		return Completion{}, err
	}
	if status < 200 || status > 299 {
		return Completion{}, &ProviderError{Status: status, Message: oaErrorMessage(data)}
	}

	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &out); err != nil || len(out.Choices) == 0 {
		return Completion{}, ErrInvalidOutput
	}
	choice := out.Choices[0]
	if choice.Message.Refusal != "" {
		return Completion{}, ErrRefused
	}
	model := out.Model
	if model == "" {
		model = conn.Model
	}
	return Completion{
		Text:               choice.Message.Content,
		Usage:              Usage{InputTokens: out.Usage.PromptTokens, OutputTokens: out.Usage.CompletionTokens},
		Model:              model,
		TemperatureApplied: tempApplied,
	}, nil
}

func (c *openAIClient) ListModels(ctx context.Context, conn Connection) ([]string, error) {
	base, client, err := c.target(ctx, conn)
	if err != nil {
		return nil, err
	}
	data, status, err := c.do(ctx, client, http.MethodGet, base+"/models", conn.APIKey, nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status > 299 {
		return nil, &ProviderError{Status: status, Message: oaErrorMessage(data)}
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, ErrInvalidOutput
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}
