# Phase G — Account menu, portfolio switcher, Settings shell, Signup capital picker

Plan written against `design/screens/Account-Menu.dc.html`,
`design/screens/Settings-AI.dc.html` (its rail + subnav only; the AI
Model form itself is Phase H) and `design/screens/Signup.dc.html` (the
capital picker and market-interest selector that FULL-APP-PLAN.md's
screen inventory assigns to G), per FULL-APP-PLAN.md section 9 and
RESUME.md's "Plan" entry.

## What exists already

- Backend multi-portfolio API since Phase B: `GET/POST /portfolios`,
  `GET /portfolios/:id/{summary,positions,equity-history,allocation,stats}`.
- Every frontend screen reads **one** portfolio: `/stocks` and the Detail
  order ticket use the singular `/portfolio` (the lazily-created
  default), `/portfolio` uses `portfolios[0]`. Nothing lets the user pick.
- Phase F creates one dedicated portfolio per Replay session through the
  same `CreatePortfolio`, so `GET /portfolios` already returns them mixed
  in with the user's real paper-trading portfolios, with nothing to tell
  them apart.
- `RailNav` and the Main header both render a disabled avatar button
  ("Menu tài khoản — sắp ra mắt"). `settingsItem` links to
  `/settings/ai`, which 404s.

## Design decisions (made before code)

1. **The active portfolio lives in an httpOnly cookie** (`vss_portfolio`),
   set by a Server Action, exactly as FULL-APP-PLAN.md section 9 says, so
   Server Components read it with no client fetch. The cookie is only a
   *preference*: `getActivePortfolio(token)` (session.ts) lists the
   caller's portfolios and uses the cookie's ID only if it's actually in
   that list, otherwise falls back to the first trading portfolio. A
   stale cookie (after logout/login as someone else, or a deleted
   in-memory store after a backend restart) can therefore never point a
   page at a portfolio the user doesn't own. `setActivePortfolioAction`
   also checks ownership (`GET /portfolios/:id`) before writing the
   cookie. `logoutAction` clears it.
2. **Every portfolio-reading screen switches to the active portfolio**:
   `/stocks` (Paper account card, open positions, sidebar balance),
   `/stocks/[symbol]` (order ticket's buying power), `/portfolio` (all
   tabs + pending orders), and `placeOrderAction` sends the active
   `portfolioId` (read server-side from the cookie, not from a hidden
   form field the browser could edit). That is the plan's stated output:
   "switching portfolios changes which portfolio the rest of the app
   operates against."
3. **Backend must check `portfolioId` ownership on order placement.**
   Pre-existing hole found while planning this: `order.Service.Create`
   books a fill against whatever `portfolioId` the request names, with
   no check that the caller owns it — any logged-in user could trade
   inside someone else's portfolio by guessing `pf_N`. It was latent
   while the frontend never sent a `portfolioId`; Phase G starts sending
   one, so fix it now: `order.PortfolioResolver` gains
   `GetPortfolio(userID, id)`, unknown/foreign IDs return 404
   `portfolio_not_found`.
4. **Portfolios get a `kind`: `"trading"` or `"replay"`.** Phase F's
   decision 4 gave each Replay session its own portfolio precisely so
   historical-price fills never mix with live-price fills. If the
   switcher could make a Replay portfolio *active*, the next live order
   from the Detail ticket would land in it and recreate exactly that
   mix. So: Replay creates portfolios through a new
   `Service.CreateReplayPortfolio` (kind `"replay"`), the switcher and
   `getActivePortfolio` only consider `"trading"` portfolios, and
   `order.Service.Create` rejects live orders into a replay portfolio
   (409 `replay_portfolio`) as a backend guarantee, not just a hidden
   menu entry. This revises phase-f.md's "the replay portfolio will show
   up in the switcher once Phase G exists" note — a Replay session's
   results stay on the Replay screen; listing past sessions is its own
   (not yet designed) feature.
5. **"First trading portfolio a user creates becomes their default."**
   Today the default is only ever created lazily by `DefaultFor` with a
   hardcoded 100,000,000 VND. For the Signup capital picker to mean
   anything, the user's main portfolio has to open with the chosen
   capital. Rather than teach `auth` about portfolios (it has no
   dependency on `portfolio` and shouldn't gain one), `registerAction`
   calls `POST /portfolios` ("Danh mục chính", chosen capital) right
   after registering, and the store marks the first *trading* portfolio
   a user creates as their default. `DefaultFor`'s lazy path is
   unchanged for everyone else (existing users, a failed follow-up
   call). A Replay portfolio never becomes the default.
6. **Signup capital options follow the design**: "1 tỷ ₫ · Mặc định"
   (selected), "500 triệu ₫ · Sát vốn thật", and "10.000 USDT · Cho
   crypto" rendered disabled with a note — there is no crypto market
   until Phase I, and a USDT portfolio would have nothing to trade.
   The backend's own fallback default stays 100,000,000 VND (unchanged,
   still what the lazy path and Replay use).
7. **Market interest is stored, not decorative.** Signup's "Cổ phiếu /
   Crypto / Cả hai" selector is saved on the user (`auth.User.
   marketInterest`, optional on `POST /auth/register`, returned by
   `/auth/me`). Nothing consumes it yet (Phase I will), but a control
   that silently discards its value would be a fake.
8. **Creating a portfolio from the menu** ("Tạo danh mục mới") opens an
   inline form inside the popover — name + the same capital presets as
   Signup (stock only, same Phase I reason) — then posts through a Server
   Action and makes the new portfolio active. The design draws only the
   button, not a form; an inline expansion avoids inventing a modal
   (design/SCREENS.md lists modals as not yet designed).
9. **Account-menu entry points.** The design draws the avatar button on
   Main's header and at the bottom of the icon rail. Both become real
   triggers. The Portfolio page's header gets one too (small, flagged
   deviation — Portfolio.dc.html has none, but that screen is where
   switching matters most, and the sidebar has no avatar of its own).
   Data for the menu (user, trading portfolios, each one's NAV and P&L
   vs. its starting capital) is fetched server-side by an async
   `AccountMenuButton` Server Component that renders the Client
   `AccountMenu` popover — passed into the Client `RailNav` as a prop,
   which is how a Server Component crosses into a Client Component
   without breaking the boundary Phase A tripped over.
10. **Menu links**: "Cài đặt" → `/settings/ai` (real); "Hồ sơ & bảo mật"
    and "Trợ giúp" keep the design's own WILL badges and link to the
    matching Settings placeholder (`/settings/account`) or render
    disabled (Help — no route or design exists).
11. **Settings shell**: `app/settings/layout.tsx` = RailNav (Settings
    active) + the 232px subnav from Settings-AI.dc.html (8 sections,
    exact labels/order/WILL flags) + the "Tài khoản · Gói Cá nhân ·
    miễn phí" card. `/settings/ai` renders the header ("Mô hình AI" + its
    subtitle) and a plain note that the provider/key form lands in Phase
    H — no fake inputs that save nothing. The other seven sections are a
    dynamic `/settings/[section]` route rendering a WILL placeholder;
    unknown slugs 404. `/settings` redirects to `/settings/ai`.
12. **RailNav gains the Settings icon** at the bottom (above the avatar),
    as in Settings-AI.dc.html's rail, active on any `/settings/*` path.
13. **Signup moves out of the `(marketing)` route group** to
    `app/register` (URL unchanged): Signup.dc.html is a full-screen dark
    split layout with no top navbar, while the marketing group wraps
    every page in the light luxalgo `Navbar`. `/login` stays where it is
    (Login's redesign isn't this phase).

## Backend changes

- `portfolio`: `Portfolio.Kind`; store `Create` takes a kind and records
  the user's default on their first trading portfolio; `DefaultFor`
  reuses that; `Service.CreateReplayPortfolio`.
- `order`: `PortfolioResolver.GetPortfolio`; `Create` validates
  ownership (`ErrPortfolioNotFound` → 404) and kind
  (`ErrReplayPortfolio` → 409).
- `replay`: `PortfolioPort.CreatePortfolio` → `CreateReplayPortfolio`.
- `auth`: optional `marketInterest` on register, on `User`.
- Postman: register body gains `marketInterest`; a negative "order into
  a foreign/unknown portfolio → 404" request.

## Frontend changes

- `lib/session.ts`: `ACTIVE_PORTFOLIO_COOKIE`, `getActivePortfolio`.
- `lib/accountActions.ts`: `setActivePortfolioAction`,
  `createPortfolioAction`.
- `lib/authActions.ts`: register sends `marketInterest` then creates the
  main portfolio with the chosen capital; logout clears the portfolio
  cookie.
- `components/AccountMenu.tsx` (client popover),
  `components/AccountMenuButton.tsx` (server data loader).
- `RailNav` (account slot + Settings icon), Main/Portfolio headers.
- `app/settings/{layout,page}.tsx`, `app/settings/ai/page.tsx`,
  `app/settings/[section]/page.tsx`, `components/SettingsSubnav.tsx`.
- `app/register/page.tsx` rebuilt against Signup.dc.html.

## Verification (done)

- `go build ./... && go vet ./...` clean; `gofmt -l` lists only the two
  files that were already non-compliant before this phase
  (`backtest/types.go`, `market/provider.go`; `auth/store.go` was a third,
  formatted here since it was being edited anyway). `npx tsc --noEmit`,
  `npx eslint .`, `npm run build` clean; the route table gains
  `/settings`, `/settings/ai`, `/settings/[section]`, and `/register` is
  now static.
- curl against the real backend: user B ordering into user A's
  `portfolioId` → 404 `portfolio_not_found`, and A's cash unchanged; a
  live order into B's Replay portfolio → 409 `replay_portfolio`; A's
  first `POST /portfolios` (500,000,000) became A's default
  (`GET /portfolio` showed 500,000,000); for B, a Replay session started
  *before* B's first real portfolio did not become the default (the lazy
  default still opened as a separate `trading` portfolio); invalid
  `marketInterest` → 400; valid value round-trips through `/auth/me`.
- Newman, full collection: 43 requests, 0 failures, including the two
  new status-asserting requests (unknown portfolio → 404, live order into
  a Replay portfolio → 409).
- Real browser (Playwright + headless Chromium against the production
  build via `next start`, and the real `go` backend): signed up through
  the new Signup screen choosing 500 triệu ₫ + "Cổ phiếu" (submitting
  without the acknowledgement checkbox was blocked first) → sidebar
  balance 500.000.000 ₫; opened the account menu from Main's avatar (one
  portfolio, active); created "Thử chiến lược RSI" at 1 tỷ from the
  menu → landed on /portfolio with that name in the header and
  1.000.000.000 ₫ in the sidebar; Detail ticket read "Danh mục: Thử
  chiến lược RSI"; placed a real MP buy of 100 VNM; switched back to
  "Danh mục chính" from the menu → /portfolio showed that name and no
  VNM, and the Detail ticket then named the main portfolio. Confirmed
  independently via the API as that same user: VNM x100 sits only in the
  RSI portfolio, the main one is empty, and the order record carries the
  RSI portfolio's ID. Started a Replay session, then opened the menu
  from the rail's avatar (right-side placement) → still exactly the two
  trading portfolios, no "Replay HPG…" row; Escape closes it. `/settings`
  redirects to `/settings/ai`; clicked every one of the 8 subnav items,
  each rendered its own heading with `aria-current="page"` on the right
  item; `/settings/bogus` → 404. Logged out from the menu → both the
  session and active-portfolio cookies gone. No page errors other than
  that deliberate 404.
- Found while testing, not a product bug: under `next dev`, Next's own
  dev-tools badge sits bottom-left over the rail's avatar button and
  intercepts clicks there. It doesn't exist in production builds, which
  is why the browser run above used `next start`.

## Verification plan (as written before code)

- `go build ./... && go vet ./...`, `gofmt -l`; `tsc`, `eslint`,
  `npm run build`.
- curl: order into another user's portfolio → 404; into a replay
  portfolio → 409; register + create-first-portfolio → it is the default
  (`GET /portfolio` shows the chosen capital).
- Newman: full collection, 0 failures.
- Real browser (Playwright): sign up choosing 500 triệu → sidebar
  balance shows 500.000.000 ₫; open the account menu, create a second
  portfolio, confirm it becomes active (Main, Portfolio, Detail ticket
  all switch); buy on the Detail ticket and confirm the fill landed in
  the active portfolio and not the other; switch back and confirm the
  first portfolio is untouched; start a Replay session and confirm its
  portfolio does not appear in the menu; click through every Settings
  subnav item.
