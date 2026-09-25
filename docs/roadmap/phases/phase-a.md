# Phase A — planning and analysis

Scope per FULL-APP-PLAN.md section 3. Foundation only: design tokens +
two nav shells, applied to the three existing pages. No content redesign
yet (that is Phases C/D).

## Current state (read before writing code)

- `frontend/src/app/layout.tsx` — root layout puts `<Navbar/>` (the
  light, luxalgo-style top bar) around every page unconditionally, body
  is `bg-white text-neutral-900`, fonts are Geist Sans/Mono only.
- `frontend/src/app/globals.css` — only two CSS vars (`--background`,
  `--foreground`), both light-theme. No dark tokens, no VN price colors,
  no Lora/Be Vietnam Pro/IBM Plex Mono.
- Pages today: `/` (marketing home, Navbar), `/login`, `/register`
  (Navbar, light forms), `/stocks` (list, Navbar), `/stocks/[symbol]`
  (detail + chart + AI insight, Navbar), `/chart` (TradingView embed,
  Navbar). All six currently share one light theme and one nav.
- No icon library is installed (`package.json` has only
  lightweight-charts/next/react) — matches this repo's existing habit of
  avoiding extra dependencies (hand-rolled auth tokens instead of a JWT
  lib). Nav icons will be small inline SVGs, not a new package.

## Decisions for this phase (flagging, not asking, where the plan already answered it)

1. **Do not delete the light "luxalgo" palette.** FULL-APP-PLAN.md
   section 14 flagged this as needing confirmation. Resolution: keep
   `Navbar`/`Button`/the light palette exactly as-is for the marketing
   surface (`/`, `/login`, `/register` — none of these are in the design
   canvas's 20 screens, they are this repo's own pre-existing pages), and
   add the new dark tokens and nav shells as an *additional*, separate
   system used only by the app screens the canvas actually designed
   (`/stocks`, `/stocks/[symbol]`, `/chart` today; `/portfolio`,
   `/replay`, `/quant`, `/settings` once later phases build them). This
   avoids a destructive rewrite and matches "extend, do not replace" from
   the plan's Phase A step 2.
2. **Route naming: keep `/stocks` as-is for now**, don't rename to
   `/market` in this phase. FULL-APP-PLAN.md Phase C is where the Main
   screen actually gets rebuilt and is the right place to decide the
   final route name; renaming now would just mean renaming again later.
3. **Nav items not yet built (Portfolio, Replay, Quant, Settings)** get
   real entries in the shared nav list so the shell is complete, but
   render as disabled (no real route yet, would 404). This is distinct
   from the canvas's own "WILL" badge, which is reserved for the true
   Phase K backlog (Screener, standalone Heatmap, Strategy Builder,
   standalone Backtesting UI, standalone Trade Journal) — those get the
   actual muted "WILL" pill; Portfolio/Replay/Quant/Settings get a
   plainer "soon" treatment since they are scheduled (Phases D-H), not
   backlog.

## Files to add

- `frontend/src/lib/navItems.ts` — single source of truth: `{ href,
  label, icon, kind: "built" | "soon" | "will" }[]`.
- `frontend/src/components/icons.tsx` — ~8 small inline SVG icon
  components (market, detail/chart, portfolio, replay, quant, settings,
  logo mark, avatar placeholder). No new dependency.
- `frontend/src/components/SidebarNav.tsx` — 236px full sidebar (logo,
  labeled nav items, spacer, placeholder avatar button at the bottom).
- `frontend/src/components/RailNav.tsx` — 72px icon-only rail (logo
  mark, icon-only nav items with a tooltip via `title=`, placeholder
  avatar button at the bottom).

## Files to change

- `frontend/src/app/globals.css` — add a second token block (not
  replacing the existing `--background`/`--foreground` pair) for the
  dark app-shell palette: `--app-bg #0F0F0E`, `--app-card #171715`,
  `--app-border #2C2C28`, `--app-hover #23231F`, `--app-surface #1A1A17`,
  `--app-accent #E08A3C`, and the five VN price colors
  `--price-up #35C77F`, `--price-down #FF5C5C`, `--price-ref #F0C243`,
  `--price-ceiling #C08BFF`, `--price-floor #4FD3E8`. Exposed as Tailwind
  utilities via `@theme inline` (`bg-app-bg`, `text-price-up`, etc.).
- `frontend/src/app/layout.tsx` — load `Lora`, `Be_Vietnam_Pro`, and
  `IBM_Plex_Mono` from `next/font/google` alongside the existing Geist
  fonts (additive: exposed as `--font-lora`/`--font-be-vietnam`/
  `--font-plex-mono` CSS vars, not swapping the default body font, so the
  marketing pages keep looking exactly as they do today). Remove the
  unconditional `<Navbar/>` from the root layout -- Navbar moves to being
  called explicitly by the three marketing pages themselves.
- `frontend/src/app/page.tsx`, `frontend/src/app/login/page.tsx`,
  `frontend/src/app/register/page.tsx` -- each gains an explicit
  `<Navbar/>` at the top of its own JSX (replacing what the root layout
  used to provide), so these three pages render identically to before.
- `frontend/src/app/stocks/page.tsx` -- wrap content with `<SidebarNav/>`
  in a flex row instead of `<Navbar/>`.
- `frontend/src/app/stocks/[symbol]/page.tsx`,
  `frontend/src/app/chart/page.tsx` -- wrap content with `<RailNav/>` in a
  flex row instead of `<Navbar/>`.

## Verification plan

1. `npx tsc --noEmit && npx eslint .` clean.
2. `next build` -- confirm the route table still lists the same six
   routes with the same static/dynamic classification as today (no
   regressions from moving Navbar out of the root layout).
3. Browser pane, dev server: `/`, `/login`, `/register` look pixel-same
   as before (Navbar present, light theme). `/stocks` shows the new
   236px dark `SidebarNav` on the left with the existing light content
   still to its right (expected to look mismatched until Phase C -- that
   mismatch itself is the confirmation this phase did not silently
   reskin content it wasn't scoped to touch). `/stocks/[symbol]` and
   `/chart` show the 72px dark `RailNav` the same way.
4. Full Docker rebuild (`docker compose build && ./run.sh -d`) + curl
   against `/`, `/stocks`, `/stocks/VNM`, `/chart` to confirm no runtime
   crash from the layout change, matching this repo's standing habit of
   verifying inside the real image, not just the dev server.
5. Record the outcome in RESUME.md, same format as every previous
   completed unit of work in this repo.

## Verification (done)

- frontend/src/app/globals.css gained a second, additive token block
  (does not touch the existing --background/--foreground pair used by
  the light luxalgo palette): --app-bg #0F0F0E, --app-card #171715,
  --app-border #2C2C28, --app-hover #23231F, --app-surface #1A1A17,
  --app-accent #E08A3C, --app-fg #EDEDEA, plus the five VN price colors
  (--price-up/down/ref/ceiling/floor), all exposed as Tailwind utilities
  via @theme inline (bg-app-bg, text-price-up, etc.).
- frontend/src/app/layout.tsx loads Lora, Be Vietnam Pro, and IBM Plex
  Mono from next/font/google alongside the existing Geist fonts
  (additive -- font-serif now resolves to Lora via @theme, the
  marketing pages are unaffected since they never used font-serif).
- New frontend/src/components/icons.tsx (8 small inline SVGs -- no icon
  package added, matching this repo's existing habit of avoiding extra
  dependencies), frontend/src/lib/navItems.ts (single source of truth
  nav list shared by both shells: Market/Chart built; Portfolio/Replay/
  Quant/Settings marked "soon" -- disabled, not linked, since those
  routes don't exist until Phases D-H), frontend/src/components/
  SidebarNav.tsx (236px full sidebar) and RailNav.tsx (72px icon rail).
- Wired in: /stocks now uses SidebarNav, /stocks/[symbol] and /chart now
  use RailNav -- content on all three is untouched (still the old light
  styling), only the nav strip changed, exactly as scoped.

Hit a real build failure fixing this, not just a style choice: the
original plan was "each of /, /login, /register calls <Navbar/>
directly" after removing Navbar from the root layout. /login and
/register are Client Components ("use client"); Navbar is an async
Server Component that reads cookies() via session.ts. A Client
Component importing a Server Component directly breaks the client/
server boundary -- `next build` failed with "You're importing a module
that depends on next/headers ... in the Pages Router" (misleading
wording; the real cause was the boundary violation, not the Pages
Router, confirmed via the import-trace Turbopack printed). Fixed by
moving /, /login, /register into a frontend/src/app/(marketing)/ route
group (URLs unchanged -- route groups don't affect the path) with its
own layout.tsx that renders <Navbar/> once for the whole group, so the
two client pages never import Navbar themselves.

Verified, not just read: `npx tsc --noEmit`, `npx eslint .`, and
`npm run build` all clean after the fix. The build's route table still
lists the same six routes with no regressions -- and /chart actually
improved from "ƒ Dynamic" to "○ Static", since it no longer depends
(transitively, via a root-layout Navbar) on cookies(). Checked visually
in the browser pane (dev server, backend not running): / and /login
render pixel-identical to before (Navbar, light theme, confirmed via
screenshot); /stocks shows the new dark SidebarNav with the active
"Market" item highlighted and Portfolio/Replay/Quant/Settings correctly
greyed out with "Soon" pills, existing content (including the
backend-unreachable error message) unchanged to its right; /chart shows
the new dark RailNav with the active icon highlighted. Also checked at
375x812: the fixed-width shells overflow the viewport horizontally at
that width -- expected and NOT a bug, responsive collapsing to a bottom
tab bar is explicitly Phase J's scope, not this phase's.

Re-verified inside the actual Docker image once Docker Desktop finished
starting: `docker compose build` succeeded for both images (frontend
build's route table matched the standalone build exactly), `docker
compose up -d`, then curl against /, /stocks, /stocks/VNM, /chart, and
/login all returned HTTP 200, /stocks' HTML contained the expected
SidebarNav labels (Market/Portfolio/Replay/Quant/Settings) and the
"Soon" pill text, and GET /api/v1/symbols returned real mock data from
the backend container. Merged as PR #13.
