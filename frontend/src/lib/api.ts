// Thin client for the VN Stock Sim backend (backend/, Go + Gin, see
// api-spec.md for the full contract). Server Components call these
// directly; NEXT_PUBLIC_API_BASE_URL lets the same code run against a
// deployed API later.
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type Symbol = {
  symbol: string;
  companyName: string;
  exchange: string;
  sector: string;
  tickSize: number;
};

export type SymbolDetail = Symbol & {
  lastPrice: number;
  change: number;
  changePercent: number;
  marketCap: number;
  peRatio: number;
  pbRatio: number;
  eps: number;
  dividendYield: number;
};

type Paged<T> = {
  data: T[];
  meta: { page: number; pageSize: number; totalItems: number; totalPages: number };
};

async function apiFetch<T>(path: string, opts?: { token?: string }): Promise<T> {
  const headers: HeadersInit = {};
  if (opts?.token) {
    headers.Authorization = `Bearer ${opts.token}`;
  }
  const res = await fetch(`${API_BASE_URL}${path}`, {
    // no-store, not next: { revalidate }: revalidate let Next.js
    // statically prerender /stocks at `next build` time, which in Docker
    // means fetching before the backend container exists -- that build-time
    // fetch always failed and baked the error page into the image, only
    // self-healing after the first background ISR revalidation. no-store
    // forces this route to render per-request instead (see RESUME.md).
    cache: "no-store",
    headers,
  });
  if (!res.ok) {
    throw new Error(`API ${path} failed: ${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export function searchSymbols(query = ""): Promise<Paged<Symbol>> {
  const qs = query ? `?q=${encodeURIComponent(query)}` : "";
  return apiFetch<Paged<Symbol>>(`/api/v1/symbols${qs}`);
}

export function getSymbolDetail(symbol: string): Promise<SymbolDetail> {
  return apiFetch<SymbolDetail>(`/api/v1/symbols/${encodeURIComponent(symbol)}`);
}

export type Bar = {
  time: number; // unix seconds, matches lightweight-charts' UTCTimestamp
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
};

export type IndicatorPoint = {
  time: number;
  value: number;
};

// from/to are unix seconds, per api-spec.md's GET /market/bars contract.
export function getBars(symbol: string, resolution: string, from: number, to: number): Promise<{ data: Bar[] }> {
  const qs = `symbol=${encodeURIComponent(symbol)}&resolution=${encodeURIComponent(resolution)}&from=${from}&to=${to}`;
  return apiFetch<{ data: Bar[] }>(`/api/v1/market/bars?${qs}`);
}

export function getIndicator(
  symbol: string,
  resolution: string,
  indicator: string,
  period: number,
  from: number,
  to: number,
): Promise<{ data: IndicatorPoint[] }> {
  const qs = `symbol=${encodeURIComponent(symbol)}&resolution=${encodeURIComponent(resolution)}&indicator=${indicator}&period=${period}&from=${from}&to=${to}`;
  return apiFetch<{ data: IndicatorPoint[] }>(`/api/v1/market/indicators?${qs}`);
}

export type InsightSignal = {
  label: string;
  detail: string;
  direction: "bullish" | "bearish" | "neutral";
};

export type Insight = {
  symbol: string;
  summary: string;
  signals: InsightSignal[];
  // "rule-based" today; see RESUME.md -- a real LLM call needs an API key
  // and has a real per-call cost, a decision left to the user.
  source: string;
  generatedAt: string;
};

export function getInsight(symbol: string): Promise<Insight> {
  return apiFetch<Insight>(`/api/v1/symbols/${encodeURIComponent(symbol)}/insight`);
}

export type User = {
  id: string;
  email: string;
  displayName: string;
};

// GET /auth/me, authenticated. Used by session.ts to re-verify a stored
// token is still valid rather than trusting a locally-decoded JWT.
export function getMe(token: string): Promise<User> {
  return apiFetch<User>("/api/v1/auth/me", { token });
}
