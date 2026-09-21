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

async function apiFetch<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    // The backend has no real-time push yet (RESUME.md), so short revalidation
    // is a reasonable default for V1 rather than fully static or no-store.
    next: { revalidate: 10 },
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
