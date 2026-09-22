# VN Stock Sim — design export

Everything needed to build this product in a Cowork / Claude Code session:
40 artboard sources, the design tokens, and the written rules behind the
screens.

```
vn-stock-sim-design/
├── README.md              ← you are here
├── DESIGN-SYSTEM.md       ← tokens, components, the rules that hold it together
├── SCREENS.md             ← every screen: what it is, what's on it, what's missing
├── tokens.css             ← CSS custom properties, ready to drop in
├── tokens.json            ← same values as data (Tailwind config, JS themes)
├── canvas.json            ← artboard positions on the design canvas
└── screens/               ← 40 artboard sources (.dc.html)
```

## How to use this in Cowork / Claude Code

1. Put this folder in your repo (`design/` is a good home) and commit it.
2. Open a session in that repo.
3. Point at one screen at a time. Screens are self-contained — a session
   never needs to read all 40.

Prompts that work well:

> Read `design/DESIGN-SYSTEM.md` and `design/tokens.css`. Set up a React +
> Vite project using these tokens as CSS variables. Don't build screens yet.

> Build the market overview screen from `design/screens/Main.dc.html`.
> Match the layout and spacing exactly; use the tokens from `tokens.css`
> instead of the hardcoded hex values in the artboard. Sample data can stay
> hardcoded in a `fixtures.ts` for now.

> `design/screens/Replay.dc.html` is the spec for Replay Mode. Implement the
> candle masking and the playback controls. The rules for what the user may
> not see are in `DESIGN-SYSTEM.md` under "Replay Mode".

> Compare `design/screens/Main.dc.html` and `design/screens/Main-EN.dc.html`
> and set up i18n with those two locales. The formatting rules are in
> `DESIGN-SYSTEM.md` under "Localisation".

## About the `.dc.html` format

Each screen is one self-contained HTML file: markup with inline styles, plus
a `<script type="text/x-dc">` block at the bottom holding a small class that
returns the screen's data (`renderVals()`).

For a coding session, read them as **specs, not as code to port**:

- The markup gives you exact structure, spacing, sizes and colours.
- `{{holes}}` mark where data goes. `<sc-for list="{{x}}">` is a loop.
- The `renderVals()` block shows the shape of the data each screen needs —
  useful for designing your API types.
- The candle / RSI / MACD maths in those blocks is real and can be lifted
  directly; it is plain JavaScript with no dependencies.

Ignore `<x-dc>`, `<helmet>` and `./support.js` — those belong to the design
tool, not to your app.

## Fonts

Three Google Fonts, all with full Vietnamese support:

```html
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Be+Vietnam+Pro:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500;600&family=Lora:wght@600;700&display=swap">
```

## One warning about the data

Every number in these screens is **sample data** — plausible, internally
consistent, and invented. Ticker symbols are real Vietnamese listings, but
the prices, volumes and fundamentals attached to them are not. Wire real
data in before showing this to anyone who might trade on it.
