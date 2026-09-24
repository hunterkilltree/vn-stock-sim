// Server-only session helpers. The backend issues a bearer JWT
// (api-spec.md: POST /auth/login -> { accessToken, expiresIn, user }); we
// store it in an httpOnly cookie so it is never exposed to client-side JS
// and every authenticated fetch happens server-side (Server Components /
// Server Actions), the same pattern the rest of the app already uses for
// data fetching (see api.ts). Importing next/headers already restricts
// this module to server-only use -- Next.js errors at build time if a
// Client Component imports it.
import { cookies } from "next/headers";
import { getMe, getPortfolios, type Portfolio, type User } from "./api";

export const SESSION_COOKIE = "vss_token";
export const ACTIVE_PORTFOLIO_COOKIE = "vss_portfolio";
export const ACTIVE_CRYPTO_COOKIE = "vss_portfolio_crypto";

export type Market = "stock" | "crypto";

export function activeCookieFor(market: Market): string {
  return market === "crypto" ? ACTIVE_CRYPTO_COOKIE : ACTIVE_PORTFOLIO_COOKIE;
}

// The cookie is only a preference (phase-g.md decision 1): it's honored
// only if it names one of the caller's own trading portfolios of this
// market, otherwise the market's first trading portfolio wins. Each market
// has its own cookie, so switching a crypto wallet never changes the
// stock portfolio (phase-i.md decision 9). Replay portfolios are never
// selectable -- live orders must not reach them.
export async function getActivePortfolio(
  token: string,
  market: Market = "stock",
): Promise<{ active: Portfolio | null; portfolios: Portfolio[]; all: Portfolio[] }> {
  const all = (await getPortfolios(token)).data.filter((p) => p.kind === "trading");
  const portfolios = all.filter((p) => p.market === market);
  const wanted = (await cookies()).get(activeCookieFor(market))?.value;
  const active = portfolios.find((p) => p.id === wanted) ?? portfolios[0] ?? null;
  return { active, portfolios, all };
}

export async function getSessionToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(SESSION_COOKIE)?.value ?? null;
}

// Re-verifies the token against the backend on every call (GET /auth/me)
// rather than trusting a locally-decoded JWT -- V1 has no token
// revocation, so this is the only way to notice an expired/invalid
// cookie and treat the user as logged out.
export async function getSessionUser(): Promise<User | null> {
  const token = await getSessionToken();
  if (!token) return null;
  try {
    return await getMe(token);
  } catch {
    return null;
  }
}
