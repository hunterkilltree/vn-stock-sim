# Phase J — Mobile layouts

Branch `phase-j-mobile`, stacked on `phase-i-crypto` (PR #26). Source:
FULL-APP-PLAN.md section 12, plus the seven phone designs in
`design/screens/Mobile-*.dc.html` (Market, Detail, Replay, Portfolio,
Quant, Settings, Login; 390 × 844 artboards).

## Starting point (measured, 390 px, production build)

With mobile emulation, a page whose layout is wider than the screen makes
the browser zoom out to fit it, so "layout width" is the overflow measure:

| Page | Layout width | Problem |
|---|---|---|
| `/stocks` | 798 | sidebar takes half the screen, columns spill off it |
| `/stocks/FPT` | 828 | 344 px order column beside the chart |
| `/portfolio` | 635 | sidebar + tables |
| `/settings/ai` | 793 | sidebar + form grid |
| `/crypto` | 798 | same as `/stocks` |
| `/crypto/BTCUSDT` | 473 | order column |
| `/replay`, `/quant`, `/heatmap`, `/crypto/replay` | 390 | fit, but the 72 px rail eats the screen and the desktop layouts are cramped |
| `/`, `/login`, `/register` | 390 | fine |

Charts are SVGs drawn in a 924-unit-wide space; at 354 px their 10.5-unit
labels shrink to about 4 px, which is unreadable.

## Decisions

1. **One breakpoint, `lg` (1024 px).** Below it the app uses the phone
   shell: no sidebar or rail, a bottom tab bar, and single-column pages.
   From `lg` up nothing changes. Responsive classes on the existing pages,
   not separate mobile pages, as FULL-APP-PLAN.md asks.
2. **`MobileTabBar`** (`lg:hidden`, fixed to the bottom, 78 px + the
   device's safe-area inset) copies the design's tab bar: two tabs, the
   raised orange Replay button, two tabs. Tabs: Thị trường, Biểu đồ,
   Replay, Danh mục, Quant. The designs show "Sổ lệnh" as the fourth tab
   on Market/Portfolio but "Quant" on the Quant screen; since "Sổ lệnh"
   points at the Portfolio page (the same place as "Danh mục"), the bar
   always shows Quant, so every built screen is reachable. In crypto mode
   the tabs go to `/crypto`, `/crypto/BTCUSDT`, `/crypto/replay` and the
   wallet on `/crypto`, and Quant stays stock-only.
3. **Market switch on phones**: a compact Cổ phiếu | Crypto control in
   the Thị trường header (Market and Crypto overview, heatmap). The
   Mobile-Login caption "Mô phỏng · HOSE · Crypto" confirms both markets
   exist on mobile.
4. **Account menu**: the avatar button sits in each page header, as in
   Mobile-Market. The popover opens below it and is capped at the screen
   width minus the gutters. Settings stays reachable through it (the tab
   bar has no Settings slot, like the design).
5. **Main → Mobile-Market**: title and time, search and avatar buttons,
   the VN-Index hero with sparkline and breadth, the other three indices
   as chips, a 3-column heatmap of the 9 largest tickers, and top
   gainers as cards. The desktop-only panels (paper account, positions,
   Replay promo, Quant card) are hidden below `lg`: the tab bar reaches
   those screens.
6. **Detail → Mobile-Detail**: back button, ticker, the account button
   (the design's watchlist button is left out: Detail has no watchlist
   action on desktop either);
   price, change and the Trần/TC/Sàn stack; horizontally scrolling
   timeframes; a compact chart (candles + SMA 20 + volume, RSI strip) in
   the design's 354 × 238 space; a 4-up stat row (KL, P/E, ROE, Vốn hóa).
   A sticky bottom bar has Replay + "Mua giấy" + "Bán giấy" and replaces
   the tab bar on this screen, as in the design. The bar
   links down to the existing order ticket, placed below the chart, with
   that side preselected (`?side=`). No bottom sheet: the ticket already
   exists and a second one would duplicate its state. The book and
   fundamentals stack below.
7. **Charts get a `compact` mode** (DetailChart, ReplayChart) that
   redraws in the phone design's coordinate space (354 wide, 9.5 labels,
   the last ~40–60 bars) rather than scaling the desktop drawing down.
   Pages render the compact chart below `lg` and the full one above it.
8. **Replay → Mobile-Replay**: exit, "Replay · HPG", bar counter,
   progress bar with dates, compact masked chart with fills and stop
   line, "Đang giữ", large step controls (back / next / auto), 4 stats
   (NAV, Lãi/lỗ, Lệnh, Điểm). A sticky bottom bar has "Mua ở …" / "Bán
   ở …" and takes the tab bar's place while a session runs (CSS on
   `[data-replay-active]`; the tab bar is back on the start form). The
   quantity and stop-loss inputs stay in a card above the bar
   because the design leaves them out but orders need them. The results
   panel and fills table stack below.
9. **Portfolio → Mobile-Portfolio**: title, NAV hero with the equity
   curve, 2 × 2 KPIs, open positions as cards. The desktop tabs stay
   above the tab content and scroll horizontally; wide tables (pending
   orders, journal) scroll inside their own card, never the page.
10. **Quant → Mobile-Quant**: the conversation fills the screen; the
    results panel stacks after the thread; the history sidebar is hidden
    (the new-conversation button stays); the composer stays above the tab
    bar. The voice-input button is not built: there is no speech feature
    to connect it to.
11. **Settings → Mobile-Settings**: the section tabs become a scrolling
    row and the form is one column. Changed while building: Settings
    keeps the tab bar instead of the design's back link to Quant, because
    it is reached from the account menu on every screen and a
    single back target would be wrong for most of them. The
    unsaved-changes footer pins above the tab bar only while there is
    something to save.
12. **Heatmap on phones**: the squarified treemap can't show 40 labels in
    354 px, so below `lg` `/heatmap` shows each sector as a heading plus
    Mobile-Market's 3-column tile grid (the design's own phone heatmap),
    with the same filters in a scrolling row. The side-panel sections
    (breadth, sectors, movers) stack below.
13. **Crypto pages** have no phone designs; they follow the matching
    stock screens (overview ≈ Market, pair ≈ Detail, replay ≈ Replay).
14. **Tap targets** at least 44 px for primary controls (the design's own
    44 px header buttons and 52 px action buttons). Dense data rows are
    links spanning the full row.
15. **Safe areas**: `viewport-fit=cover` plus `env(safe-area-inset-*)`
    padding on the tab bar and the sticky action bars.

## Out of scope

- New features on phones that the desktop lacks (voice input, push
  alerts).
- A native app. This phase is the responsive web app only.

## Verification plan

- A Playwright sweep at 375 × 812 and 390 × 844 (mobile emulation) of
  every screen built in Phases C–I, both signed in and as a guest. Each
  view must show:
  - layout width equal to the screen width (no horizontal overflow);
  - no element under the tab bar at the end of the page;
  - primary controls at least 44 px.
- Flows on a phone:
  - switch markets;
  - open a stock, buy from the bottom bar;
  - run a Replay (step, buy, sell);
  - open the account menu and change portfolio;
  - Quant with a stub model;
  - Settings save;
  - the heatmap filters.
- Desktop regression: the same sweep at 1440 × 960 compared against the
  Phase I screenshots.
- tsc / eslint / build, and Go tests unchanged.

## Verification (done)

All on the production build (`next start`) with the backend on mock data
(this sandbox cannot reach VCI or Binance; see phase-i.md).

- **Phone sweep** (`mobile-audit.js`: 16 routes, guest and signed in,
  mobile emulation) at 390 × 844, 375 × 812 and 768 × 1024. Every view has
  layout width equal to the screen width, with nothing sticking out except
  inside rows that scroll on purpose (filters, timeframes, tabs, wide
  tables). No page errors. It started at 798 px (`/stocks`), 828
  (`/stocks/FPT`), 635 (`/portfolio`), 793 (`/settings/ai`), 798
  (`/crypto`) and 473 (`/crypto/BTCUSDT`).
- **Tap targets**: the remaining sub-32 px hits are checkboxes and the
  range slider inside full-width labels, and the light marketing pages'
  header links (outside this phase). Undersized section links and the
  crypto search input were fixed.
- **Phone flows** (`phase-j-flows.js`, 390 × 844, touch), no page errors:
  - Tab bar shown and sidebar hidden; the market switch works both ways,
    and crypto tabs are Thị trường/Biểu đồ/Replay/Ví/Quant.
  - Heatmap tile → FPT. On Detail the tab bar gives way to the action
    bar. "Bán giấy" scrolls to the ticket with Bán selected, and "Mua
    giấy" with Mua. A market order fills (52,26, fee 7.838 ₫) and shows as
    a holding card on Portfolio.
  - The account menu opens within the screen (18–372 px).
  - Replay HPG: the tab bar is hidden and the buy/sell bar is in view.
    Step ×3, buy 1.000 HPG at 60,39, step, sell, exit, and the tab bar
    returns.
  - Settings: the stub model is configured and the dirty footer is in
    view above the tab bar. After saving, Quant answers "35 mã HOSE đạt cả
    3 điều kiện", the results panel sits inline, and the composer stays in
    view.
  - Heatmap on the phone: HOSE + 1 tháng gives 9 sector tile grids.
- **Desktop regression**: the Phase I browser suite still passes at
  1440 × 960, after updating its selectors to skip the hidden phone copies
  (`:visible`). The 1440 sweep shows no overflow, and the Main/Detail
  screenshots match Phase I's.
- **Fixed after reviewing the screenshots:**
  - white strip under the tab bar (the body background);
  - wrapping timeframe and range labels;
  - clipped market-cap and NAV tiles;
  - a single-point equity curve drawn as a slope;
  - the Quant send button wrapping to its own line;
  - an ambiguous crypto date in the Replay header.
- `tsc`, `eslint`, `npm run build` clean; `go test ./...` passes (one
  backend change: movers now include `companyName`/`exchange` for the
  phone list).

### Not done

- A real device check (iOS Safari safe areas, Android Chrome). Only
  Chromium mobile emulation was available here.
- Voice input, and the design's watchlist button on Detail (see
  decisions 6 and 10).

