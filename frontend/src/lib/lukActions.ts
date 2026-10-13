"use server";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

// POST /symbols/:symbol/luk-review/ai -- Gemini's commentary on the Martin
// Luk checklist (phase-luk-review.md). Public, no sign-in needed; the
// Gemini key itself only ever lives on the backend.
export async function lukAIReviewAction(
  symbol: string,
): Promise<{ text: string; model: string; error: null } | { error: string }> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/symbols/${encodeURIComponent(symbol)}/luk-review/ai`, {
      method: "POST",
      cache: "no-store",
      signal: AbortSignal.timeout(70_000),
    });
  } catch {
    return { error: "Không thể kết nối tới máy chủ." };
  }
  if (res.status === 503) return { error: "Máy chủ chưa cấu hình GEMINI_API_KEY." };
  if (!res.ok) return { error: "AI chưa trả lời được (kiểm tra GEMINI_API_KEY / hạn mức Gemini), thử lại sau." };
  const body = (await res.json()) as { text: string; model: string };
  return { text: body.text, model: body.model, error: null };
}
