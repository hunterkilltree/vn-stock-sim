package luk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const geminiBase = "https://generativelanguage.googleapis.com/v1beta"

// GeminiReviewer calls Google's Gemini generateContent API with the
// operator's server-side key (GEMINI_API_KEY) -- unlike Trợ lý Quant's
// BYOK keys, this key is the server's and never leaves it.
type GeminiReviewer struct {
	apiKey string
	model  string
	base   string
	client *http.Client
}

// NewGeminiReviewer returns nil when apiKey is empty, so the service
// reports the AI review as unavailable.
func NewGeminiReviewer(apiKey, model string) *GeminiReviewer {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return &GeminiReviewer{apiKey: apiKey, model: model, base: geminiBase, client: &http.Client{Timeout: 60 * time.Second}}
}

func (g *GeminiReviewer) Model() string { return g.model }

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

func (g *GeminiReviewer) Review(ctx context.Context, system, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"systemInstruction": geminiContent{Parts: []geminiPart{{Text: system}}},
		"contents":          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}},
		"generationConfig":  map[string]any{"temperature": 0.3, "maxOutputTokens": 4096},
	})
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/models/%s:generateContent", g.base, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// Header, not ?key=, so the key never shows up in a logged URL.
	req.Header.Set("x-goog-api-key", g.apiKey)
	resp, err := g.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("%w: Gemini timed out", ErrAIFailed)
		}
		return "", fmt.Errorf("%w: could not reach Gemini", ErrAIFailed)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAIFailed, err)
	}

	var out struct {
		Candidates []struct {
			Content      geminiContent `json:"content"`
			FinishReason string        `json:"finishReason"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(data, &out)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := out.Error.Message
		if msg == "" {
			msg = resp.Status
		}
		return "", fmt.Errorf("%w: Gemini %d: %s", ErrAIFailed, resp.StatusCode, msg)
	}
	var b strings.Builder
	for _, c := range out.Candidates {
		for _, p := range c.Content.Parts {
			b.WriteString(p.Text)
		}
		break
	}
	if strings.TrimSpace(b.String()) == "" {
		return "", fmt.Errorf("%w: Gemini returned no text", ErrAIFailed)
	}
	return b.String(), nil
}
