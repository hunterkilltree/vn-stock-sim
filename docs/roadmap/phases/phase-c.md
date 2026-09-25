# Phase C — planning and analysis

Scope per FULL-APP-PLAN.md section 5, corrected per the design-alignment
decision: rebuild `/stocks` (the Main/market-overview screen) reading
`design/screens/Main.dc.html` as the literal spec, not the plan's own
older prose summary. Backed entirely by Phase B's endpoints.

## Reading design/screens/Main.dc.html

Read in full (392 lines, including its `renderVals()` data-generation
block, which doubles as the exact response-shape and formatting spec).
Key findings that change what Phase A built:

1. **The sidebar's nav list is NOT the 6-item list Phase A invented.**
   The real spec has exactly 10 items in the scrolling nav (`navRaw` in
   the script block), each with its own exact SVG path (`d`), plus a
   separate `Cai dat` (Settings) link rendered OUTSIDE the loop, right
   before the balance card, always in the inactive style. Only 2 of the
   10 have an href to a page that will exist after Phase C/D (`Tong
   quan` -> Main, `Bieu do & chi bao` -> Detail); `Giao dich giay`
   (Portfolio), `Che do Replay`, and `Tro ly Quant` point at real,
   *planned* screens (Phase E/F/H) with no WILL badge (`planned` flag is
   `false` for these three in the source); `Bo loc co phieu`, `Ban do
   nhiet`, `Xay chien luoc`, `Kiem thu lich su`, `So giao dich` are
   `planned: true` -- true backlog, WILL-badged, matching
   `design/SCREENS.md`'s own "Not designed yet" list exactly.
2. **The logo lockup is two lines**, not the single-line mark Phase A
   built: a 32x32 accent-filled rounded-9 icon box (bar-chart glyph) +
   "VN Stock Sim" (Lora 700 17px) + a caption line "Mo phong . HOSE"
   (10px muted uppercase). Phase A's `SidebarNav` only had the single
   `LogoMark` line.
3. Every icon in this spec is a literal SVG `d` path string carried in
   the data, not a named component -- the cleanest way to stay exact is
   to store those `d` strings directly rather than hand-drawing new
   icons, so this phase replaces `navItems.ts`'s ad hoc icon-component
   list with path data lifted verbatim from the source.
4. **Movers table needs price and volume**, which
   `screener.GetTopMovers` (Phase B) does not return -- it only has
   `{symbol, changePercent}`. The design's columns are Ma/Gia/+-/KL
   (symbol/price/change%/volume). This is a real gap in what Phase B
   shipped, not a frontend-only concern -- see Decision 3.
5. Sparkline data: the design generates a 30-point seeded random walk
   client-side. Phase B's `market.GetIndex` already returns a real
   20-point `sparkline: []float64` server-side -- render that (normalized
   into the same 112x36 viewBox math the spec uses), not a fake one; the
   backend's real data replaces the design's placeholder generator,
   consistent with this repo's standing rule to never fabricate data
   client-side when the backend has real numbers.
6. Heatmap tile color intensity formula (`tile()` in the script block):
   alpha = `0.12 + min(|pct|, 5) / 5 * 0.33`, green/red/amber background
   by sign, border at higher opacity. Directly portable to TS.
7. `tone()`/`sign()`/`num()` helpers (VN-locale number formatting,
   +/-/tham chieu color selection) are directly portable to a shared TS
   util.
8. The header's session-status line ("Phien 22/09/2026 . 14:45 . Khop
   lenh lien tuc") assumes a VN trading-session/hours model this repo
   has not built (out of scope, not in Phase B) -- see Decision 4.
9. The right column's `Tai khoan giay` (paper account) card, `Vi the
   dang mo` (open positions) table, Replay promo card, and "Hoi Quant"
   quick-ask card are all present in the design assuming a logged-in
   user with an existing portfolio -- guest handling isn't shown in this
   artboard at all (per design/DESIGN-SYSTEM.md section 9, guest-first
   is a documented product rule, just not drawn on this specific
   artboard) -- see Decision 5.

## Decisions

1. **Sidebar rework is in-scope for Phase C, not deferred.** `SidebarNav`
   is a shared component Main is the first real consumer of; building
   Main's real content on top of Phase A's placeholder 6-item nav would
   mean redoing the sidebar again in Phase D/E anyway. Rebuilding it now
   against the actual spec is strictly less total work.
2. **Icon storage: path data lives in `navItems.ts`, not in
   `icons.tsx`.** Matches the design's own approach (SVG generated from
   data, not hand-authored named components) and guarantees pixel-exact
   icons since the `d` strings are copied verbatim from
   `design/screens/Main.dc.html`, not redrawn by eye. `icons.tsx` keeps
   only the handful of icons used outside the nav loop (logo glyph,
   settings gear, search, replay, quant sparkle, chevron/arrow) as
   plain functions returning a `d` string, same treatment.
3. **`screener.TickerChange` gains `Price`/`Volume`.** Small, targeted
   backend addition, not scope creep: `market.Service` gains
   `LatestQuote(sym) (price float64, volume int64, ok bool)` (reads the
   most recent bar from the existing `GetBars`, same technique
   `LatestClose` already uses), and `screener.Service` takes a second
   dependency, a `MarketPort` satisfied by `*market.Service`, to fill
   these two fields when building `TickerChange` rows. `GetSectorHeatmap`
   ignores the new fields (heatmap tiles only ever showed
   symbol+percent); only `GetTopMovers` populates them.
4. **Session-status line is simplified, not modeled.** No VN
   trading-hours/session engine exists (correctly deferred -- nothing in
   FULL-APP-PLAN.md schedules that work this early). Render a real,
   live-computed date/time (server-rendered "now", not the design's
   fixed sample string) plus a static "Khop lenh lien tuc" ("Continuous
   matching") label and the green status dot, honestly representing
   "this is always shown as open" rather than building a fake session
   calendar just to make the label dynamic.
5. **Guest state**: the right column (paper account card, open
   positions, Replay promo, Quant quick-ask) only renders for an
   authenticated session (`getSessionUser()`, same pattern the existing
   `Navbar` already uses); a guest sees the left/center content (indices,
   heatmap, movers) plus a sign-up prompt card in the right column's
   place -- matches design/DESIGN-SYSTEM.md section 9's "guest-first"
   rule, which this specific artboard doesn't draw but the system-wide
   rules document requires.
6. **Sparse heatmap/movers is expected, not a bug.** V1's mock fixture
   universe is 5 symbols in 4 sectors (`VNM`/`VCB`/`HPG`/`FPT`/`SHB`);
   the design's sample data assumes ~24 symbols. The screen renders
   whatever the real backend returns -- a sparser grid -- rather than
   padding it with fabricated tickers to visually match the mockup's
   density. Revisit once a real market-data vendor replaces the mock
   fixtures (already a tracked RESUME.md future-work item).
7. **Avatar button**: shows the first two letters of the user's
   `displayName` (matching the spec's static "TD" initials placeholder)
   when authenticated, a generic account glyph for guests; the button
   itself stays non-interactive (Account Menu is Phase G), same
   `cursor-not-allowed` treatment Phase A already established.

## Files to add

- `frontend/src/lib/format.ts` -- `formatVN(value, decimals)`,
  `signVN(value, decimals)` (+/-, matches the design's `sign()`), `tone
  (value)` (up/down/ref hex, matches `tone()`), ported directly from
  Main.dc.html's script block.
- `frontend/src/components/IndexCard.tsx`, `SectorHeatmap.tsx`,
  `MoversTable.tsx`, `PaperAccountCard.tsx`, `OpenPositionsCard.tsx`,
  `ReplayPromoCard.tsx`, `QuantPromptCard.tsx` -- one component per
  Main.dc.html section, each taking already-fetched data as props (the
  page Server Component does all fetching; these stay presentational).

## Files to change

- `frontend/src/lib/navItems.ts` -- replace with the real 10-item list +
  separate Settings entry, exact `d` path data, exact `kind` per item
  (`built`/`soon`/`will`) matching the source's `planned` flags.
- `frontend/src/components/SidebarNav.tsx` -- two-line logo lockup,
  separate Settings link below the nav loop, balance card matching the
  spec's exact markup/values-shape (still wired to real
  `GET /api/v1/portfolio` data, not the spec's static sample).
- `frontend/src/app/stocks/page.tsx` -- full rebuild: parallel fetch of
  indices/heatmap/movers (+ portfolio summary/positions when
  authenticated), assembled into the sections above.
- `backend/internal/market/service.go` -- add `LatestQuote`.
- `backend/internal/screener/service.go`, `types.go`, `handler.go` --
  `TickerChange` gains `Price`/`Volume`; `Service` takes a `MarketPort`.
- `backend/cmd/api/main.go` -- pass `marketSvc` into
  `screener.NewService`.

## Verification plan

1. `go build ./... && go vet ./...` (Docker, no local Go) for the
   backend changes; curl `GET /market/movers` and confirm `price`/
   `volume` are now present and non-zero.
2. `npx tsc --noEmit && npx eslint . && npm run build` for the frontend.
3. Browser-pane check at 1440px, both guest and logged-in (register a
   test user, log in): index cards render with real sparklines, heatmap
   renders the 4 real sectors, movers tables populate, and the right
   column correctly switches between the sign-up prompt (guest) and the
   real paper-account/positions cards (logged in). Compare directly
   against `design/screens/Main.dc.html` opened in the browser pane side
   by side for layout/spacing fidelity.
4. Full Docker rebuild + curl, matching this repo's standing habit.
5. Record the outcome in RESUME.md.

## Verification (done)

- Small, targeted backend addition first: `market.Service` gained
  `LatestQuote(sym) (price, volume, ok)`; `screener.Service` now takes a
  second dependency (`MarketPort`) so `GetTopMovers` can fill
  `TickerChange.Price`/`Volume` -- the design's movers table has
  Mã/Giá/+-/KL columns, and Phase B's `screener` only had symbol+percent.
  `GetSectorHeatmap` is unaffected (heatmap tiles never showed
  price/volume).
- `SidebarNav` fully rebuilt against the real spec: the exact 10-item nav
  list with exact SVG path data copied verbatim from
  `Main.dc.html`'s `navRaw` array (not redrawn by eye), a two-line logo
  lockup (icon box + "VN Stock Sim" + "Mô phỏng · HOSE" caption), a
  separate "Cài đặt" (Settings) link outside the nav loop, and a balance
  card now wired to the real `GET /portfolio` cash balance instead of a
  static sample. `navItems.ts` correspondingly reworked: `kind` per item
  now matches the source's real `planned` flags (Portfolio/Replay/Quant
  are real *planned* screens, not backlog -- only Screener/Heatmap/
  Strategy-Builder/Backtest/Trade-Journal are true `will` items, matching
  design/SCREENS.md's own "Not designed yet" list exactly). `RailNav`
  updated to compile against the new `NavItem` shape; its own full spec
  fidelity (icon set, ordering) is deferred to Phase D against
  `design/screens/Detail.dc.html`.
- New `frontend/src/lib/format.ts` -- `formatVN`/`signVN`/`tone`/
  `formatVolumeVN`, ported directly from `Main.dc.html`'s `renderVals()`
  script block (not reinvented), so number formatting matches the design
  exactly (VN locale, literal minus sign U+2212, "tr" short form).
- New presentational components, one per Main.dc.html section:
  `IndexCard` (real 20-point sparkline from Phase B's `IndexSnapshot`,
  normalized into the design's 112x36 viewBox math -- not the design's
  fake client-generated sparkline), `SectorHeatmap` (tile color-intensity
  formula ported verbatim), `MoversTable`, `PaperAccountCard`,
  `OpenPositionsCard`, `ReplayPromoCard`, `QuantPromptCard`.
- `/stocks/page.tsx` fully rebuilt: parallel-fetches indices/heatmap/
  movers (public) plus portfolio summary/positions/portfolios (only when
  `getSessionUser()` resolves), assembling the full screen. Guest state
  (no session) renders the market-wide content plus a sign-up prompt card
  in the right column instead of crashing or faking portfolio data --
  matches design/DESIGN-SYSTEM.md section 9's guest-first rule, which
  this specific artboard doesn't draw but the system-wide rules document
  requires.
- Two documented, deliberate simplifications, not bugs: (1) the header's
  session-status line shows a real live-computed date/time + a static
  "Khớp lệnh liên tục" label rather than a fake VN trading-hours model
  (none exists yet, correctly out of scope this early); (2) the heatmap/
  movers are visibly sparser than the mockup's sample data, because V1's
  real mock fixture universe is 5 symbols across 4 sectors vs. the
  design's ~24 -- rendering whatever the real backend returns rather than
  padding with fabricated tickers.
- Replay/Quant promo cards and the header's "Vào Replay" button render
  disabled (not linked) since `/replay` and `/quant` don't exist until
  Phases F/H -- clicking them would otherwise 404.

Verified, not just read: `go build ./... && go vet ./...` clean (Docker,
no local Go); curled `GET /market/movers` and confirmed real
`price`/`volume` now present. `npx tsc --noEmit`, `npx eslint .`,
`npm run build` all clean. Browser-pane check at 1440x960: registered a
real test account through the actual UI (hit one real, since-fixed
browser-automation flake along the way -- a `form_input`-then-`click by
ref` sequence silently didn't submit twice in a row with no console
error; a direct coordinate click on the rendered button worked, and the
Server Action logs confirmed the POST only fired on that click), and
confirmed both states: guest (sign-up prompt card, no fabricated
balance) and logged-in (real "100.000.000 ₫" balance in both the sidebar
card and the right-column Paper Account card, "Vị thế đang mở" showing
the correct empty state for a brand-new account with no positions).
Checked 375x812: sidebar/content overflow horizontally as expected --
same documented, deferred-to-Phase-J behavior as Phase A, not a new
regression. Re-verified inside the actual rebuilt Docker image: all six
routes return HTTP 200, and the Main screen's real content (heatmap,
paper account, Quant prompt) is present in the served HTML.

Branch: phase-c-main-screen (off master, after PR #15 merged).
Merged as PR #16.


### Follow-up: VN session timezone bug (2026-09-22)

Bug found and fixed 2026-09-22 (follow-up to Phase C, user-reported) --
DONE. The Market Overview header's session-status line ("Phiên
<date> · <time> · Khớp lệnh liên tục") was computed with
`new Date().toLocaleDateString/toLocaleTimeString("vi-VN", ...)` with no
explicit `timeZone`, so it rendered in the server process's local
timezone (UTC inside this repo's Docker containers) rather than the
Vietnamese exchanges' own timezone (Asia/Ho_Chi_Minh, ICT, UTC+7) --
mislabeling a UTC clock reading as if it were VN local session time, off
by exactly 7 hours (and on the wrong calendar date whenever "now" fell
in that 7-hour window past UTC midnight). Same class of bug as the
2026-09-21 stock-price/chart UTC-freshness fix: a real timestamp
rendered against the wrong reference. Fixed by passing
`timeZone: "Asia/Ho_Chi_Minh"` explicitly to both calls
(frontend/src/app/stocks/page.tsx's `nowSessionLabel`).

Verified, not just read: confirmed the Alpine-based Docker image's
Node/ICU build actually has Asia/Ho_Chi_Minh timezone data available
(worth checking explicitly on `node:20-alpine`, not assumed) by
rebuilding the frontend image and comparing `date -u` against the
rendered page inside the real container -- UTC 23:36 correctly rendered
as VN "23/09/2026 · 06:36" (next calendar day, +7h), both via the dev
server and the built Docker image. All six routes re-curled, still
HTTP 200.

Branch: fix-vn-session-timezone (off master, after PR #16 merged).
Merged as PR #17.

### Also shipped around this time: the Postman collection (2026-09-23)

Postman collection (backend/postman/vn-stock-sim.postman_collection.json),
2026-09-23 -- DONE, per an explicit user request ("document the
collection of API such that I can test backend result in postman").
Covers every route currently registered in cmd/api/main.go, generated by
reading each feature's actual handler.go/types.go (not api-spec.md's
intent document), so field names and binding rules are exactly what the
code accepts today -- 8 folders (Auth, Symbols, Market, Watchlist,
Portfolio, Orders, Backtests) totaling 24 requests:

- Auth: Register, Login (both public; test scripts save the returned
  accessToken into a collection variable), Me.
- Symbols: Search, Get Detail, Get Insight (all public).
- Market: Get Bars, Get Indicator, Get Indices, Get Order Book, Get
  Heatmap, Get Movers, Get Server Time (all public; a collection-level
  pre-request script computes fresh nowTs/from10dTs/from180dTs unix-
  second variables before every request, so the date-ranged endpoints
  work with zero manual editing).
- Watchlist, Portfolio (singular default-portfolio routes + the plural
  multi-portfolio API from Phase B), Orders (market + queued limit/atc/
  stop examples), Backtests (ema_crossover) -- all authenticated,
  inheriting the collection-level Bearer auth.
- Collection variables (baseUrl, accessToken, symbol, portfolioId,
  orderId, backtestId, ...) plus test scripts on List/Create Portfolio,
  both Create Order variants, and Create Backtest, so a first-time user
  can run every request top-to-bottom with zero manual copy-pasting of
  IDs between requests.

Verified, not just written: ran the entire collection against the real
backend with Newman (`npx newman run backend/postman/vn-stock-sim.postman_collection.json`)
-- 31 requests (24 defined + Newman's own iteration bookkeeping), 0
failures. Caught and fixed one real sequencing bug this way, not just by
reading the JSON: "Cancel Order" run against the accumulated
{{orderId}} initially hit the filled *market* order's id (only
"queued" orders can be cancelled) and returned 409 -- confusing as a
default first-run experience. Fixed by having "Create Order (Limit,
queued)" also overwrite {{orderId}}, so the default top-to-bottom run
now demonstrates a real 200 OK cancel; re-ran the full collection after
the fix and confirmed Cancel Order returns 200.

Branch: postman-api-collection (off master, after PR #17 merged).
Merged as PR #18.
