"use server";

import { getSessionToken } from "./session";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type BacktestInput = {
  symbol: string;
  rule: "ema_crossover" | "rsi_reversion";
  params: Record<string, number>;
  years: number;
  startingCapital: number;
};

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

// Runs one backtest for the /backtest page (phase-k.md decision 14) on
// the same POST /backtests engine Quant uses; the page then shows it by id.
export async function runBacktestPageAction(input: BacktestInput): Promise<{ id: string | null; error: string | null }> {
  const token = await getSessionToken();
  if (!token) return { id: null, error: "Bạn cần đăng nhập để kiểm thử." };
  const symbol = input.symbol.trim().toUpperCase();
  if (!/^[A-Z0-9]{2,10}$/.test(symbol)) return { id: null, error: "Mã cổ phiếu không hợp lệ." };

  // The mock market generates bars for any string, so check the ticker
  // exists before spending a run on it.
  try {
    const check = await fetch(`${API_BASE_URL}/api/v1/symbols/${encodeURIComponent(symbol)}`, { cache: "no-store" });
    if (check.status === 404) return { id: null, error: `Không tìm thấy mã ${symbol}.` };
  } catch {
    return { id: null, error: "Không thể kết nối tới máy chủ." };
  }

  const to = new Date();
  const from = new Date(to);
  from.setUTCFullYear(to.getUTCFullYear() - Math.min(Math.max(Math.round(input.years), 1), 10));

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/backtests`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        symbol,
        from: isoDate(from),
        to: isoDate(to),
        startingCapital: input.startingCapital,
        rule: { type: input.rule, params: input.params },
      }),
      cache: "no-store",
    });
  } catch {
    return { id: null, error: "Không thể kết nối tới máy chủ." };
  }
  const payload = await res.json().catch(() => null);
  if (!res.ok) return { id: null, error: payload?.message ?? "Kiểm thử thất bại." };
  return { id: payload.id as string, error: null };
}
