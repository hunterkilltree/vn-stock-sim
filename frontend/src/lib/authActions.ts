"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ACTIVE_CRYPTO_COOKIE, ACTIVE_PORTFOLIO_COOKIE, SESSION_COOKIE } from "./session";
import { CRYPTO_CAPITAL_PRESET, isStockCapitalPreset } from "./capital";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type AuthFormState = { error: string | null };

type AuthResponse = {
  accessToken: string;
  expiresIn: number;
  user: { id: string; email: string; displayName: string };
};

// Not routed through api.ts's apiFetch: these need the raw response body
// on failure (the backend's { code, message } error shape) to show a
// useful message on the form, which apiFetch's generic error doesn't
// preserve.
async function postAuth(path: string, body: unknown): Promise<{ ok: true; data: AuthResponse } | { ok: false; message: string }> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return { ok: false, message: "Could not reach the server. Is the backend running?" };
  }
  if (!res.ok) {
    const errBody = await res.json().catch(() => null);
    return { ok: false, message: errBody?.message ?? "Something went wrong. Please try again." };
  }
  return { ok: true, data: (await res.json()) as AuthResponse };
}

// httpOnly so the token is never exposed to client-side JS; every
// authenticated request happens server-side (see api.ts/session.ts).
// secure: false because this demo runs over plain HTTP (no TLS
// configured anywhere in docker-compose.yml) -- flip to true behind
// HTTPS in a real deployment.
async function setSessionCookie(data: AuthResponse) {
  (await cookies()).set(SESSION_COOKIE, data.accessToken, {
    httpOnly: true,
    secure: false,
    sameSite: "lax",
    path: "/",
    maxAge: data.expiresIn,
  });
}

export async function loginAction(_prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  const result = await postAuth("/api/v1/auth/login", { email, password });
  if (!result.ok) {
    return { error: result.message };
  }
  await setSessionCookie(result.data);
  redirect("/stocks");
}

export async function registerAction(_prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  const displayName = String(formData.get("displayName") ?? "").trim();
  const marketInterest = String(formData.get("marketInterest") ?? "both");
  const startingCapital = Number(formData.get("startingCapital") ?? 0);

  if (formData.get("acknowledge") !== "on") {
    return { error: "Vui lòng xác nhận bạn hiểu đây là công cụ mô phỏng." };
  }
  const cryptoStart = startingCapital === CRYPTO_CAPITAL_PRESET.amount;
  if (!isStockCapitalPreset(startingCapital) && !cryptoStart) {
    return { error: "Vui lòng chọn vốn ảo ban đầu." };
  }

  const result = await postAuth("/api/v1/auth/register", { email, password, displayName, marketInterest });
  if (!result.ok) {
    return { error: result.message };
  }
  await setSessionCookie(result.data);

  // The user's first stock trading portfolio becomes their default
  // (phase-g.md decision 5); choosing 10.000 USDT opens a crypto wallet
  // instead and the stock default opens lazily at 100,000,000 VND
  // (phase-i.md decision 9). If this call fails the account still works
  // -- don't fail the signup over it.
  await fetch(`${API_BASE_URL}/api/v1/portfolios`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${result.data.accessToken}` },
    body: JSON.stringify(
      cryptoStart
        ? { name: "Ví crypto", market: "crypto", startingCapital }
        : { name: "Danh mục chính", market: "stock", startingCapital },
    ),
    cache: "no-store",
  }).catch(() => null);

  redirect(cryptoStart ? "/crypto" : "/stocks");
}

export async function logoutAction() {
  const store = await cookies();
  store.delete(SESSION_COOKIE);
  store.delete(ACTIVE_PORTFOLIO_COOKIE);
  store.delete(ACTIVE_CRYPTO_COOKIE);
  redirect("/");
}
