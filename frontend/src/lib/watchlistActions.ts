"use server";

import { revalidatePath } from "next/cache";
import { getSessionToken } from "./session";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

// Adds or removes one symbol (a stock ticker or a pair like BTCUSDT) --
// the Detail bookmark and the Main watchlist card (phase-k.md decision 12).
export async function setWatchedAction(symbol: string, watched: boolean): Promise<{ error: string | null }> {
  const token = await getSessionToken();
  if (!token) return { error: "Bạn cần đăng nhập để theo dõi mã." };
  let res: Response;
  try {
    res = watched
      ? await fetch(`${API_BASE_URL}/api/v1/watchlist`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
          body: JSON.stringify({ symbol }),
          cache: "no-store",
        })
      : await fetch(`${API_BASE_URL}/api/v1/watchlist/${encodeURIComponent(symbol)}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
  } catch {
    return { error: "Không thể kết nối tới máy chủ." };
  }
  if (!res.ok) return { error: res.status === 404 ? "Không tìm thấy mã này." : "Cập nhật danh sách theo dõi thất bại." };
  revalidatePath("/stocks", "layout");
  revalidatePath("/crypto", "layout");
  return { error: null };
}
