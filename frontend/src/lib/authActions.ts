"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { SESSION_COOKIE } from "./session";

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

  const result = await postAuth("/api/v1/auth/register", { email, password, displayName });
  if (!result.ok) {
    return { error: result.message };
  }
  await setSessionCookie(result.data);
  redirect("/stocks");
}

export async function logoutAction() {
  (await cookies()).delete(SESSION_COOKIE);
  redirect("/");
}
