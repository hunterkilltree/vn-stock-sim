"use client";

import { useState, useTransition } from "react";
import type { Order } from "@/lib/api";
import { cancelOrderAction } from "@/lib/orderActions";
import { formatThousandsVN } from "@/lib/format";

const KIND_LABEL: Record<Order["type"], string> = {
  limit: "LO",
  atc: "ATC",
  stop: "Stop",
  market: "MP",
  oco: "OCO",
};

// Portfolio.dc.html's "Lệnh chờ khớp" panel -- real queued orders (GET
// /orders, filtered by the caller for status "queued" + this
// portfolioId) with a working cancel button wired to
// POST /orders/:id/cancel (already existed server-side since Phase B;
// this is the first UI that calls it). o.price is Phase E's own backend
// addition -- see order/types.go's comment on why it didn't exist before.
export default function PendingOrdersCard({ orders }: { orders: Order[] }) {
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [, startTransition] = useTransition();

  function cancel(id: string) {
    setError(null);
    setPendingId(id);
    startTransition(async () => {
      const res = await cancelOrderAction(id);
      setPendingId(null);
      if (res.error) setError(res.error);
    });
  }

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Lệnh chờ khớp</h2>
        <span className="rounded-md bg-app-surface-3 px-2 py-[2px] font-plex-mono text-[11px] text-app-text-2">{orders.length}</span>
      </div>
      {error && <p className="text-[11.5px] text-price-down">{error}</p>}
      {orders.length === 0 ? (
        <p className="text-[12.5px] text-app-text-muted">Không có lệnh nào đang chờ khớp.</p>
      ) : (
        <div className="flex flex-col gap-[9px]">
          {orders.map((o) => {
            const buy = o.side === "buy";
            return (
              <div key={o.id} className="flex items-center gap-[11px] rounded-[10px] border border-app-hairline bg-app-surface-2 p-[10px_12px]">
                <span
                  className="shrink-0 rounded-md px-2 py-[3px] text-[11px] font-semibold"
                  style={{ color: buy ? "#35C77F" : "#FF5C5C", background: buy ? "rgba(53,199,127,0.14)" : "rgba(255,92,92,0.14)" }}
                >
                  {buy ? "Mua" : "Bán"}
                </span>
                <div className="flex min-w-0 flex-1 flex-col gap-[2px]">
                  <span className="font-plex-mono text-[13px] font-semibold">
                    {o.symbol} · {o.quantity.toLocaleString("vi-VN")}
                  </span>
                  <span className="text-[11px] text-app-text-muted">{KIND_LABEL[o.type]}</span>
                </div>
                <span className="font-plex-mono text-[13px]">{o.price ? formatThousandsVN(o.price) : "—"}</span>
                <button
                  type="button"
                  aria-label="Huỷ lệnh"
                  disabled={pendingId === o.id}
                  onClick={() => cancel(o.id)}
                  className="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-[8px] border border-app-border bg-app-surface-2 text-app-text-muted disabled:opacity-50"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                    <path d="M6 6l12 12M18 6L6 18" />
                  </svg>
                </button>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
