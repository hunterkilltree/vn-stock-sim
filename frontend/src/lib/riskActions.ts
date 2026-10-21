"use server";

import { revalidatePath } from "next/cache";
import { getActivePortfolio, getSessionToken } from "./session";
import type { RiskAssessment, RiskMode, RiskSettings } from "./api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type CheckInput = {
  symbol: string;
  type: string;
  quantity: number;
  price: number;
  stopLoss: number;
  target: number;
};

// The order ticket's dry run (phase-risk-rules.md design item 1): the
// same rules POST /orders applies, against the active portfolio, so the
// warnings show before the user clicks Mua. null when it can't be run
// (signed out, invalid quantity, server unreachable): the ticket then
// shows no findings rather than a wrong one.
export async function checkOrderAction(input: CheckInput): Promise<RiskAssessment | null> {
  const token = await getSessionToken();
  if (!token || input.quantity <= 0) return null;
  try {
    const portfolioId = (await getActivePortfolio(token)).active?.id;
    const res = await fetch(`${API_BASE_URL}/api/v1/orders/check`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ portfolioId, side: "buy", ...input }),
      cache: "no-store",
    });
    if (!res.ok) return null;
    return (await res.json()) as RiskAssessment;
  } catch {
    return null;
  }
}

export type RiskSettingsState = { error: string | null; saved: boolean };

// Settings → Tài khoản giấy: saves the active portfolio's thresholds.
export async function saveRiskSettingsAction(_prev: RiskSettingsState, formData: FormData): Promise<RiskSettingsState> {
  const token = await getSessionToken();
  if (!token) return { error: "Bạn cần đăng nhập.", saved: false };
  const num = (k: string) => Number(String(formData.get(k) ?? "").replace(",", "."));
  const body: RiskSettings = {
    mode: (formData.get("mode") === "strict" ? "strict" : "coach") as RiskMode,
    riskPercent: num("riskPercent"),
    maxRiskPercent: num("maxRiskPercent"),
    minRewardRisk: num("minRewardRisk"),
  };
  const portfolioId = String(formData.get("portfolioId") ?? "");
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/portfolios/${encodeURIComponent(portfolioId)}/risk-settings`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return { error: "Không thể kết nối tới máy chủ.", saved: false };
  }
  if (!res.ok) {
    const payload = await res.json().catch(() => null);
    return { error: payload?.message ?? "Không lưu được.", saved: false };
  }
  revalidatePath("/settings/paper-account");
  revalidatePath("/stocks/[symbol]", "page");
  return { error: null, saved: true };
}
