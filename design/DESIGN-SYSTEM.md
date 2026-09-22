# VN Stock Sim — design system

A dark trading terminal with a warm cast. Every value below appears in the
artboards; `tokens.css` and `tokens.json` carry the same numbers.

---

## 1. Colour

### Surfaces

| Token | Hex | Used for |
|---|---|---|
| `--bg` | `#0F0F0E` | Page ground |
| `--surface` | `#171715` | Panels, cards, inputs on the page ground |
| `--surface-2` | `#1F1F1C` | Inputs and controls inside panels |
| `--surface-3` | `#23231F` | Active nav item, icon wells, pressed states |
| `--chrome` | `#131311` | Sidebar, icon rail, bottom tab bar |
| `--hairline` | `#23231F` | Dividers inside a panel |
| `--border` | `#2C2C28` | Panel and control borders |
| `--border-strong` | `#3C3C36` | Focused / selected control borders |

The ground is a warm near-black, not blue-black. Panels sit one step up from
the page; controls sit one step up from panels. Nothing is pure black or
pure white.

### Text

| Token | Hex | Used for |
|---|---|---|
| `--text` | `#F3F0E9` | Primary |
| `--text-2` | `#D7D2C7` | Secondary headings, table body |
| `--text-3` | `#A5A19A` | Labels, inactive nav |
| `--text-muted` | `#8A867E` | Captions, helper text — **the lightest text allowed on a panel** (5.0:1) |
| `--text-faint` | `#6F6C63` | Decorative only, never information |

### Brand accent

| Token | Hex | Used for |
|---|---|---|
| `--accent` | `#E08A3C` | Primary buttons, Replay, active mode |
| `--accent-hover` | `#F0A65C` | Link hover |
| `--accent-ink` | `#131210` | **Text on accent fills** — never white |
| `--accent-surface` | `#1C1813` | Accent-tinted cards |
| `--accent-border` | `#4A3521` | Accent-tinted borders |

Amber is scarce on purpose: it means *Replay*, *primary action*, or *the
market you are in*. If three amber things are visible at once, one of them is
wrong.

### Market colours — Vietnamese convention

| Token | Hex | Meaning |
|---|---|---|
| `--up` | `#35C77F` | Price up |
| `--down` | `#FF5C5C` | Price down |
| `--ref` | `#F0C243` | Reference price (unchanged) |
| `--ceiling` | `#C08BFF` | Ceiling — daily upper limit |
| `--floor` | `#4FD3E8` | Floor — daily lower limit |
| `--up-ink` | `#08130D` | Text on a green fill |
| `--down-ink` | `#1A0808` | Text on a red fill |

**Ceiling, reference and floor exist only in the stock market.** Crypto has no
daily price bands, so crypto screens use up / flat / down and nothing else.
This is the single most important rule in the system; see §7.

### Indicator colours

`--sma-20: #E08A3C` · `--sma-50: #7FA2FF` · `--rsi: #C08BFF` · `--macd: #4FD3E8`

### Caution block

`--warn-surface: #1A1713` · `--warn-border: #3A3226` · `--warn-text: #C6BFB1`
with a `--ref` coloured triangle icon. Used for every "this is simulated /
not advice / past ≠ future" notice.

---

## 2. Type

| Role | Family | Weights |
|---|---|---|
| Display | **Lora** (serif) | 600, 700 |
| UI | **Be Vietnam Pro** | 400, 500, 600, 700 |
| Numerals | **IBM Plex Mono** | 400, 500, 600 |

All three carry full Vietnamese diacritics — that is why they were chosen.

**Every number is mono.** Prices, percentages, volumes, quantities, dates in
tables, token counts. This is what makes columns of figures scannable and is
non-negotiable.

Scale in use: 42 / 40 / 30 (auth headings) · 27 / 26 / 25 (page titles,
Lora) · 19–16 (KPI figures, mono) · 15 (panel headings, 600) · 14–13.5
(body, buttons) · 12.5–12 (labels, table cells) · 11.5–11 (captions) ·
10.5–10 (uppercase micro-labels, `letter-spacing: 0.07–0.1em`) · 9 (badges).

Headings use `letter-spacing: -0.015em`. Long prose uses `text-wrap: pretty`.

---

## 3. Shape and space

Radii: `5–7` micro-badges · `8–9` chips and small buttons · `10–12` nav items,
inputs, buttons · `13–14` panels and cards · `16` popovers and large cards ·
`18` the mobile Replay button · `28` phone artboard corners.

Spacing steps: 2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 14, 16, 18, 20, 22, 24, 28.
Panel padding 16–20 desktop, 12–16 mobile. Page padding 24×28 desktop,
18–22 mobile.

Control heights: 30 (inline chip) · 32–34 (small button) · 36 (segmented) ·
40 (nav row) · 44 (**minimum touch target**) · 46–48 (form input) ·
52–54 (primary button, mobile controls).

**Layout is flex/grid with `gap` everywhere** — never margins between
siblings, never whitespace-based spacing.

---

## 4. Components

**Panel** — `background: --surface; border: 1px solid --border; radius: 14px`.
Heading 15px/600 left, meta caption 11.5px `--text-muted` right.

**Nav item (sidebar)** — 40px tall, 10px radius, 11px gap, 18px icon + 13.5px
label. Active: `--text` on `--surface-3`. Inactive: `--text-3` on transparent.

**Icon rail item** — 44×44, 11px radius, 19px icon, same active treatment.
Icon-only, so every one carries an `aria-label`.

**WILL badge** — `margin-left: auto`, 1.5×6 padding, 1px `--border-strong`
border, 5px radius, mono 9px/600, `letter-spacing: 0.12em`, `--text-muted`.
Marks a feature with no screen in this set. Grey, never amber — a planned
feature must not out-shout a built one. Rename to "Sắp có" / "Coming soon"
before it ships to users.

**Data table** — `<table>` with real `<thead>`/`<th scope="col">`. Header:
10–10.5px uppercase `--text-muted`, `letter-spacing: 0.07em`. Rows separated
by `border-top: 1px solid --hairline`, 6.5–9.5px vertical padding. Numeric
columns right-aligned and mono.

**KPI tile** — uppercase micro-label, mono 16–19px figure coloured by
sentiment, 11.5px caption underneath.

**Chart panel** — OHLC readout row in mono 11.5px across the top, "Dữ liệu
mẫu / Sample data" pill on the right, then the SVG.

**Caution block** — warn surface + triangle icon + 11.5–12px `--warn-text`.

---

## 5. Charts

All charts are inline SVG generated from real computed series — no image
assets, no chart library.

- **Candles**: body = `rect` from open to close, wick = `path` M…V…, coloured
  `--up` / `--down`. Body width ≈ 58–62% of the slot.
- **Overlays**: SMA 20 amber, SMA 50 blue, 1.5–1.6px stroke.
- **Volume**: bars under the price plot at 60–65% opacity, same colour rule.
- **RSI(14)**: 30–70 band as a filled `--surface-2` rect with dashed edges.
- **MACD(12,26,9)**: histogram bars plus MACD and signal lines.
- **Grid**: horizontal `--hairline` lines with mono 10.5px right-hand labels.

The indicator maths (EMA, RSI with Wilder smoothing, SMA) is in the
`renderVals()` block of any chart screen and is directly reusable.

---

## 6. Replay Mode — the signature feature

Rules the implementation must honour:

1. Candles after the playhead are **covered by an opaque mask**, not merely
   unrendered — the mask is visible, labelled "TƯƠNG LAI BỊ CHE / FUTURE
   HIDDEN" with a count of the candles left.
2. The boundary is an amber dashed line; the revealed side is at full
   opacity, everything outside dimmed.
3. Controls: step back, play, **next candle** (primary), speed 0.5×–4×, a
   progress scrubber with real dates, and Space as the keyboard shortcut.
4. Entries and exits are drawn as triangles on the bars where they filled,
   with the price beside them.
5. The score is **provisional** until the session ends, and the screen says so.
6. Every order lands in the trade journal automatically.
7. Crypto Replay uses hourly candles, never skips a weekend, and grades risk
   management more strictly.

---

## 7. Stock mode vs crypto mode

One app, one switch (segmented control at the top of the sidebar; STK/CRY
pair on the icon rail). The layout never moves — only the content changes.

| | Stocks | Crypto |
|---|---|---|
| Instrument | Ticker (FPT) | Pair (BTC/USDT) |
| Session | Hours + status | 24/7, no session |
| Price bands | Ceiling / reference / floor | **None** |
| Order types | LO, MP, ATC, Stop | Market, Limit, Stop, OCO |
| Size | Shares, lots | Coin amount + USDT value |
| Depth | 3-level bước giá | 8-level book with cumulative depth bars |
| Facts | P/E, P/B, ROE, EPS | Market cap, supply, ATH distance, volatility |
| Money | ₫ (prices in thousands) | USDT |

---

## 8. Localisation

Two locales, identical layouts. Vietnamese labels run 15–30% longer than
English, so the Vietnamese build is the tight case — if it fits in VI it fits
in EN.

| | Vietnamese | English |
|---|---|---|
| Numbers | `1.412,58` | `1,412.58` |
| Dates | `22/09/2026` | `22/09/2026` (**DD/MM kept** — it is a Vietnamese exchange) |
| Currency | `1.086.420.000 ₫` | `1,086,420,000 ₫` |
| Short forms | tr / tỷ / nghìn tỷ | M / bn / T |

Never translated: ticker symbols, exchange names (HOSE, HNX, UPCOM), order
types (LO, MP, ATC). Ceiling/Reference/Floor translate as words but keep
their colours.

Glossary: Tổng quan → Overview · Bản đồ nhiệt → Heatmap · Giao dịch giấy →
Paper trading · Sổ giao dịch → Trade journal · Vị thế → Positions · Giá vốn →
Avg cost · Sụt giảm tối đa → Max drawdown · Hệ số lợi nhuận → Profit factor ·
Trợ lý Quant → Quant assistant · Bước giá → Order book.

---

## 9. Accounts

- **Guest first.** Replay, charts and the screener run with no account. The
  sign-in prompt fires at the **first paper order** — the first thing that
  needs somewhere to live.
- **Email + password only.** Accounts are created manually; no social
  providers in this build. If they are added, the buttons go above the
  divider and the guest button stays below it.
- **Starting capital is chosen at sign-up** (1 tỷ ₫ / 500 triệu ₫ /
  10.000 USDT) because it is the denominator of every return the app will
  ever show that user.
- **One person, several paper portfolios.** The account menu switches
  between them; portfolios sit above profile and settings in that menu.
- No ID documents, no brokerage connection, no fees — stated on the sign-up
  screen, not buried in terms.

---

## 10. The AI assistant (Quant)

- Show the **parsed conditions** as editable chips before showing results.
  The user corrects the condition, not the prompt.
- Every answer ends in a product action: backtest, Replay, save, watchlist.
- **Re-test generated rules on the same window they came from**, and show
  that step in the pipeline.
- The settings screen lists exactly **what data leaves the device**, as
  toggles. Paper balance and private journal notes default to **off**.
- "Let Quant place paper orders by itself" defaults to **off**.
- Every strategy result carries the caution block: sample data, past ≠
  future, not investment advice, paper account only.

---

## 11. Accessibility

- Real `<button>`, `<a href>`, `<input>` + `<label>` everywhere — no
  `onClick` divs, even in static mockups.
- Icon-only controls carry `aria-label`; charts carry `role="img"` and a
  description of what they show.
- Text contrast ≥ 4.5:1. `--text-muted` (#8A867E) is the floor; anything
  lighter is decoration.
- Dark fills take dark ink (`--accent-ink`, `--up-ink`, `--down-ink`), never
  white.
- Touch targets ≥ 44px on phone layouts.
- No fake status bars or fake keyboards on the phone artboards.
