# Phase H — AI Model settings (bring-your-own-key) and the Quant assistant

Plan written against `design/screens/Settings-AI.dc.html` and
`design/screens/Quant.dc.html`, per FULL-APP-PLAN.md section 10 and 2.2.
Quant-Chart (the radial-menu stretch item, 10.5) is **not** in this
phase — the plan says to build it only after 1–4 are solid, and it is a
UI layer over the same `/quant/chat` call.

## Ground rules carried from the plan

- **Bring your own key, never a company-funded call.** The "Quant Cloud"
  provider is rendered disabled with a note that it needs a platform
  decision. Nothing calls a model until the user has entered a key.
- **The key is never stored server-side.** It lives in the browser's
  `localStorage` and travels with each request, through the Next.js
  server and the Go backend, straight to the provider. Neither server
  persists or logs it.
- **The model never does the arithmetic.** It only turns the question
  into structured conditions / a strategy draft; the backend then runs
  those against real data (the same symbol/market services the rest of
  the app uses) and the existing backtest engine.

## Design decisions (made before code)

1. **Structured output, validated twice.** The model must return one JSON
   object with a fixed shape: `reply` (Vietnamese text), `screen`
   (`exchange` + a list of `{field, op, value, period}` conditions) and
   `strategy` (`kind` = `none` | `ema_crossover` | `rsi_reversion` plus its
   parameters). No nullable fields — "nothing" is an empty condition list
   / `kind: "none"` — so the same schema works for every provider.
   - Claude: the official Go SDK (`github.com/anthropics/anthropic-sdk-go`,
     per the Claude API guidance: use the SDK, not raw HTTP) with
     **structured outputs** (`output_config.format` json_schema), which
     guarantees schema-valid JSON.
   - OpenAI: Chat Completions with `response_format` json_schema (strict).
   - Self-hosted OpenAI-compatible: same endpoint shape without
     `response_format` (support varies by server); the prompt asks for
     the JSON and the backend extracts the first JSON object.
   - Either way the backend re-validates: unknown fields/operators are
     dropped, numbers are range-clamped, and what was dropped is reported
     back so the UI can say so instead of silently ignoring it.
2. **Screen fields are only what the app can really compute**: price
   (thousand ₫, as the Detail screen quotes it), change %, RSI(14), ROE %,
   P/E, P/B, EPS, dividend yield %, market cap (tỷ ₫), 20-session average
   volume, and "price vs SMA(N)". Prices, RSI, SMA and volume come from
   live bars (VCI with mock fallback); fundamentals are the hand-seeded
   values RESUME.md already documents. The universe is whatever
   `symbol.Search` returns (5 tickers today), and the UI says "N trên M
   mã" with the real M, not the design's 394.
3. **Strategy drafts map to backtest rules that actually run.**
   `ema_crossover` already exists. The design's example ("buy when RSI
   crosses up 35, sell above 70 or at −7%") needs a new backtest rule,
   `rsi_reversion` (entry level, exit level, stop-loss %, optional
   trend filter "price > SMA(N)"). Both rules also gain max drawdown and
   profit factor, which the design's result card shows.
4. **Temperature ("Mức sáng tạo") is best-effort.** Current Claude
   models (Opus 5, Sonnet 5, Opus 4.7/4.8, Fable) and OpenAI reasoning
   models reject sampling parameters with a 400. The backend only sends
   temperature where the model is known to accept it, retries once
   without it on a temperature-related 400, and reports
   `temperatureApplied: false` so the UI shows "this model ignores the
   creativity setting" rather than pretending.
5. **Model choice.** For Claude the design's three tiers map to
   `claude-opus-5` (strongest, recommended), `claude-sonnet-5`
   (balanced) and `claude-haiku-4-5` (light). For every provider,
   "Kiểm tra" also lists the models the key can use (`GET /v1/models`),
   matching the design's "model list comes from your key after testing";
   OpenAI and self-hosted have no presets, so their model must be picked
   from that list or typed.
6. **Refusal handling.** Claude responses with `stop_reason: "refusal"`
   become a clear error, and for `claude-opus-5`/Fable models the
   request opts into server-side refusal fallbacks (`fallbacks:
   "default"`, beta `server-side-fallback-2026-07-01`), per the Claude
   API guidance's default.
7. **"Kiểm tra" / "Thử trước khi lưu" = one real round trip** on the
   design's own sample question ("Lọc HOSE có RSI dưới 35 và ROE trên
   15%."): returns success, latency, token count, and how many of the 3
   expected conditions (HOSE, RSI < 35, ROE > 15) the model got right —
   the design's "Hiểu đúng 5/5 điều kiện · 1,2 giây · 480 token", with
   real numbers.
8. **Data sent to the model follows the "Dữ liệu gửi cho mô hình"
   toggles.** The question always goes. Optional context, each only when
   its toggle is on: a snapshot of the symbol universe (prices/volume —
   "Giá và khối lượng"), RSI/SMA values ("Chỉ báo kỹ thuật"), the user's
   watchlist, and the active portfolio's positions ("Số dư và vị thế").
   "Sổ giao dịch và ghi chú riêng" is shown disabled: there are no
   private trade notes in the app yet.
9. **Permissions**: "Tự động kiểm thử quy tắc Quant sinh ra" is real —
   when on, a drafted strategy is backtested immediately. "Cho phép Quant
   đặt lệnh trên tài khoản giấy" is shown disabled/off with a WILL badge:
   letting a model place orders deserves its own design and phase.
10. **Honest key copy.** The design says the key is "encrypted and only
    stored on this device — VN Stock Sim cannot read your key". With a
    backend proxy the second half isn't literally true (the key passes
    through the server per request), and plain `localStorage` isn't
    encryption. The UI says what actually happens instead: stored only in
    this browser, sent with each request to the provider via the VN Stock
    Sim server, never saved or logged there.
11. **No API key in logs, and no SSRF.**
    - The browser talks to Next.js **Route Handlers**
      (`/api/quant/test`, `/api/quant/chat`), not Server Actions:
      `next dev` prints Server Action arguments to the terminal, which
      would print the key. The route handler adds the session token and
      forwards to Go.
    - A self-hosted endpoint URL comes from the user, and the backend
      will connect to it. Private/loopback/link-local addresses are
      refused at connect time (checked on the resolved IP, so DNS tricks
      don't bypass it) unless `QUANT_ALLOW_PRIVATE_ENDPOINTS=true`
      (for local development against e.g. Ollama).
12. **Usage panel ("Mức dùng tháng 9") shows real tokens** summed per
    month in this browser from each response's usage. There is no quota
    to show a "/ 200.000" against (that was Quant Cloud's free tier), so
    it shows the count and the question count only.
13. **Conversation history ("Hội thoại gần đây")** is kept in
    `localStorage` (last 10 conversations). "Mã Python" tab and "Lưu
    chiến lược" (Strategy Library, V2) render with WILL badges.
14. **Quant requires sign-in** (it reads the watchlist/positions and
    costs the user money per call); guests see a sign-in prompt, same
    pattern as Portfolio/Replay.

## Backend

- `internal/quant`: `types.go` (request/response, the output schema),
  `service.go` (prompt building, validation, `Test`/`Chat`, the
  `ModelClient` port it defines), `screen.go` (condition executor),
  `claude_client.go` (Anthropic SDK adapter), `openai_client.go`
  (OpenAI + OpenAI-compatible adapter), `netguard.go` (SSRF guard),
  `handler.go` (`POST /api/v1/quant/test`, `POST /api/v1/quant/chat`,
  authenticated).
- `internal/backtest`: `rsi_reversion` rule, max drawdown + profit factor
  on both rules.
- `go.mod` → Go 1.24 (the Anthropic SDK requires it); backend Dockerfile
  base image `golang:1.24-alpine`.

## Frontend

- `lib/quantSettings.ts` (localStorage settings, usage, history).
- `app/api/quant/{test,chat}/route.ts` (proxies).
- `app/settings/ai/page.tsx` → `AISettingsForm.tsx` (the full design).
- `app/quant/page.tsx` + `QuantChat.tsx` (chat thread, condition chips,
  results table with CSV export and add-to-watchlist, strategy card with
  backtest, "Chạy thử bằng Replay" linking `/replay?symbol=`).
- `navItems.ts`: Quant → built; Main's Quant prompt card links to it.

## Verification (done)

- Backend: `go build ./... && go vet ./...` clean, `gofmt` clean on every
  touched file. **The repo's first unit tests**, 18 total, all passing:
  `internal/backtest` (RSI round trip, 7% stop-loss exit, trend filter)
  and `internal/quant` (validation drops/clamps, condition labels, screen
  run on fake data, no context sent with every toggle off, the 3/3 test
  score, missing key rejected before any call, SSRF address list and URL
  checks, and both adapters against local fake servers speaking the real
  wire formats — including that the Claude adapter sends only the user's
  key even with `ANTHROPIC_API_KEY` set on the server, never sends
  `temperature` to `claude-opus-5`, requests structured JSON output and
  refusal fallbacks, and maps a 401 and a refusal correctly; and that the
  OpenAI adapter retries once without temperature on a 400).
- curl against the real backend: `/quant/test` via a local
  OpenAI-compatible stub → 3/3, 2 models listed, the user's key forwarded
  as `Bearer`; `/quant/chat` screen → correct matches, an unsupported
  "sentiment" condition dropped and reported, and the RSI values in the
  result equal to `GET /market/indicators?indicator=rsi` for the same
  symbols (16,27 / 17,59 / 45,32 / 63,43), i.e. computed by the app, not
  the model; `rsi_reversion` and `ema_crossover` backtests both return
  drawdown and profit factor; missing key → 400, unauthenticated → 401,
  unknown provider → 400. **One real call to Anthropic:** a deliberately
  invalid key through the SDK adapter came back as 502 `provider_error`
  with Anthropic's own "API key is invalid." (confirmed identical to
  calling api.anthropic.com directly), so the adapter reaches the live
  API and surfaces its errors. SSRF on a backend without the opt-in flag:
  127.0.0.1, localhost, 169.254.169.254, 10.0.0.5 and `file://` all
  refused with 400 `endpoint_not_allowed`.
- Frontend: `tsc`, `eslint`, `npm run build` clean; Newman 46 requests, 0
  failures (new: `rsi_reversion` backtest, two Quant error-path requests).
- Real browser (Playwright, production build, stub model): Quant shows
  the "connect a model" card with send disabled until configured; Settings
  → Máy chủ riêng + URL + key + model → "Kiểm tra" → "Hiểu đúng 3/3 điều
  kiện" and "Đã kết nối" chip, footer "Có 4 thay đổi chưa lưu" → Lưu →
  persisted across reload; a toggle then "Hoàn tác" restores. Quant: a
  screen question → chips, "3 trên 4 mã HOSE", results table, CSV export
  downloaded (3 rows), "Thêm vào theo dõi" added 3; a strategy question →
  strategy card with Mua/Bán/Cắt lỗ rows, auto-backtest KPIs, then
  "Kiểm thử chiến lược trên 3 mã" → per-symbol table; history, "Hội thoại
  mới", reopening a conversation, a general question (no screen), the
  Replay link pre-filling its symbol, and Main's "Hỏi Quant" card
  pre-filling the question all worked. No page errors. The test keys
  never appeared in the backend or Next.js logs.
- Found and fixed during the browser run: a long conversation grew the
  whole page and pushed the question box below the fold. The Quant page
  is now viewport-high with the thread scrolling inside (re-checked:
  page height 960, input still in view).

**Still to do (needs the user's own key):** a successful live round trip
with a real Claude or OpenAI key through the UI. Everything around it —
the adapter's request shape, the real API's error path, the full UI flow —
is verified above.

## Verification plan (as written before code)

- Go build/vet; unit tests for the parts that don't need a network:
  output validation, the screen executor, the RSI backtest rule, the SSRF
  guard, and both adapters against local fake HTTP servers speaking the
  real Anthropic / OpenAI wire formats.
- tsc/eslint/build; Newman (new requests for `/quant/*` error paths).
- Real browser: this sandbox has no provider key, so the end-to-end run
  uses the **self-hosted** provider pointed at a small local
  OpenAI-compatible stub (with `QUANT_ALLOW_PRIVATE_ENDPOINTS=true`) —
  exercising the real settings form, the real proxy chain, the real
  backend validation/screen/backtest. A live Claude/OpenAI round trip
  needs the user's own key and is listed as still to do.
