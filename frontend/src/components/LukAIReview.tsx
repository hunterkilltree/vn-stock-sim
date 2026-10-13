"use client";

import { useState, useTransition } from "react";
import { lukAIReviewAction } from "@/lib/lukActions";

// The "Nhờ AI nhận xét" button under the Martin Luk checklist: one Gemini
// call on demand, never on page load (it spends the server's quota).
export default function LukAIReview({ symbol }: { symbol: string }) {
  const [pending, start] = useTransition();
  const [result, setResult] = useState<{ text: string; model: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  function run() {
    setError(null);
    start(async () => {
      const r = await lukAIReviewAction(symbol);
      if (r.error !== null) setError(r.error);
      else setResult({ text: r.text, model: r.model });
    });
  }

  return (
    <div className="flex flex-col gap-2">
      {!result && (
        <button
          type="button"
          onClick={run}
          disabled={pending}
          className="h-8 rounded-[9px] border border-app-border bg-app-surface-2 text-[11.5px] font-medium text-app-accent hover:bg-app-surface-3 disabled:opacity-60"
        >
          {pending ? "Gemini đang đọc checklist…" : "Nhờ AI (Gemini) nhận xét"}
        </button>
      )}
      {error && <p className="m-0 text-[11px] text-price-down">{error}</p>}
      {result && (
        <div className="flex flex-col gap-1 rounded-[11px] border border-app-hairline bg-app-surface-2 p-[9px_11px]">
          <span className="text-[10px] uppercase tracking-[0.04em] text-app-text-faint">Nhận xét AI · {result.model}</span>
          <p className="m-0 whitespace-pre-line text-[11.5px] leading-[1.5] text-app-text-2">{result.text}</p>
        </div>
      )}
    </div>
  );
}
