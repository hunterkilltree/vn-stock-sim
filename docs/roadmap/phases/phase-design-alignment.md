# Design-system alignment — planning and analysis

The user supplied an authoritative design export at
`C:\Users\USER\Downloads\vn-stock-sim-design\vn-stock-sim-design\`
(README.md, DESIGN-SYSTEM.md, SCREENS.md, tokens.css, tokens.json,
canvas.json, 40 `.dc.html` screen sources) with the instruction "this app
have to follow design." This is a precise, machine-readable version of
the same design canvas read earlier in this session via the Claude
artifact browser (same 20 screens x VI/EN, same screen names) -- but with
exact hex values, exact token names, and an exact type/spacing/radius
scale, where the earlier artifact-reading pass only captured approximate
values by eye. This is a correction pass on Phase A's foundation before
Phase C builds real screen content on top of it, not a new lettered
phase in FULL-APP-PLAN.md's sequence.

## What was already right vs. wrong in Phase A

Comparing `frontend/src/app/globals.css`'s Phase A tokens against
`tokens.css`:

**Already exact** (no change needed): the five VN price colors --
`up #35C77F`, `down #FF5C5C`, `ref #F0C243`, `ceiling #C08BFF`,
`floor #4FD3E8` all match `tokens.css` byte-for-byte. `--app-accent
#E08A3C` matches `--accent`. `--app-bg #0F0F0E` matches `--bg`.
`--app-border #2C2C28` matches `--border`. The three Google Fonts (Lora,
Be Vietnam Pro, IBM Plex Mono) are already loaded correctly via
`next/font/google` in `layout.tsx` -- no font changes needed.

**Wrong or missing:**

1. `--app-card #171715` matches `--surface` correctly, but was named
   "card" instead of "surface" -- cosmetic, but worth renaming to reduce
   friction reading the design docs against this code going forward.
2. `--app-hover #23231F` happens to equal BOTH `--surface-3` (active nav
   background) and `--hairline` (divider) in the real system -- an
   accidental match, not a designed one; these are two different
   semantic roles that need two different token names even though they
   currently share a hex value.
3. `--app-surface #1A1A17` was an invented value with no counterpart in
   the real system at all -- the real `--surface-2` is `#1F1F1C`, a
   different number entirely.
4. `--app-fg #EDEDEA` (invented) vs. the real `--text: #F3F0E9` --
   different value, wrong by a few percent, but wrong.
5. **Missing entirely:** `--chrome` (`#131311`, distinct from `--bg` --
   this is the real bug worth fixing: DESIGN-SYSTEM.md section 4 says
   sidebar/icon-rail/tab-bar use `--chrome`, not `--bg`. Phase A's
   `SidebarNav`/`RailNav` were built using `bg-app-bg`, i.e. the same
   flat color as the page ground -- visually wrong, the nav should read
   as a slightly darker "chassis" than the page), `--text-2`, `--text-3`,
   `--text-muted`, `--text-faint`, `--border-strong`, `--accent-hover`,
   `--accent-ink`, `--accent-surface`, `--accent-border`, `--up-ink`,
   `--down-ink`, the three wash colors, all four indicator colors
   (`--sma-20/50`, `--rsi`, `--macd`), and the three caution-block colors.
6. `SidebarNav`/`RailNav`'s active-nav-item styling used `text-app-accent`
   for the active label -- DESIGN-SYSTEM.md section 4 is explicit:
   "Active: `--text` on `--surface-3`", i.e. the active item's TEXT stays
   primary-white, only the background changes. Accent is reserved for
   "primary buttons, Replay, active mode" (section 1) -- using it for
   ordinary nav-item active state was exactly the kind of accent overuse
   section 1 warns against ("If three amber things are visible at once,
   one of them is wrong").
7. The "Soon" pill (Phase A's stand-in for undesigned/upcoming nav items)
   didn't match the spec's real `.badge-will` component at all: spec is
   mono font, 9px, 600 weight, `0.12em` tracking, 5px radius,
   `--border-strong` border, `--text-muted` color, `margin-left: auto`.
   Phase A's version used a generic sans pill, 10px, full radius,
   `--app-border`. Also worth adopting the spec's own English string for
   it, "WILL", instead of "Soon" -- DESIGN-SYSTEM.md names it "the WILL
   badge" and SCREENS.md's "Not designed yet" section is literally
   titled "carries the WILL badge".

## Decisions

1. **Commit the design export into the repo**, per the export's own
   README ("Put this folder in your repo (`design/` is a good home) and
   commit it"), at `design/`. It currently lives only in the user's
   Downloads folder, outside any repo -- every later phase (C onward)
   needs to read `design/screens/*.dc.html` as the literal spec, so it
   has to be durable, versioned, and not dependent on a path outside the
   project.
2. **Token naming: keep the existing `app-` prefix, correct the values.**
   Renaming to the export's bare names (`--bg`, `--surface`, ...) would
   collide with nothing today, but `bg-bg`/`text-text` as Tailwind
   utility class names reads worse than `bg-app-bg`/`text-app-text` for
   anyone (human or future session) editing this code -- the `app-`
   prefix also keeps these visually distinct at a glance from the
   existing light-theme `--background`/`--foreground` pair used by the
   marketing pages. Every hex value is corrected to match `tokens.css`
   exactly; only the CSS variable *names* keep the `app-` prefix.
3. **`SidebarNav`/`RailNav` switch to `bg-app-chrome`** for their own
   background (was `bg-app-bg`), and their active-nav-item styling
   switches from accent-colored text to `text-app-text` on
   `bg-app-surface-3`, matching DESIGN-SYSTEM.md section 4 exactly.
4. **The WILL badge becomes a real shared component**
   (`frontend/src/components/WillBadge.tsx`) matching `.badge-will`'s
   exact CSS (mono, 9px/600, 0.12em tracking, 5px radius,
   `border-app-border-strong`, `text-app-text-muted`, `margin-left:
   auto`), replacing the ad hoc "Soon"/"Will" pill markup duplicated in
   `SidebarNav`/`RailNav`. Per DESIGN-SYSTEM.md's own distinction (WILL
   = "a feature with no screen in this set" vs. this app's own
   Portfolio/Replay/Quant/Settings nav items, which DO have designed
   screens and are simply not built yet) -- the badge text for
   Portfolio/Replay/Quant/Settings stays a plain, non-WILL "disabled"
   treatment (still greyed out, still unclickable, but without claiming
   they're undesigned when they are designed and simply pending a later
   phase). Only true backlog items (screener, standalone heatmap,
   strategy builder, etc., per SCREENS.md's own list) would ever get the
   real WILL badge, and none of those have nav entries yet.
5. **FULL-APP-PLAN.md's design-system section (section 1) gets a pointer
   update**, not a rewrite: add a note that `design/DESIGN-SYSTEM.md` and
   `design/tokens.css` are now the authoritative source (superseding the
   earlier artifact-reading summary), so later phases (C onward) read the
   committed `design/` folder directly instead of this plan's own prose
   summary of it.

## Files to add

- `design/` -- the entire design export, copied verbatim from the user-
  supplied path (README.md, DESIGN-SYSTEM.md, SCREENS.md, tokens.css,
  tokens.json, canvas.json, screens/*.dc.html -- 40 files). Copied, not
  retyped, so nothing drifts from the source.
- `frontend/src/components/WillBadge.tsx`.

## Files to change

- `frontend/src/app/globals.css` -- replace the Phase A `--app-*` token
  block with a corrected, complete one matching every value in
  `design/tokens.css` (prefixed `--app-*`, values exact).
- `frontend/src/components/SidebarNav.tsx`,
  `frontend/src/components/RailNav.tsx` -- `bg-app-chrome` for the shell
  background; active nav item text becomes `text-app-text` (not accent);
  inactive becomes `text-app-text-3` (was `text-neutral-300`); disabled/
  upcoming items use `text-app-text-muted`-ish treatment distinct from
  the true `WillBadge`.
- `FULL-APP-PLAN.md` -- add a short pointer note in section 1 to
  `design/DESIGN-SYSTEM.md`/`design/tokens.css` as canonical.

## Verification plan

1. `npx tsc --noEmit && npx eslint . && npm run build` all clean.
2. Browser-pane check on `/stocks` and `/chart`: `SidebarNav`/`RailNav`
   render visibly darker than the page background (chrome vs. bg are
   close but distinct -- `#131311` vs `#0F0F0E`), the active nav item
   shows white text on a lighter grey background (not amber text), and
   the WILL-style badge on Portfolio/Replay/Quant/Settings renders in
   the small mono uppercase style from the spec.
3. Full Docker rebuild + curl, matching this repo's standing habit.
4. Record the outcome in RESUME.md.

## Verification (done)

- Copied the entire export into the repo at `design/` (40 screen files +
  4 docs), per the export's own README instruction to do exactly that --
  now the durable, versioned source every later phase (C onward) reads
  directly, not an external Downloads path.
- Corrected `frontend/src/app/globals.css`'s Phase A `--app-*` token
  block against `design/tokens.css` byte-for-byte. The five VN price
  colors and `--app-accent`/`--app-bg`/`--app-border` were already
  exact; `--app-card`/`--app-hover`/`--app-surface`/`--app-fg` were
  renamed and corrected (`--app-surface` had been an invented value,
  `#1A1A17`, with no counterpart in the real system -- the real
  `--surface-2` is `#1F1F1C`). Added everything that was missing
  entirely: `--app-chrome` (`#131311`, the real bug -- `SidebarNav`/
  `RailNav` had been using the page-ground color for their own
  background instead of this distinct, slightly darker "chassis"
  color), `--app-text-2/3/muted/faint`, `--app-border-strong`,
  `--app-accent-hover/ink/surface/border`, `--price-up-ink/down-ink`,
  the four indicator colors (`--app-sma-20/50`, `--app-rsi`,
  `--app-macd`), and the three caution-block colors.
- `SidebarNav.tsx`/`RailNav.tsx`: background switched from `bg-app-bg`
  to `bg-app-chrome`; active-nav-item styling switched from
  accent-colored text to `design/DESIGN-SYSTEM.md` section 4's actual
  rule ("Active: `--text` on `--surface-3`") -- accent had been
  overused for an ordinary nav state, when the design system explicitly
  reserves amber for "primary buttons, Replay, active mode" and warns
  "if three amber things are visible at once, one of them is wrong."
- New `frontend/src/components/WillBadge.tsx`, matching the spec's real
  `.badge-will` component exactly (mono 9px/600, 0.12em tracking, 5px
  radius, `--border-strong` border, `--text-muted` text). Per
  DESIGN-SYSTEM.md's own distinction, this is reserved for true backlog
  items with no screen at all (design/SCREENS.md's "Not designed yet"
  list) -- Portfolio/Replay/Quant/Settings are designed screens simply
  not built yet, so their disabled nav state stays a plain muted
  treatment without the badge, which no longer falsely claims those
  screens don't exist.
- FULL-APP-PLAN.md section 1 gained a pointer note marking `design/` as
  the design-system source of truth going forward, superseding that
  section's own prose (written from the earlier, approximate
  artifact-reading pass).

Verified: `npx tsc --noEmit`, `npx eslint .`, `npm run build` all clean,
route table unchanged. Browser-pane check (dev server, backend running):
`/stocks`'s SidebarNav and `/chart`'s RailNav both visibly read as a
distinct, slightly darker chassis than the page background; the active
nav item shows white text on a lighter grey background, not amber text;
Portfolio/Replay/Quant/Settings show plain muted text with no badge.
Re-verified inside the actual rebuilt Docker image: `docker compose
build frontend && docker compose up -d`, then curl against /, /stocks,
/stocks/VNM, /chart, /login all returned HTTP 200, and the served HTML
contains the `bg-app-chrome` utility class.

Branch: design-system-alignment (off master, after PR #14 merged).
Merged as PR #15.
