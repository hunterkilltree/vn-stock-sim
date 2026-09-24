import type { NextRequest } from "next/server";
import { forwardQuant } from "@/lib/quantProxy";

export async function POST(req: NextRequest) {
  return forwardQuant(req, "test");
}
