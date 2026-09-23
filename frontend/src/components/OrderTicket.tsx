"use client";

import { useActionState, useMemo, useState } from "react";
import Link from "next/link";
import { placeOrderAction, type OrderFormState } from "@/lib/orderActions";
import { formatVN } from "@/lib/format";

type Props = {
  symbol: string;
  lastPrice: number; // raw VND
  buyingPower: number | null; // raw VND cash balance, null when guest
};

const ORDER_TYPES: { label: string; value: "limit" | "market" | "atc" | "stop" }[] = [
  { label: "LO", value: "limit" },
  { label: "MP", value: "market" },
  { label: "ATC", value: "atc" },
  { label: "Stop", value: "stop" },
];

const FEE_RATE = 0.0015;
const LOT_SIZE = 100;

// "Dat lenh giay" order ticket from design/screens/Detail.dc.html --
// a real form posting to POST /api/v1/orders via orderActions.ts's
// Server Action (httpOnly-cookie token, same pattern as authActions.ts),
// not a static mockup panel. Guest state (buyingPower === null) replaces
// the submit area with a sign-in prompt instead of a modal -- see
// phase-d.md decision 5 (a modal over the order ticket is explicitly
// listed as NOT YET DESIGNED in design/SCREENS.md).
export default function OrderTicket({ symbol, lastPrice, buyingPower }: Props) {
  const initialState: OrderFormState = { error: null, success: null };
  const [state, formAction, pending] = useActionState(placeOrderAction, initialState);

  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [type, setType] = useState<(typeof ORDER_TYPES)[number]["value"]>("limit");
  const [priceK, setPriceK] = useState((lastPrice / 1000).toFixed(2));
  const [quantity, setQuantity] = useState(LOT_SIZE);

  const priceVnd = (Number(priceK.replace(",", ".")) || 0) * 1000;
  const orderValue = priceVnd * quantity;
  const fee = orderValue * FEE_RATE;
  const remaining = buyingPower !== null ? buyingPower - (side === "buy" ? orderValue + fee : 0) : null;

  const maxShares = useMemo(() => {
    if (buyingPower === null || priceVnd <= 0) return 0;
    return Math.floor(buyingPower / priceVnd / LOT_SIZE) * LOT_SIZE;
  }, [buyingPower, priceVnd]);

  const lots: { label: string; pct: number }[] = [
    { label: "25%", pct: 0.25 },
    { label: "50%", pct: 0.5 },
    { label: "75%", pct: 0.75 },
    { label: "Tối đa", pct: 1 },
  ];

  return (
    <section className="flex flex-col gap-[14px] rounded-2xl border border-app-border bg-app-surface p-[18px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Đặt lệnh giấy</h2>
        <span className="text-[11px] text-app-text-muted">Tài khoản mô phỏng</span>
      </div>

      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setSide("buy")}
          className="h-11 flex-1 rounded-[10px] border text-sm font-semibold"
          style={
            side === "buy"
              ? { borderColor: "rgba(53,199,127,0.5)", background: "rgba(53,199,127,0.16)", color: "#35C77F" }
              : { borderColor: "var(--app-border)", background: "var(--app-surface-2)", color: "var(--app-text-3)" }
          }
        >
          Mua
        </button>
        <button
          type="button"
          onClick={() => setSide("sell")}
          className="h-11 flex-1 rounded-[10px] border text-sm font-medium"
          style={
            side === "sell"
              ? { borderColor: "rgba(255,92,92,0.5)", background: "rgba(255,92,92,0.16)", color: "#FF5C5C" }
              : { borderColor: "var(--app-border)", background: "var(--app-surface-2)", color: "var(--app-text-3)" }
          }
        >
          Bán
        </button>
      </div>

      <div className="flex flex-col gap-[6px]">
        <span className="text-[11px] text-app-text-muted">Loại lệnh</span>
        <div className="flex gap-[6px]">
          {ORDER_TYPES.map((t) => (
            <button
              key={t.value}
              type="button"
              onClick={() => setType(t.value)}
              className="h-9 flex-1 rounded-[9px] border font-plex-mono text-xs font-semibold"
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

      <div className="grid grid-cols-2 gap-[10px]">
        <div className="flex flex-col gap-[6px]">
          <label htmlFor="price" className="text-[11px] text-app-text-muted">Giá (nghìn ₫)</label>
          <input
            id="price"
            type="text"
            value={priceK}
            onChange={(e) => setPriceK(e.target.value)}
            disabled={type === "market"}
            className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none disabled:opacity-50"
          />
        </div>
        <div className="flex flex-col gap-[6px]">
          <label htmlFor="qty" className="text-[11px] text-app-text-muted">Khối lượng</label>
          <input
            id="qty"
            type="text"
            value={quantity}
            onChange={(e) => setQuantity(Number(e.target.value.replace(/\D/g, "")) || 0)}
            className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none"
          />
        </div>
      </div>

      <div className="flex gap-[6px]">
        {lots.map((l) => (
          <button
            key={l.label}
            type="button"
            disabled={buyingPower === null}
            onClick={() => setQuantity(Math.max(LOT_SIZE, Math.round((maxShares * l.pct) / LOT_SIZE) * LOT_SIZE))}
            className="h-8 flex-1 rounded-lg border border-app-border bg-app-surface-2 font-plex-mono text-[11.5px] text-app-text-3 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {l.label}
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-[7px] border-y border-app-hairline py-3">
        <div className="flex justify-between text-[12.5px]">
          <span className="text-app-text-muted">Giá trị lệnh</span>
          <span className="font-plex-mono font-semibold">{formatVN(orderValue, 0)} ₫</span>
        </div>
        <div className="flex justify-between text-[12.5px]">
          <span className="text-app-text-muted">Phí mô phỏng (0,15%)</span>
          <span className="font-plex-mono">{formatVN(fee, 0)} ₫</span>
        </div>
        {remaining !== null && (
          <div className="flex justify-between text-[12.5px]">
            <span className="text-app-text-muted">Sức mua còn lại</span>
            <span className="font-plex-mono">{formatVN(remaining, 0)} ₫</span>
          </div>
        )}
      </div>

      {buyingPower === null ? (
        <Link
          href="/login"
          className="flex h-12 items-center justify-center rounded-[11px] bg-app-accent text-[14.5px] font-semibold text-app-accent-ink"
        >
          Đăng nhập để đặt lệnh
        </Link>
      ) : (
        <form action={formAction} className="flex flex-col gap-2">
          <input type="hidden" name="symbol" value={symbol} />
          <input type="hidden" name="side" value={side} />
          <input type="hidden" name="type" value={type} />
          <input type="hidden" name="quantity" value={quantity} />
          {state.error && (
            <p className="rounded-lg border border-app-warn-border bg-app-warn-surface px-3 py-2 text-xs text-app-warn-text">{state.error}</p>
          )}
          {state.success && (
            <p className="rounded-lg border border-app-border bg-app-surface-2 px-3 py-2 text-xs text-price-up">
              {state.success.status === "filled"
                ? `Đã khớp ở ${formatVN((state.success.filledPrice ?? 0) / 1000, 2)} (phí ${formatVN(state.success.fee ?? 0, 0)} ₫)`
                : "Lệnh đã được đặt và đang chờ khớp."}
            </p>
          )}
          <button
            type="submit"
            disabled={pending || quantity <= 0}
            className="h-12 rounded-[11px] border-0 text-[14.5px] font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            style={{
              background: side === "buy" ? "#35C77F" : "#FF5C5C",
              color: side === "buy" ? "#08130D" : "#1A0808",
            }}
          >
            {pending ? "Đang gửi lệnh..." : `${side === "buy" ? "Mua" : "Bán"} ${quantity} ${symbol} ở ${priceK}`}
          </button>
        </form>
      )}
    </section>
  );
}
