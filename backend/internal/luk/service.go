package luk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var (
	ErrSymbolNotFound = errors.New("symbol not found")
	// ErrAIDisabled: no GEMINI_API_KEY on this server.
	ErrAIDisabled = errors.New("ai review is not configured")
	ErrAIFailed   = errors.New("ai review failed")
)

// lookback: ~1 year of daily bars -- EMA 50 warm-up plus the 3-month
// RS and AVWAP windows with room to spare.
const lookbackSeconds = 400 * 24 * 60 * 60

const indexName = "VN-Index"

// aiCacheTTL: one Gemini call per symbol per trading snapshot is enough;
// the checklist only changes when a new bar arrives.
const aiCacheTTL = 15 * time.Minute

type SymbolPort interface {
	Detail(sym string) (symbol.Detail, bool)
}

type MarketPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

// Reviewer writes a short commentary from a prompt. The Gemini adapter
// (gemini.go) is the only implementation; the service never sees the
// vendor's wire format.
type Reviewer interface {
	Review(ctx context.Context, system, prompt string) (string, error)
	Model() string
}

type Service struct {
	symbols  SymbolPort
	market   MarketPort
	reviewer Reviewer // nil when GEMINI_API_KEY is unset

	mu      sync.Mutex
	aiCache map[string]cachedReview
}

type cachedReview struct {
	review AIReview
	at     time.Time
}

func NewService(symbols SymbolPort, market MarketPort, reviewer Reviewer) *Service {
	return &Service{symbols: symbols, market: market, reviewer: reviewer, aiCache: map[string]cachedReview{}}
}

func (s *Service) Review(sym string) (Result, error) {
	detail, ok := s.symbols.Detail(sym)
	if !ok || detail.Exchange == symbol.ExchangeCrypto {
		return Result{}, ErrSymbolNotFound
	}
	now := time.Now()
	from, to := now.Unix()-lookbackSeconds, now.Unix()
	res := Compute(Input{
		Symbol:    detail.Symbol.Symbol,
		Bars:      s.market.GetBars(detail.Symbol.Symbol, "1D", from, to),
		IndexBars: s.market.GetBars(indexName, "1D", from, to),
		Tick:      func(p float64) float64 { return symbol.FloorToTick(p, detail.Exchange) },
	})
	res.AIAvailable = s.reviewer != nil
	res.GeneratedAt = now.UTC().Format(time.RFC3339)
	return res, nil
}

// AIReview asks Gemini to review the checklist. The model only comments on
// numbers the rules computed -- it never sees anything else, so it can't
// invent prices.
func (s *Service) AIReview(ctx context.Context, sym string) (AIReview, error) {
	if s.reviewer == nil {
		return AIReview{}, ErrAIDisabled
	}
	res, err := s.Review(sym)
	if err != nil {
		return AIReview{}, err
	}
	key := res.Symbol + "|" + checklistKey(res)
	s.mu.Lock()
	if c, ok := s.aiCache[key]; ok && time.Since(c.at) < aiCacheTTL {
		s.mu.Unlock()
		log.Printf("luk ai review: symbol=%s served from cache (no Gemini call)", res.Symbol)
		return c.review, nil
	}
	s.mu.Unlock()

	text, err := s.reviewer.Review(ctx, systemPrompt, userPrompt(res))
	if err != nil {
		log.Printf("luk ai review: symbol=%s FAILED: %v", res.Symbol, err)
		return AIReview{}, err
	}
	log.Printf("luk ai review: symbol=%s OK verdict=%s", res.Symbol, res.Verdict)
	out := AIReview{
		Symbol:      res.Symbol,
		Verdict:     res.Verdict,
		Text:        strings.TrimSpace(text),
		Model:       s.reviewer.Model(),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.mu.Lock()
	for k, c := range s.aiCache {
		if time.Since(c.at) >= aiCacheTTL {
			delete(s.aiCache, k)
		}
	}
	s.aiCache[key] = cachedReview{review: out, at: time.Now()}
	s.mu.Unlock()
	return out, nil
}

func checklistKey(r Result) string {
	var b strings.Builder
	for _, c := range r.Checks {
		b.WriteString(c.Key + "=" + c.Status + "," + c.Value + ";")
	}
	return b.String()
}

const systemPrompt = `Bạn là trợ lý luyện tập giao dịch cho một ứng dụng mô phỏng chứng khoán Việt Nam (tiền ảo, không phải tư vấn đầu tư).
Bạn đánh giá một cổ phiếu theo phương pháp swing trading công khai của Martin Luk (vô địch US Investing Championship):
- Chỉ mua cổ phiếu dẫn dắt (mạnh hơn thị trường) đang trong xu hướng tăng, EMA 9 > EMA 21 > EMA 50.
- Mua khi giá điều chỉnh về hỗ trợ: EMA 9/21/50 và Anchored VWAP, tốt nhất khi các mức hội tụ; không đuổi giá khi bị kéo giãn.
- Dừng lỗ chặt dưới đáy phiên, 1,5–5%, không quá nửa ADR; rủi ro 0,5% tài khoản mỗi lệnh; vị thế tối đa 35%.
- Thoát dần ~10% vào sức mạnh, phần còn lại trailing theo EMA 9 (EMA 21/50 khi xu hướng rất mạnh).
- Tỷ lệ thắng thấp (~25%), lãi đến từ vài lệnh thắng lớn ~5R.
- Chỉ giao dịch khi thị trường chung thuận lợi.
Lưu ý riêng Việt Nam: T+2,5 (không bán được ngay), biên độ giá ±7%/±10%/±15%.

Chỉ dùng số liệu trong checklist được cung cấp, không bịa thêm giá, tin tức hay số liệu cơ bản.
Trả lời bằng tiếng Việt, ngắn gọn (tối đa khoảng 180 từ), dạng văn bản thuần với các gạch đầu dòng "- ", gồm:
1. Kết luận một câu: Mua / Theo dõi / Không mua theo quy tắc Luk, và lý do chính.
2. Điểm mạnh và điểm yếu quan trọng nhất của thiết lập.
3. Nếu chưa mua được: điều kiện cụ thể cần xuất hiện (ví dụ giá về EMA 21 ở mức nào).
4. Một câu về quản trị rủi ro với T+2,5.`

func userPrompt(r Result) string {
	payload := map[string]any{
		"symbol":  r.Symbol,
		"verdict": r.Verdict,
		"passed":  fmt.Sprintf("%d/%d", r.Passed, r.Total),
		"checks":  r.Checks,
		"plan":    r.Plan,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return "Checklist theo quy tắc Martin Luk (verdict: buy = đủ điều kiện mua, watch = theo dõi, avoid = không mua):\n" + string(b)
}
