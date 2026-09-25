"use client";

import Link from "next/link";
import { useOptimistic, useTransition } from "react";
import type { WatchlistItem } from "@/lib/api";
import { setWatchedAction } from "@/lib/watchlistActions";
import { formatCryptoPrice, formatThousandsVN, signVN, tone } from "@/lib/format";

// "Danh sách theo dõi" on Main (phase-k.md decision 12). No screen in the
// design shows the list itself, so it borrows the positions card's row
// shape: symbol + name, price, change, and a remove button. Stocks and
// crypto pairs share one list.
export default function WatchlistCard({ items }: { items: WatchlistItem[] }) {
  const [shown, remove] = useOptimistic(items, (list, sym: string) => list.filter((i) => i.symbol !== sym));
  const [, start] = useTransition();

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Danh sách theo dõi</h2>
        <span className="text-[11.5px] text-app-text-muted">{shown.length} mã</span>
      </div>
      {shown.length === 0 ? (
        <p className="m-0 text-[12.5px] leading-[1.5] text-app-text-muted">
          Chưa theo dõi mã nào. Bấm biểu tượng dấu trang trên trang biểu đồ của một mã hoặc một cặp crypto để thêm vào đây.
        </p>
      ) : (
        <ul className="m-0 flex list-none flex-col p-0">
          {shown.map((i) => {
            const crypto = i.exchange === "CRYPTO";
            return (
              <li key={i.symbol} className="flex items-center gap-2 border-t border-app-hairline first:border-t-0">
                <Link href={crypto ? `/crypto/${i.symbol}` : `/stocks/${i.symbol}`} className="flex min-h-12 min-w-0 flex-1 items-center gap-3 py-[7px] text-app-text">
                  <span className="flex min-w-0 flex-1 flex-col gap-[1px]">
                    <span className="font-plex-mono text-[13px] font-semibold">{crypto ? i.symbol.replace(/USDT$/, "/USDT") : i.symbol}</span>
                    <span className="truncate text-[11px] text-app-text-muted">{i.companyName}</span>
                  </span>
                  <span className="flex flex-col items-end gap-[1px]">
                    <span className="font-plex-mono text-[13px]">{crypto ? formatCryptoPrice(i.lastPrice) : formatThousandsVN(i.lastPrice)}</span>
                    <span className="font-plex-mono text-[11.5px] font-semibold" style={{ color: tone(i.changePercent) }}>
                      {signVN(i.changePercent, 2)}%
                    </span>
                  </span>
                </Link>
                <button
                  type="button"
                  aria-label={`Bỏ ${i.symbol} khỏi danh sách theo dõi`}
                  onClick={() =>
                    start(async () => {
                      remove(i.symbol);
                      await setWatchedAction(i.symbol, false);
                    })
                  }
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[8px] text-app-text-muted hover:bg-app-surface-2"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                    <path d="M6 6l12 12M18 6L6 18" />
                  </svg>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
