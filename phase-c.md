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
