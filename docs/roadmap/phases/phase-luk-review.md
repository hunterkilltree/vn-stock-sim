# Phase: Martin Luk rule review on the stock detail page

A **"Quy tắc Martin Luk"** card on `/stocks/[symbol]` (right column, under
"Khuyến nghị") that answers *"should I buy this today by Martin Luk's
rules?"* with a verdict -- **Đủ điều kiện mua / Theo dõi / Không mua** --
the checklist behind it, and the trade plan the rules imply (stop, 5R
target, position size, trailing exit). An optional Gemini commentary
reads the checklist and explains it in plain Vietnamese.

- Backend: `backend/internal/luk` -- `rules.go` (pure `Compute`, every
  threshold a named constant), `service.go`, `gemini.go` (the only vendor
  code, behind the `Reviewer` interface), `handler.go`.
- API: `GET /api/v1/symbols/:symbol/luk-review` (public, like `/rating`);
  `POST /api/v1/symbols/:symbol/luk-review/ai` (signed-in; 503
  `ai_disabled` without a key, 502 `ai_failed` on a Gemini error).
- Frontend: `LukReviewCard.tsx`, `LukAIReview.tsx` (client button),
  `lib/lukActions.ts` (server action), `getLukReview` in `lib/api.ts`.
- Config: `GEMINI_API_KEY` (unset hides the AI button), `GEMINI_MODEL`
  (default `gemini-2.5-flash`).

## Who Martin Luk is and where the rules come from

Martin Luk won the 2025 U.S. Investing Championship (stock division,
reported ~+969%) after +283% in 2024, trading momentum leaders with
pullback entries and very tight stops. There is no rulebook from him;
the rules below come from his own post on X (stop / risk / size numbers)
and interview summaries (EMA/AVWAP pullbacks, trailing exits, low win
rate, ~5R winners). Treat the specifics as approximate -- he says himself
they're "not hard rules".

- His post: "Stop = 1.5-5%, Portfolio risk per trade = 0.5%, Max position
  sizing = 35% ... Faster ADR names tend to have wider stop and smaller
  position sizing" -- https://x.com/martinlukkt/status/1906196925122310291
- Pullback method (9/21/50 EMA + anchored VWAP confluence, stop under
  the day's low, trail the 9 EMA, sell ~10% partials into strength):
  financialwisdomtv.com, tradingresourcehub.substack.com (USIC 2024
  review), en.forexclub.pl, ideas.finetiqsystems.com.

## Decisions

1. **Rules are deterministic; the AI only explains.** Every rule is a
   number on daily bars, so a rule engine is exact and free. Gemini gets
   only the computed checklist (no free-form market data) and is told not
   to invent figures -- it can't hallucinate a price. Same reasoning as
   `internal/rating` and `internal/insight`.
2. **Nine checks in three groups** (✓ pass / ! warn / ✕ fail; *core*
   checks decide the verdict):
   - *Chọn cổ phiếu (setup)*
     - **Trend** (core): close > EMA21, EMA9 > EMA21 > EMA50. Warn when
       only EMA21 > EMA50 and price > EMA50; fail otherwise. Luk only
       buys pullbacks inside uptrends.
     - **Relative strength** (core): 63-session change minus VN-Index's;
       ≥ +10 pp pass, > 0 warn, else fail. He trades leaders.
     - **ADR 20** = mean(high/low − 1): ≥ 4% pass, ≥ 2.5% warn. Luk
       prefers > 5% in the US; HOSE's ±7% band compresses ranges, so the
       bar is lower here.
     - **Liquidity** (core): 20-day average traded value ≥ 10 tỷ pass,
       ≥ 3 tỷ warn (he sizes down on small caps).
   - *Điểm vào (entry)*
     - **Pullback to EMA 9/21** (core): a low within 1% of max(EMA9,
       EMA21) in the last 3 sessions and close ≤ 1.5 ADR above EMA21 →
       pass; > 2 ADR above → warn "extended, don't chase"; close > 1%
       below EMA21 → fail (support broke).
     - **Confluence**: EMA9, EMA21 and an anchored VWAP (anchored at the
       3-month swing low) within one ADR of each other → pass; close below
       that AVWAP → fail.
     - **Volume dry-up**: last 3 sessions ≤ 0.8× the 20-day average → pass.
   - *Rủi ro & thị trường (risk)*
     - **Stop** (core): stop = today's low − 0.2%, but at least 1.5%
       below the close (tighter is noise). Pass within max(1.5%, ½ ADR)
       capped at 5%; warn ≤ 5%; fail > 5%.
     - **Market** (core): VN-Index > EMA21 > EMA50 pass; > EMA50 warn;
       else fail.
3. **Verdict.** A failed core check in setup or risk → *Không mua*. Any
   other non-passing core check, or any failed check → *Theo dõi* (a
   failed entry check means "not today", not "never"). Otherwise *Đủ điều
   kiện mua*.
4. **Plan.** Entry = last close; risk 0.5% of the account; position =
   0.5% ÷ stop%, capped at 35%; target = entry + 5 × (entry − stop);
   trailing exit = close below EMA 9, selling ~10% partials into
   strength. Prices are floored onto the exchange's tick grid. Hidden in
   the UI when the verdict is *Không mua* so it doesn't read as a trade
   idea.
5. **Vietnam caveats shown on every review**: T+2.5 settlement (a stop
   can't be honoured the first two sessions), price bands and limit-down
   gaps, Luk's ~25% win rate, and "not investment advice".
6. **Gemini, server key.** Unlike Trợ lý Quant (BYOK per user, phase-h.md),
   this uses the operator's `GEMINI_API_KEY`, sent as the
   `x-goog-api-key` header (never in a URL that could be logged). The AI
   endpoint requires sign-in because it spends that quota, and a review
   is cached per symbol + checklist for 15 minutes. Crypto is not
   reviewed (404).

## Verification

- `go test ./internal/luk`: leader pullback → buy with a capped 0.5%
  position; downtrend → avoid; extended leader → watch; weak market →
  avoid; short history; ADR/AVWAP math; Gemini request shape (path,
  header key, systemInstruction) and error propagation against an
  `httptest` server.
- Ran the API on mock data: HPG/VCB/FPT return full checklists; the AI
  endpoint returns 401 signed out, and with a dummy key reaches the real
  Gemini API and maps its "API key not valid" to 502 `ai_failed`.
- Rendered the card in Chromium (dev server against the mock API).
- Not verified: a real Gemini answer (no key in the sandbox).

## Checking Gemini works (logs)

Every line starts with `luk gemini:` or `luk ai review:`, so
`grep luk` on the backend log (Render → Logs, `docker compose logs
backend`) shows them. The key is never logged in full -- only its first
and last 4 characters and its length, enough to spot a wrong key or one
pasted with quotes/spaces.

| When | Line | Meaning |
|---|---|---|
| Startup | `AI review DISABLED (button hidden)` | `GEMINI_API_KEY` is empty or missing. |
| Startup | `AI review ENABLED (model=… key=AIza…1234 (39 chars))` | A key was read; a check follows. |
| Startup | `setup check OK` | Key valid and `GEMINI_MODEL` exists (a free `GET models/{model}`, no tokens). |
| Startup | `setup check FAILED …: Gemini 400: API key not valid` | Bad key. `404 … not found` = wrong model name; `403` = API not enabled / key restricted. |
| Each click | `call OK model=… took=… finish=STOP promptTokens=… outputTokens=…` | Gemini answered; token counts are what the call cost. |
| Each click | `call FAILED … Gemini 429: …` | Quota/rate limit. `no text (finish=MAX_TOKENS/SAFETY)` = Gemini answered empty. |
| Each click | `luk ai review: symbol=FPT OK verdict=watch` / `FAILED` / `served from cache` | Per-stock outcome; a cache hit makes no Gemini call. |

## Future work

- A screener preset "Luk setups today" over the whole universe.
- Intraday (5-minute) entries and opening-range highs, which Luk also
  uses, once intraday bars are reliable.
- Let the Replay score grade entries against this checklist.
