"use server";

import { revalidatePath } from "next/cache";
import { getSessionToken } from "./session";
import type { BacktestResult, QuantStrategy } from "./quantTypes";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

// Runs a Quant strategy draft through the existing POST /backtests engine
// (FULL-APP-PLAN.md section 10.4) -- the numbers shown on the strategy
// card come from here, never from the model.
export async function runBacktestAction(
  symbol: string,
  strategy: QuantStrategy,
  years: number,
): Promise<{ result: BacktestResult | null; error: string | null }> {
  const token = await getSessionToken();
  if (!token) return { result: null, error: "Bạn cần đăng nhập." };
  if (strategy.kind === "none") return { result: null, error: "Chưa có chiến lược để kiểm thử." };

  const to = new Date();
  const from = new Date(to);
  from.setUTCFullYear(to.getUTCFullYear() - Math.min(Math.max(years, 1), 10));

  const params: Record<string, number> =
    strategy.kind === "ema_crossover"
      ? { fast: strategy.fast, slow: strategy.slow }
      : {
          period: 14,
          entry: Math.round(strategy.rsiEntry),
          exit: Math.round(strategy.rsiExit),
          stopLossPercent: Math.round(strategy.stopLossPercent),
          trendSma: strategy.trendSma,
        };

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/backtests`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        symbol,
        from: isoDate(from),
        to: isoDate(to),
        startingCapital: 100_000_000,
        rule: { type: strategy.kind, params },
      }),
      cache: "no-store",
    });
  } catch {
    return { result: null, error: "Không thể kết nối tới máy chủ." };
  }
  const payload = await res.json().catch(() => null);
  if (!res.ok) return { result: null, error: payload?.message ?? "Kiểm thử thất bại." };
  return { result: payload as BacktestResult, error: null };
}

export async function addToWatchlistAction(symbols: string[]): Promise<{ added: number; error: string | null }> {
  const token = await getSessionToken();
  if (!token) return { added: 0, error: "Bạn cần đăng nhập." };
  let added = 0;
  for (const symbol of symbols.slice(0, 50)) {
    try {
      const res = await fetch(`${API_BASE_URL}/api/v1/watchlist`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ symbol }),
        cache: "no-store",
      });
      if (res.ok) added++;
    } catch {
      return { added, error: "Không thể kết nối tới máy chủ." };
    }
  }
  revalidatePath("/stocks");
  return { added, error: null };
}
