"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

// Crypto-Main.dc.html's "Tìm cặp: BTC/USDT, SOL…" box: pick from the
// app's pairs and open its detail page.
export default function CryptoSearch({ pairs }: { pairs: { symbol: string; base: string; name: string }[] }) {
  const router = useRouter();
  const [q, setQ] = useState("");
  function go(e: React.FormEvent) {
    e.preventDefault();
    const needle = q.trim().toUpperCase().replace(/[\s/-]/g, "");
    const hit = pairs.find((p) => p.symbol === needle || p.base === needle || p.name.toUpperCase() === q.trim().toUpperCase());
    if (hit) router.push(`/crypto/${hit.symbol}`);
  }
  return (
    <form onSubmit={go} className="flex h-11 items-center gap-2 rounded-[11px] border border-app-border bg-app-surface px-[14px]">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#8A867E" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
        <circle cx="11" cy="11" r="7" />
        <path d="M16.5 16.5L21 21" />
      </svg>
      <label htmlFor="cq" className="sr-only">Tìm cặp giao dịch</label>
      <input
        id="cq"
        list="crypto-pairs"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Tìm cặp: BTC/USDT, SOL…"
        className="w-[200px] border-0 bg-transparent text-[13px] text-app-text outline-none placeholder:text-app-text-muted"
      />
      <datalist id="crypto-pairs">
        {pairs.map((p) => (
          <option key={p.symbol} value={`${p.base}/USDT`}>{p.name}</option>
        ))}
      </datalist>
    </form>
  );
}
