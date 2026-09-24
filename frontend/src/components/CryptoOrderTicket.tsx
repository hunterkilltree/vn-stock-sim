"use client";

import Link from "next/link";
import { useActionState, useState, useTransition } from "react";
import { openCryptoWalletAction, placeCryptoOrderAction, type CryptoOrderState } from "@/lib/cryptoActions";
import { formatAmount, formatCryptoPrice, formatVN } from "@/lib/format";

type Props = {
  symbol: string; // "BTCUSDT"
  base: string;
  lastPrice: number;
  // null = guest; undefined = signed in without a crypto wallet yet.
  usdtBalance: number | null | undefined;
  positionQty: number;
};

const TYPES = [
  { label: "Thị trường", value: "market" },
  { label: "Giới hạn", value: "limit" },
  { label: "Dừng", value: "stop" },
  { label: "OCO", value: "oco" },
] as const;

const FEE_RATE = 0.001; // Crypto-Detail.dc.html: "Phí mô phỏng (0,10%)"

// VN-formatted input ("64.820,50", "0,025") -> number.
function parseVN(s: string): number {
  return Number(s.replace(/\./g, "").replace(",", ".")) || 0;
}

// Crypto-Detail.dc.html's "Đặt lệnh giấy" panel (phase-i.md decision 8).
export default function CryptoOrderTicket({ symbol, base, lastPrice, usdtBalance, positionQty }: Props) {
  const [state, formAction, pending] = useActionState<CryptoOrderState, FormData>(placeCryptoOrderAction, { error: null, success: null });
  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [type, setType] = useState<(typeof TYPES)[number]["value"]>("market");
  const [priceIn, setPriceIn] = useState(formatCryptoPrice(lastPrice));
  const [stopIn, setStopIn] = useState("");
  const [qtyIn, setQtyIn] = useState("");
  const [opening, startOpening] = useTransition();
  const [openError, setOpenError] = useState<string | null>(null);

  const price = type === "market" ? lastPrice : parseVN(priceIn);
  const qty = parseVN(qtyIn);
  const value = price * qty;
  const fee = value * FEE_RATE;
  const balance = usdtBalance ?? 0;
  const remaining = side === "buy" ? balance - value - fee : balance + value - fee;
  const maxQty = side === "buy" ? (price > 0 ? balance / (price * (1 + FEE_RATE)) : 0) : positionQty;

  function lot(pct: number) {
    const q = Math.floor(maxQty * pct * 1e6) / 1e6;
    setQtyIn(formatAmount(q, 6));
  }

  const sideStyle = (on: boolean, buy: boolean) =>
    on
      ? buy
        ? { borderColor: "rgba(53,199,127,0.5)", background: "rgba(53,199,127,0.16)", color: "#35C77F" }
        : { borderColor: "rgba(255,92,92,0.5)", background: "rgba(255,92,92,0.16)", color: "#FF5C5C" }
      : { borderColor: "var(--app-border)", background: "var(--app-surface-2)", color: "var(--app-text-3)" };

  const input =
    "box-border h-11 w-full rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none disabled:opacity-50";

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-4">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Đặt lệnh giấy</h2>
        <span className="text-[11px] text-app-text-muted">Giao ngay · không đòn bẩy</span>
      </div>
      <div className="flex gap-2">
        <button type="button" onClick={() => setSide("buy")} className="h-11 flex-1 rounded-[10px] border text-sm font-semibold" style={sideStyle(side === "buy", true)}>
          Mua
        </button>
        <button type="button" onClick={() => setSide("sell")} className="h-11 flex-1 rounded-[10px] border text-sm font-semibold" style={sideStyle(side === "sell", false)}>
          Bán
        </button>
      </div>
      <div className="flex flex-col gap-[6px]">
        <span className="text-[11px] text-app-text-muted">Loại lệnh</span>
        <div className="grid grid-cols-4 gap-[6px]">
          {TYPES.map((t) => (
            <button
              key={t.value}
              type="button"
              aria-pressed={type === t.value}
              onClick={() => setType(t.value)}
              className="h-9 rounded-[9px] border text-[12px] font-semibold"
              style={
                type === t.value
                  ? { borderColor: "var(--app-border-strong)", background: "var(--app-border)", color: "var(--app-text)" }
                  : { borderColor: "var(--app-border)", background: "var(--app-surface-2)", color: "var(--app-text-muted)" }
              }
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-[6px]">
        <label htmlFor="cprice" className="text-[11px] text-app-text-muted">
          {type === "oco" ? "Giá giới hạn (USDT)" : type === "stop" ? "Giá kích hoạt (USDT)" : "Giá (USDT)"}
        </label>
        <input
          id="cprice"
          className={input}
          disabled={type === "market"}
          value={type === "market" ? formatCryptoPrice(lastPrice) : priceIn}
          onChange={(e) => setPriceIn(e.target.value)}
        />
      </div>
      {type === "oco" && (
        <div className="flex flex-col gap-[6px]">
          <label htmlFor="cstop" className="text-[11px] text-app-text-muted">Giá dừng (USDT)</label>
          <input id="cstop" className={input} value={stopIn} onChange={(e) => setStopIn(e.target.value)} placeholder="Chốt lỗ / chốt lời" />
        </div>
      )}
      <div className="flex flex-col gap-[6px]">
        <label htmlFor="cqty" className="text-[11px] text-app-text-muted">Số lượng ({base})</label>
        <input id="cqty" className={input} inputMode="decimal" value={qtyIn} onChange={(e) => setQtyIn(e.target.value)} placeholder="0,00000" />
      </div>
      <div className="grid grid-cols-4 gap-[6px]">
        {[
          { label: "25%", pct: 0.25 },
          { label: "50%", pct: 0.5 },
          { label: "75%", pct: 0.75 },
          { label: "Tối đa", pct: 1 },
        ].map((l) => (
          <button
            key={l.label}
            type="button"
            disabled={usdtBalance == null}
            onClick={() => lot(l.pct)}
            className="h-8 rounded-lg border border-app-border bg-app-surface-2 font-plex-mono text-[11.5px] text-app-text-3 disabled:opacity-40"
          >
            {l.label}
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-[6px] border-t border-app-hairline pt-3 text-[12.5px]">
        <div className="flex justify-between">
          <span className="text-app-text-muted">Giá trị lệnh</span>
          <span className="font-plex-mono">{formatVN(value, 2)} USDT</span>
        </div>
        <div className="flex justify-between">
          <span className="text-app-text-muted">Phí mô phỏng (0,10%)</span>
          <span className="font-plex-mono">{formatVN(fee, 2)} USDT</span>
        </div>
        {usdtBalance != null && (
          <div className="flex justify-between">
            <span className="text-app-text-muted">USDT còn lại</span>
            <span className="font-plex-mono">{formatVN(type === "market" ? remaining : balance, 2)} USDT</span>
          </div>
        )}
      </div>

      {usdtBalance === null ? (
        <Link href="/login" className="flex h-12 items-center justify-center rounded-[11px] bg-app-accent text-[14.5px] font-semibold text-app-accent-ink">
          Đăng nhập để đặt lệnh giấy
        </Link>
      ) : usdtBalance === undefined ? (
        <>
          <button
            type="button"
            disabled={opening}
            onClick={() =>
              startOpening(async () => {
                const r = await openCryptoWalletAction();
                setOpenError(r.error);
              })
            }
            className="h-12 rounded-[11px] bg-app-accent text-[14.5px] font-semibold text-app-accent-ink disabled:opacity-60"
          >
            {opening ? "Đang mở ví…" : "Mở ví crypto 10.000 USDT"}
          </button>
          {openError && <p className="m-0 text-xs text-price-down">{openError}</p>}
        </>
      ) : (
        <form action={formAction} className="flex flex-col gap-2">
          <input type="hidden" name="symbol" value={symbol} />
          <input type="hidden" name="side" value={side} />
          <input type="hidden" name="type" value={type} />
          <input type="hidden" name="quantity" value={qty} />
          <input type="hidden" name="price" value={type === "market" ? 0 : price} />
          <input type="hidden" name="stopPrice" value={type === "oco" ? parseVN(stopIn) : 0} />
          {state.error && <p className="m-0 rounded-lg border border-app-warn-border bg-app-warn-surface px-3 py-2 text-xs text-app-warn-text">{state.error}</p>}
          {state.success && (
            <p className="m-0 rounded-lg border border-app-border bg-app-surface-2 px-3 py-2 text-xs text-price-up" role="status">
              {state.success.status === "filled"
                ? `Đã khớp ${formatAmount(state.success.quantity ?? 0)} ${base} ở ${formatCryptoPrice(state.success.filledPrice ?? 0)} (phí ${formatVN(state.success.fee ?? 0, 2)} USDT)`
                : "Lệnh đã được đặt và đang chờ khớp."}
            </p>
          )}
          <button
            type="submit"
            disabled={pending || qty <= 0}
            className="h-12 rounded-[11px] text-[14.5px] font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            style={{ background: side === "buy" ? "#35C77F" : "#FF5C5C", color: side === "buy" ? "#08130D" : "#1A0808" }}
          >
            {pending ? "Đang gửi lệnh..." : `${side === "buy" ? "Mua" : "Bán"} ${formatAmount(qty)} ${base}`}
          </button>
        </form>
      )}
    </section>
  );
}
