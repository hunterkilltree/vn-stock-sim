// Server-only forwarding for /api/quant/* (phase-h.md decision 11). A
// Route Handler rather than a Server Action because `next dev` prints
// Server Action arguments to the terminal, and these requests carry the
// user's model API key. Nothing here logs the body.
import type { NextRequest } from "next/server";
import { getActivePortfolio, getSessionToken } from "./session";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

function jsonError(status: number, code: string, message: string) {
  return Response.json({ code, message }, { status });
}

export async function forwardQuant(req: NextRequest, path: "test" | "chat"): Promise<Response> {
  // The session cookie is SameSite=lax, so cross-site POSTs don't carry
  // it anyway; this also refuses same-browser requests from other origins.
  const origin = req.headers.get("origin");
  if (origin) {
    let originHost = "";
    try {
      originHost = new URL(origin).host;
    } catch {
      originHost = "";
    }
    if (originHost !== req.headers.get("host")) {
      return jsonError(403, "forbidden", "Cross-origin request refused.");
    }
  }
  if (!req.headers.get("content-type")?.includes("application/json")) {
    return jsonError(415, "unsupported_media_type", "Expected application/json.");
  }

  const token = await getSessionToken();
  if (!token) return jsonError(401, "unauthorized", "Bạn cần đăng nhập để dùng Trợ lý Quant.");

  let body: Record<string, unknown>;
  try {
    body = (await req.json()) as Record<string, unknown>;
  } catch {
    return jsonError(400, "invalid_json", "Invalid JSON body.");
  }

  if (path === "chat") {
    try {
      body.portfolioId = (await getActivePortfolio(token)).active?.id ?? "";
    } catch {
      body.portfolioId = "";
    }
  }

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1/quant/${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return jsonError(502, "backend_unreachable", "Không thể kết nối tới máy chủ VN Stock Sim.");
  }
  return new Response(await res.text(), { status: res.status, headers: { "Content-Type": "application/json" } });
}
