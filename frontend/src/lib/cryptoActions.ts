"use server";

import { cookies } from "next/headers";
import { revalidatePath } from "next/cache";
import { ACTIVE_CRYPTO_COOKIE, getActivePortfolio, getSessionToken } from "./session";
import type { Portfolio } from "./api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type CryptoOrderState = {
  error: string | null;
  success: { status: string; filledPrice?: number; fee?: number; quantity?: number } | null;
};

// Places a paper order on the user's active crypto wallet (its own
// cookie -- phase-i.md decision 9). The wallet is resolved server-side,
// never taken from the form.
export async function placeCryptoOrderAction(_prev: CryptoOrderState, formData: FormData): Promise<CryptoOrderState> {
  const token = await getSessionToken();
  if (!token) return { error: "Bạn cần đăng nhập để đặt lệnh giấy.", success: null };

  let walletId: string | undefined;
  try {
    walletId = (await getActivePortfolio(token, "crypto")).active?.id;
  } catch {
    return { error: "Không thể kết nối tới máy chủ.", success: null };
  }
  if (!walletId) return { error: "Bạn chưa có ví crypto.", success: null };

  const body = {
    portfolioId: walletId,
    symbol: String(formData.get("symbol") ?? ""),
    side: String(formData.get("side") ?? "buy"),
    type: String(formData.get("type") ?? "market"),
    quantity: Number(formData.get("quantity") ?? 0),
    price: Number(formData.get("price") ?? 0),
    stopPrice: Number(formData.get("stopPrice") ?? 0),
  };
  if (!body.symbol || !(body.quantity > 0)) return { error: "Vui lòng nhập số lượng hợp lệ.", success: null };

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/orders`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return { error: "Không thể kết nối tới máy chủ.", success: null };
  }
  const payload = await res.json().catch(() => null);
  if (!res.ok) return { error: payload?.message ?? "Đặt lệnh thất bại.", success: null };

  revalidatePath("/crypto", "layout");
  return { error: null, success: { status: payload.status, filledPrice: payload.filledPrice, fee: payload.fee, quantity: payload.quantity } };
}

// Opens the design's 10.000 USDT crypto wallet and makes it active.
export async function openCryptoWalletAction(): Promise<{ error: string | null }> {
  const token = await getSessionToken();
  if (!token) return { error: "Bạn cần đăng nhập." };
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/portfolios`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name: "Ví crypto", market: "crypto", startingCapital: 10_000 }),
      cache: "no-store",
    });
  } catch {
    return { error: "Không thể kết nối tới máy chủ." };
  }
  const payload = await res.json().catch(() => null);
  if (!res.ok) return { error: payload?.message ?? "Không mở được ví." };
  (await cookies()).set(ACTIVE_CRYPTO_COOKIE, (payload as Portfolio).id, {
    httpOnly: true,
    secure: false,
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
  });
  revalidatePath("/", "layout");
  return { error: null };
}
