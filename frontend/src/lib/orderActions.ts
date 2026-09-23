"use server";

import { revalidatePath } from "next/cache";
import { getSessionToken } from "./session";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type OrderFormState = {
  error: string | null;
  success: { status: string; filledPrice?: number; fee?: number } | null;
};

// Same rationale as authActions.ts's postAuth: not routed through
// api.ts's apiFetch, because a failed order needs the backend's raw
// { code, message } body (e.g. "not enough virtual cash") shown on the
// form, which apiFetch's generic thrown Error doesn't preserve.
export async function placeOrderAction(_prevState: OrderFormState, formData: FormData): Promise<OrderFormState> {
  const token = await getSessionToken();
  if (!token) {
    return { error: "Bạn cần đăng nhập để đặt lệnh giấy.", success: null };
  }

  const body = {
    symbol: String(formData.get("symbol") ?? ""),
    side: String(formData.get("side") ?? "buy"),
    type: String(formData.get("type") ?? "market"),
    quantity: Number(formData.get("quantity") ?? 0),
  };

  if (!body.symbol || body.quantity <= 0) {
    return { error: "Vui lòng nhập khối lượng hợp lệ.", success: null };
  }

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
  if (!res.ok) {
    return { error: payload?.message ?? "Đặt lệnh thất bại. Vui lòng thử lại.", success: null };
  }

  // The order ticket's own buying-power figure and any portfolio views
  // elsewhere on this route need to reflect the new cash/position state
  // on next render -- revalidatePath forces that instead of showing a
  // stale balance until the next unrelated navigation.
  revalidatePath("/stocks/[symbol]", "page");

  return {
    error: null,
    success: { status: payload.status, filledPrice: payload.filledPrice, fee: payload.fee },
  };
}
