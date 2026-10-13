package luk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

// MaskedKey is the key as logs may show it: enough to tell which key is
// loaded (and that it isn't padded with quotes or spaces), never usable.
func (g *GeminiReviewer) MaskedKey() string { return maskKey(g.apiKey) }

func maskKey(k string) string {
	if len(k) <= 8 {
		return fmt.Sprintf("*** (%d chars)", len(k))
	}
	return fmt.Sprintf("%s…%s (%d chars)", k[:4], k[len(k)-4:], len(k))
}

// Check asks Gemini for the configured model's metadata: one cheap call
// (no tokens spent) that fails the same way a review would on a bad key,
// a disabled API or a misspelled GEMINI_MODEL.
func (g *GeminiReviewer) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/models/%s", g.base, g.model), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", g.apiKey)
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach Gemini: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("Gemini %d: %s", resp.StatusCode, errMessage(data, resp.Status))
	}
	return nil
}

// LogStartupCheck runs Check and logs the outcome -- the line to look for
// after a deploy to know the key and model work.
func (g *GeminiReviewer) LogStartupCheck(ctx context.Context) {
	if err := g.Check(ctx); err != nil {
		log.Printf("luk gemini: setup check FAILED (model=%s key=%s): %v", g.model, g.MaskedKey(), err)
		return
	}
	log.Printf("luk gemini: setup check OK (model=%s key=%s)", g.model, g.MaskedKey())
}

func errMessage(data []byte, fallback string) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}
	return fallback
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

// Review logs one line per call: status, latency and token usage on
// success; status and Gemini's own error message on failure. Never the
// key, the prompt or the answer.
func (g *GeminiReviewer) Review(ctx context.Context, system, prompt string) (string, error) {
	start := time.Now()
	text, usage, err := g.generate(ctx, system, prompt)
	took := time.Since(start).Round(time.Millisecond)
	if err != nil {
		log.Printf("luk gemini: call FAILED model=%s took=%s: %v", g.model, took, err)
		return "", err
	}
	log.Printf("luk gemini: call OK model=%s took=%s finish=%s promptTokens=%d outputTokens=%d totalTokens=%d chars=%d",
		g.model, took, usage.finish, usage.PromptTokenCount, usage.CandidatesTokenCount, usage.TotalTokenCount, len(text))
	return text, nil
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	finish               string
}

func (g *GeminiReviewer) generate(ctx context.Context, system, prompt string) (string, geminiUsage, error) {
	var usage geminiUsage
	body, err := json.Marshal(map[string]any{
		"systemInstruction": geminiContent{Parts: []geminiPart{{Text: system}}},
		"contents":          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}},
		"generationConfig":  map[string]any{"temperature": 0.3, "maxOutputTokens": 4096},
	})
	if err != nil {
		return "", usage, err
	}
	url := fmt.Sprintf("%s/models/%s:generateContent", g.base, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", usage, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Header, not ?key=, so the key never shows up in a logged URL.
	req.Header.Set("x-goog-api-key", g.apiKey)
	resp, err := g.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", usage, fmt.Errorf("%w: Gemini timed out", ErrAIFailed)
		}
		return "", usage, fmt.Errorf("%w: could not reach Gemini", ErrAIFailed)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", usage, fmt.Errorf("%w: %v", ErrAIFailed, err)
	}

	var out struct {
		Candidates []struct {
			Content      geminiContent `json:"content"`
			FinishReason string        `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata geminiUsage `json:"usageMetadata"`
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", usage, fmt.Errorf("%w: Gemini %d: %s", ErrAIFailed, resp.StatusCode, errMessage(data, resp.Status))
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", usage, fmt.Errorf("%w: unreadable Gemini response: %v", ErrAIFailed, err)
	}
	usage = out.UsageMetadata
	var b strings.Builder
	for _, c := range out.Candidates {
		usage.finish = c.FinishReason
		for _, p := range c.Content.Parts {
			b.WriteString(p.Text)
		}
		break
	}
	if strings.TrimSpace(b.String()) == "" {
		// e.g. finish=SAFETY, or MAX_TOKENS spent entirely on thinking.
		return "", usage, fmt.Errorf("%w: Gemini returned no text (finish=%s)", ErrAIFailed, usage.finish)
	}
	return b.String(), usage, nil
}
