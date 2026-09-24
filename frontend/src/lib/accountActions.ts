"use server";

import { cookies } from "next/headers";
import { revalidatePath } from "next/cache";
import { activeCookieFor, getSessionToken, type Market } from "./session";
import type { Portfolio } from "./api";
import { CRYPTO_CAPITAL_PRESET, isStockCapitalPreset } from "./capital";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

async function setActiveCookie(id: string, market: Market) {
  (await cookies()).set(activeCookieFor(market), id, {
    httpOnly: true,
    secure: false,
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
  });
}

// Ownership is checked (GET /portfolios/:id is user-scoped) before the
// cookie is written, and getActivePortfolio re-checks on every read.
export async function setActivePortfolioAction(id: string): Promise<{ error: string | null; market?: Market }> {
  const token = await getSessionToken();
  if (!token) return { error: "Bạn cần đăng nhập." };

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/portfolios/${encodeURIComponent(id)}`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
  } catch {
    return { error: "Không thể kết nối tới máy chủ." };
  }
  if (!res.ok) return { error: "Không tìm thấy danh mục." };
  const p = (await res.json()) as Portfolio;
  if (p.kind !== "trading") return { error: "Danh mục Replay không dùng để giao dịch." };

  const market: Market = p.market === "crypto" ? "crypto" : "stock";
  await setActiveCookie(p.id, market);
  revalidatePath("/", "layout");
  return { error: null, market };
}

export async function createPortfolioAction(
  name: string,
  startingCapital: number,
  market: Market = "stock",
): Promise<{ error: string | null; market?: Market }> {
  const token = await getSessionToken();
  if (!token) return { error: "Bạn cần đăng nhập." };
  const trimmed = name.trim();
  if (!trimmed) return { error: "Vui lòng đặt tên danh mục." };
  const validCapital = market === "crypto" ? startingCapital === CRYPTO_CAPITAL_PRESET.amount : isStockCapitalPreset(startingCapital);
  if (!validCapital) return { error: "Vui lòng chọn vốn ảo ban đầu." };

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/portfolios`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name: trimmed, market, startingCapital }),
      cache: "no-store",
    });
  } catch {
    return { error: "Không thể kết nối tới máy chủ." };
  }
  const payload = await res.json().catch(() => null);
  if (!res.ok) return { error: payload?.message ?? "Tạo danh mục thất bại." };

  await setActiveCookie((payload as Portfolio).id, market);
  revalidatePath("/", "layout");
  return { error: null, market };
}
