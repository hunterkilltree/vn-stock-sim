"use server";

import { getSessionToken } from "./session";
import type { ReplaySession } from "./api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type ReplayActionResult = { session: ReplaySession | null; error: string | null };

// Every Replay mutation (start/advance/order/end) goes through a Server
// Action rather than a client-side fetch, same reasoning as
// orderActions.ts: the session token lives in an httpOnly cookie
// invisible to client JS, and inside Docker the browser can't resolve
// the "backend" hostname the server-side fetch uses. A fast auto-play
// loop just means more of these round-trips, not a different mechanism.
async function replayFetch(path: string, method: "GET" | "POST", body?: object): Promise<ReplayActionResult> {
  const token = await getSessionToken();
  if (!token) {
    return { session: null, error: "Bạn cần đăng nhập để dùng Chế độ Replay." };
  }

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: body ? JSON.stringify(body) : undefined,
      cache: "no-store",
    });
  } catch {
    return { session: null, error: "Không thể kết nối tới máy chủ." };
  }

  const payload = await res.json().catch(() => null);
  if (!res.ok) {
    return { session: null, error: payload?.message ?? "Yêu cầu thất bại. Vui lòng thử lại." };
  }
  return { session: payload as ReplaySession, error: null };
}

export async function startReplayAction(
  symbol: string,
  market: "stock" | "crypto" = "stock",
  totalBars?: number,
  startDate?: string,
): Promise<ReplayActionResult> {
  return replayFetch("/api/v1/replay/sessions", "POST", { symbol, market, totalBars, startDate });
}

export async function advanceReplayAction(sessionId: string): Promise<ReplayActionResult> {
  return replayFetch(`/api/v1/replay/sessions/${encodeURIComponent(sessionId)}/advance`, "POST");
}

export async function placeReplayOrderAction(
  sessionId: string,
  side: "buy" | "sell",
  quantity: number,
  stopLoss?: number,
): Promise<ReplayActionResult> {
  return replayFetch(`/api/v1/replay/sessions/${encodeURIComponent(sessionId)}/orders`, "POST", {
    side,
    quantity,
    stopLoss: stopLoss ?? 0,
  });
}

export async function endReplayAction(sessionId: string): Promise<ReplayActionResult> {
  return replayFetch(`/api/v1/replay/sessions/${encodeURIComponent(sessionId)}/end`, "POST");
}
