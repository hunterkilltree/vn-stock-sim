"use client";

import { useState, useTransition } from "react";
import type { Order } from "@/lib/api";
import { cancelOrderAction } from "@/lib/orderActions";
import { formatAmount, formatCryptoPrice, formatThousandsVN } from "@/lib/format";

const KIND_LABEL: Record<Order["type"], string> = {
  limit: "LO",
  atc: "ATC",
  stop: "Stop",
  market: "MP",
  oco: "OCO",
};

const STATUS: Record<Exclude<Order["status"], "queued">, { label: string; color: string }> = {
  filled: { label: "Đã khớp", color: "var(--price-up)" },
  rejected: { label: "Bị từ chối", color: "var(--price-down)" },
  cancelled: { label: "Đã huỷ", color: "var(--app-text-muted)" },
};

// The ledger's reasons, in the UI's language.
const REJECT_VI: Record<string, string> = {
  "insufficient virtual cash": "Không đủ tiền mặt ảo lúc lệnh khớp",
  "insufficient shares": "Không đủ số lượng nắm giữ lúc lệnh khớp",
};

type Props = {
  orders: Order[];
  // Orders the matcher finished with or the user cancelled (Phase K).
  recent?: Order[];
  market?: "stock" | "crypto";
};

// Portfolio.dc.html's "Lệnh chờ khớp" panel -- real queued orders with a
// working cancel button (POST /orders/:id/cancel). Since Phase K the
// matcher fills them (phase-k.md), so the card also lists the latest
// processed ones: filled at what price, which OCO leg, or why rejected.
export default function PendingOrdersCard({ orders, recent = [], market = "stock" }: Props) {
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [, startTransition] = useTransition();
  const crypto = market === "crypto";
  const price = (v: number) => (crypto ? formatCryptoPrice(v) : formatThousandsVN(v));
  const qty = (o: Order) => (crypto ? `${formatAmount(o.quantity)} ${o.symbol.replace(/USDT$/, "")}` : o.quantity.toLocaleString("vi-VN"));
  const sym = (o: Order) => (crypto ? o.symbol.replace(/USDT$/, "/USDT") : o.symbol);

  // What the order waits for, in the ticket's own terms.
  function condition(o: Order): string {
    if (o.type === "atc") return "Khớp giá đóng cửa 14:45";
    if (o.type === "oco") return `Chốt ${price(o.price ?? 0)} · dừng ${price(o.stopPrice ?? 0)}`;
    if (o.type === "stop") return `Kích hoạt ${price(o.price ?? 0)}`;
    return `Giới hạn ${price(o.price ?? 0)}`;
  }

  function cancel(id: string) {
    setError(null);
    setPendingId(id);
    startTransition(async () => {
      const res = await cancelOrderAction(id);
      setPendingId(null);
      if (res.error) setError(res.error);
    });
  }

  const sideChip = (buy: boolean) => (
    <span
      className="shrink-0 rounded-md px-2 py-[3px] text-[11px] font-semibold"
      style={{ color: buy ? "#35C77F" : "#FF5C5C", background: buy ? "rgba(53,199,127,0.14)" : "rgba(255,92,92,0.14)" }}
    >
      {buy ? "Mua" : "Bán"}
    </span>
  );

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
          {orders.map((o) => (
            <div key={o.id} className="flex items-center gap-[11px] rounded-[10px] border border-app-hairline bg-app-surface-2 p-[10px_12px]">
              {sideChip(o.side === "buy")}
              <div className="flex min-w-0 flex-1 flex-col gap-[2px]">
                <span className="truncate font-plex-mono text-[13px] font-semibold">
                  {sym(o)} · {qty(o)}
                </span>
                <span className="truncate text-[11px] text-app-text-muted">
                  {KIND_LABEL[o.type]} · {condition(o)}
                </span>
              </div>
              <button
                type="button"
                aria-label="Huỷ lệnh"
                disabled={pendingId === o.id}
                onClick={() => cancel(o.id)}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[8px] border border-app-border bg-app-surface-2 text-app-text-muted disabled:opacity-50 lg:h-[30px] lg:w-[30px]"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                  <path d="M6 6l12 12M18 6L6 18" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      )}

      {recent.length > 0 && (
        <div className="flex flex-col gap-2 border-t border-app-hairline pt-3">
          <h3 className="m-0 text-[12.5px] font-semibold text-app-text-2">Vừa xử lý</h3>
          {recent.map((o) => {
            const st = STATUS[o.status as keyof typeof STATUS];
            return (
              <div key={o.id} className="flex items-start gap-[10px] text-[12px]">
                {sideChip(o.side === "buy")}
                <div className="flex min-w-0 flex-1 flex-col gap-[2px]">
                  <span className="truncate font-plex-mono font-semibold">
                    {sym(o)} · {qty(o)} · {KIND_LABEL[o.type]}
                    {o.triggeredBy && <span className="font-normal text-app-text-muted"> ({o.triggeredBy === "stop" ? "chân dừng" : "chân chốt"})</span>}
                  </span>
                  {o.status === "rejected" && o.rejectReason && (
                    <span className="text-[11px] text-price-down">{REJECT_VI[o.rejectReason] ?? o.rejectReason}</span>
                  )}
                </div>
                <span className="shrink-0 text-right font-plex-mono">
                  <span style={{ color: st?.color }}>{st?.label}</span>
                  {o.status === "filled" && o.filledPrice !== undefined && <span className="block text-app-text-3">{price(o.filledPrice)}</span>}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
