// Server-only session helpers. The backend issues a bearer JWT
// (api-spec.md: POST /auth/login -> { accessToken, expiresIn, user }); we
// store it in an httpOnly cookie so it is never exposed to client-side JS
// and every authenticated fetch happens server-side (Server Components /
// Server Actions), the same pattern the rest of the app already uses for
// data fetching (see api.ts). Importing next/headers already restricts
// this module to server-only use -- Next.js errors at build time if a
// Client Component imports it.
import { cookies } from "next/headers";
import { getMe, type User } from "./api";

export const SESSION_COOKIE = "vss_token";

export async function getSessionToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(SESSION_COOKIE)?.value ?? null;
}

// Re-verifies the token against the backend on every call (GET /auth/me)
// rather than trusting a locally-decoded JWT -- V1 has no token
// revocation, so this is the only way to notice an expired/invalid
// cookie and treat the user as logged out.
export async function getSessionUser(): Promise<User | null> {
  const token = await getSessionToken();
  if (!token) return null;
  try {
    return await getMe(token);
  } catch {
    return null;
  }
}
