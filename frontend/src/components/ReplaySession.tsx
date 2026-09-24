"use client";

import { useCallback, useEffect, useRef, useState, useTransition } from "react";
import type { ReplaySession as Session } from "@/lib/api";
import { startReplayAction, advanceReplayAction, placeReplayOrderAction, endReplayAction } from "@/lib/replayActions";
import { formatAmount, formatCryptoPrice, formatVN } from "@/lib/format";
import ReplayChart from "./ReplayChart";
import ReplayResultsPanel from "./ReplayResultsPanel";

const SPEEDS = [0.5, 1, 2, 4];
const AUTOPLAY_BASE_MS = 900;

// Replay.dc.html's full session screen: a start form, then (once a
// session exists) the chart + playback controls + order ticket + fill
// log + results panel, all driven by one piece of state fetched fresh
// from the backend after every action (see backend/internal/replay's
// SessionView -- the "future hidden" guarantee is enforced there, not
// by anything this component chooses not to render).
// market "crypto" is Crypto-Replay.dc.html: 1-hour candles from
// 01/05/2021, USDT prices, fractional amounts (phase-i.md decision 11).
export default function ReplaySession({ initialSymbol, market = "stock" }: { initialSymbol: string; market?: "stock" | "crypto" }) {
  const crypto = market === "crypto";
  const fmtPrice = (v: number) => (crypto ? formatCryptoPrice(v) : formatVN(v / 1000, 2));
  const fmtWhen = (iso: string) =>
    new Date(iso).toLocaleString("vi-VN", crypto
      ? { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "UTC" }
      : { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" });
  const [session, setSession] = useState<Session | null>(null);
  const [symbol, setSymbol] = useState(initialSymbol);
  const [error, setError] = useState<string | null>(null);
  const [pending, startTransition] = useTransition();
  const [autoPlay, setAutoPlay] = useState(false);
  const [speed, setSpeed] = useState(1);
  const [qty, setQty] = useState(market === "crypto" ? "0,05" : "1000");
  const [stopLossInput, setStopLossInput] = useState("");
  const busyRef = useRef(false);

  const advanceOne = useCallback(() => {
    if (!session || busyRef.current || session.status !== "active") return;
    busyRef.current = true;
    startTransition(async () => {
      const res = await advanceReplayAction(session.id);
      busyRef.current = false;
      if (res.error) setError(res.error);
      if (res.session) setSession(res.session);
    });
  }, [session]);

  // Auto-play: ticks advanceOne on an interval scaled by speed, stops
  // itself once the session runs out of candles or completes.
  useEffect(() => {
    if (!autoPlay || !session || session.done || session.status !== "active") return;
    const id = setInterval(advanceOne, AUTOPLAY_BASE_MS / speed);
    return () => clearInterval(id);
  }, [autoPlay, session, speed, advanceOne]);

  // Space bar steps one candle, matching the design's own keyboard hint
  // -- ignored while focus is in a text input so typing a quantity
  // doesn't also advance the chart.
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.code !== "Space") return;
      const tag = (e.target as HTMLElement | null)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA") return;
      e.preventDefault();
      advanceOne();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [advanceOne]);

  function start() {
    setError(null);
    startTransition(async () => {
      const res = await startReplayAction(symbol.trim().toUpperCase(), market);
      if (res.error) setError(res.error);
      if (res.session) setSession(res.session);
    });
  }

  function placeOrder(side: "buy" | "sell") {
    if (!session) return;
    setError(null);
    const quantity = crypto ? Number(qty.replace(/\./g, "").replace(",", ".")) || 0 : Number(qty.replace(/\D/g, "")) || 0;
    const stopRaw = Number(stopLossInput.replace(/\./g, "").replace(",", ".")) || 0;
    const stopLoss = side === "buy" ? (crypto ? stopRaw : stopRaw * 1000) : 0;
    startTransition(async () => {
      const res = await placeReplayOrderAction(session.id, side, quantity, stopLoss);
      if (res.error) setError(res.error);
      if (res.session) setSession(res.session);
    });
  }

  function endSession() {
    if (!session) return;
    setAutoPlay(false);
    startTransition(async () => {
      const res = await endReplayAction(session.id);
      if (res.error) setError(res.error);
      if (res.session) setSession(res.session);
    });
  }

  function exitSession() {
    // "Thoát phiên" -- leaves the session as-is server-side (it stays
    // "active" and could in principle be resumed by id), just stops
    // rendering it here. Session resumption across a page reload isn't
    // built in V1 -- a documented simplification, see phase-f.md.
    setAutoPlay(false);
    setSession(null);
    setError(null);
  }

  if (!session) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <section className="flex w-[380px] flex-col gap-4 rounded-2xl border border-app-border bg-app-surface p-6">
          <div className="flex flex-col gap-1">
            <h1 className="m-0 font-display text-[22px] font-bold tracking-[-0.015em]">Chế độ Replay</h1>
            <p className="m-0 text-[12.5px] text-app-text-muted">
              Giao dịch lại quá khứ, từng nến một -- tương lai bị che hoàn toàn phía backend, không chỉ ở giao diện.
            </p>
          </div>
          <div className="flex flex-col gap-[6px]">
            <label htmlFor="rsym" className="text-[11px] text-app-text-muted">
              {crypto ? "Cặp giao dịch" : "Mã cổ phiếu"}
            </label>
            <input
              id="rsym"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value)}
              className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm uppercase text-app-text outline-none"
            />
          </div>
          {error && <p className="text-[12px] text-price-down">{error}</p>}
          <button
            type="button"
            disabled={pending || !symbol.trim()}
            onClick={start}
            className="flex h-12 items-center justify-center rounded-[11px] bg-app-accent text-[14.5px] font-semibold text-app-accent-ink disabled:opacity-60"
          >
            {pending ? "Đang bắt đầu..." : "Bắt đầu phiên"}
          </button>
        </section>
      </div>
    );
  }

  const lastClose = session.bars[session.bars.length - 1]?.close ?? 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <header className="flex items-center justify-between gap-6">
        <div className="flex items-center gap-5">
          <div className="flex flex-col gap-1">
            <h1 className="m-0 font-display text-[26px] font-bold tracking-[-0.015em]">Chế độ Replay</h1>
            <span className="text-[12.5px] text-app-text-muted">
              {crypto
                ? `${session.symbol.replace(/USDT$/, "/USDT")} · Nến 1 giờ · chạy liên tục, kể cả cuối tuần`
                : `${session.symbol} · Nến ngày · Phiên mô phỏng`}
            </span>
          </div>
          <div className="flex items-center gap-2 rounded-[10px] border border-[#4A3521] bg-[#1C1813] px-[13px] py-[7px]">
            <span className="font-plex-mono text-[12.5px] font-semibold text-app-accent">
              Nến {session.currentBar} / {session.totalBars}
            </span>
          </div>
          {session.status === "active" && (
            <span className="text-[12.5px] text-app-text-3">Không thể xem trước tương lai</span>
          )}
        </div>
        <div className="flex items-center gap-[10px]">
          <button
            type="button"
            onClick={exitSession}
            className="h-11 rounded-[11px] border border-app-border bg-app-surface px-[15px] text-[13px] font-medium text-app-text-3"
          >
            Thoát phiên
          </button>
          {session.status === "active" && (
            <button
              type="button"
              disabled={pending}
              onClick={endSession}
              className="h-11 rounded-[11px] bg-app-accent px-4 text-[13.5px] font-semibold text-app-accent-ink disabled:opacity-60"
            >
              Kết thúc &amp; tính điểm
            </button>
          )}
        </div>
      </header>

      {error && (
        <p className="rounded-xl border border-app-warn-border bg-app-warn-surface px-4 py-2 text-sm text-app-warn-text">{error}</p>
      )}

      <div className="flex min-h-0 flex-1 gap-5">
        <div className="flex min-w-0 flex-1 flex-col gap-[14px]">
          <ReplayChart session={session} scale={crypto ? 1 : 1000} />

          <section className="flex shrink-0 items-center gap-[18px] rounded-2xl border border-app-border bg-app-surface p-[14px_18px]">
            <div className="flex items-center gap-2">
              <button
                type="button"
                disabled
                title="Chưa hỗ trợ lùi nến"
                aria-label="Lùi một nến"
                className="flex h-11 w-11 items-center justify-center rounded-[11px] border border-app-border bg-app-surface-2 text-app-text-muted opacity-40"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M15 6l-7 6 7 6V6z" />
                  <path d="M6 5v14" />
                </svg>
              </button>
              <button
                type="button"
                aria-label={autoPlay ? "Tạm dừng" : "Phát tự động"}
                disabled={session.done || session.status !== "active"}
                onClick={() => setAutoPlay((v) => !v)}
                className="flex h-11 w-[52px] items-center justify-center rounded-[11px] bg-app-accent text-app-accent-ink disabled:opacity-40"
              >
                {autoPlay ? (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <rect x="5" y="4" width="5" height="16" />
                    <rect x="14" y="4" width="5" height="16" />
                  </svg>
                ) : (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M8 5l12 7-12 7V5z" />
                  </svg>
                )}
              </button>
              <button
                type="button"
                disabled={pending || session.done || session.status !== "active"}
                onClick={advanceOne}
                className="flex h-11 items-center gap-[9px] rounded-[11px] border border-app-border-strong bg-app-border px-4 text-[13px] font-semibold text-app-text disabled:opacity-40"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M9 6l7 6-7 6V6z" />
                  <path d="M18 5v14" />
                </svg>
                <span>Nến tiếp</span>
              </button>
            </div>

            <div className="flex items-center gap-[9px]">
              <span className="text-[11px] text-app-text-muted">Tốc độ</span>
              <div className="flex gap-[5px]">
                {SPEEDS.map((s) => (
                  <button
                    key={s}
                    type="button"
                    onClick={() => setSpeed(s)}
                    className="h-8 rounded-lg border px-[11px] font-plex-mono text-[11.5px]"
                    style={
                      speed === s
                        ? { borderColor: "var(--app-border-strong)", background: "var(--app-border)", color: "var(--app-text)" }
                        : { borderColor: "var(--app-border)", background: "var(--app-surface-2)", color: "var(--app-text-muted)" }
                    }
                  >
                    {s}×
                  </button>
                ))}
              </div>
            </div>

            <div className="min-w-0 flex-1">
              <div className="flex justify-between font-plex-mono text-[10.5px] text-app-text-muted">
                <span>0</span>
                <span className="text-app-text-2">
                  {session.currentBar} / {session.totalBars}
                </span>
                <span>{session.totalBars}</span>
              </div>
              <div className="relative mt-[7px] h-[6px] rounded-[3px] bg-app-surface-3">
                <div
                  className="absolute left-0 top-0 h-[6px] rounded-[3px] bg-app-accent"
                  style={{ width: `${(session.currentBar / session.totalBars) * 100}%` }}
                />
              </div>
            </div>

            <div className="flex items-center gap-[7px] text-[11.5px] text-app-text-muted">
              <kbd className="box-border rounded-[6px] border border-app-border bg-app-surface-2 px-2 py-[3px] font-plex-mono text-[11px] text-app-text-2">
                Space
              </kbd>
              <span>nến tiếp</span>
            </div>
          </section>

          <section className="flex h-[176px] shrink-0 flex-col gap-[10px] rounded-2xl border border-app-border bg-app-surface p-[16px_18px]">
            <div className="flex items-center justify-between">
              <h2 className="m-0 text-[14px] font-semibold">Lệnh trong phiên này</h2>
              <span className="text-[11.5px] text-app-text-muted">Tự động ghi vào Sổ giao dịch</span>
            </div>
            {session.fills.length === 0 ? (
              <p className="text-[12.5px] text-app-text-muted">Chưa có lệnh nào trong phiên này.</p>
            ) : (
              <div className="min-h-0 flex-1 overflow-auto">
                <table className="w-full text-[12.5px]">
                  <thead>
                    <tr className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">
                      <th className="p-0 pb-[7px] text-left font-medium">Nến</th>
                      <th className="p-0 pb-[7px] text-left font-medium">Ngày mô phỏng</th>
                      <th className="p-0 pb-[7px] text-left font-medium">Lệnh</th>
                      <th className="p-0 pb-[7px] text-right font-medium">KL</th>
                      <th className="p-0 pb-[7px] text-right font-medium">Giá</th>
                      <th className="p-0 pb-[7px] pl-4 text-left font-medium">Ghi chú</th>
                    </tr>
                  </thead>
                  <tbody>
                    {session.fills.map((f, i) => {
                      const buy = f.side === "buy";
                      return (
                        <tr key={i} className="border-t border-app-hairline">
                          <td className="py-2 font-plex-mono text-app-text-muted">#{f.barIndex}</td>
                          <td className="py-2 font-plex-mono">
                            {fmtWhen(f.date)}
                          </td>
                          <td className="py-2">
                            <span
                              className="rounded-md px-2 py-[2px] text-[11.5px] font-semibold"
                              style={{ color: buy ? "#35C77F" : "#FF5C5C", background: buy ? "rgba(53,199,127,0.14)" : "rgba(255,92,92,0.14)" }}
                            >
                              {buy ? "Mua" : "Bán"}
                            </span>
                          </td>
                          <td className="py-2 text-right font-plex-mono">{formatAmount(f.quantity)}</td>
                          <td className="py-2 text-right font-plex-mono">{fmtPrice(f.price)}</td>
                          <td className="py-2 pl-4 text-app-text-muted">{f.note || "—"}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>

        <div className="flex w-[340px] shrink-0 flex-col gap-4">
          <section className="flex flex-col gap-[13px] rounded-2xl border border-app-border bg-app-surface p-[18px]">
            <div className="flex items-center justify-between">
              <h2 className="m-0 text-[15px] font-semibold">Đặt lệnh tại nến này</h2>
              <span className="font-plex-mono text-[12px] text-price-up">{fmtPrice(lastClose)}</span>
            </div>
            <div className="grid grid-cols-2 gap-[10px]">
              <div className="flex flex-col gap-[6px]">
                <label htmlFor="rqty" className="text-[11px] text-app-text-muted">
                  Khối lượng
                </label>
                <input
                  id="rqty"
                  value={qty}
                  onChange={(e) => setQty(e.target.value)}
                  disabled={session.status !== "active"}
                  className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none disabled:opacity-50"
                />
              </div>
              <div className="flex flex-col gap-[6px]">
                <label htmlFor="rstop" className="text-[11px] text-app-text-muted">
                  {crypto ? "Cắt lỗ (USDT)" : "Cắt lỗ (nghìn ₫)"}
                </label>
                <input
                  id="rstop"
                  value={stopLossInput}
                  onChange={(e) => setStopLossInput(e.target.value)}
                  placeholder="Không đặt"
                  disabled={session.status !== "active"}
                  className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none placeholder:text-app-text-muted disabled:opacity-50"
                />
              </div>
            </div>
            <div className="flex gap-[10px]">
              <button
                type="button"
                disabled={pending || session.status !== "active"}
                onClick={() => placeOrder("buy")}
                className="h-12 flex-1 rounded-[11px] text-[14.5px] font-semibold disabled:opacity-50"
                style={{ background: "#35C77F", color: "#08130D" }}
              >
                Mua
              </button>
              <button
                type="button"
                disabled={pending || session.status !== "active" || session.positionQty === 0}
                onClick={() => placeOrder("sell")}
                className="h-12 flex-1 rounded-[11px] text-[14.5px] font-semibold disabled:opacity-50"
                style={{ background: "#FF5C5C", color: "#1A0808" }}
              >
                Bán
              </button>
            </div>
            <div className="flex justify-between border-t border-app-hairline pt-3 text-[12.5px]">
              <span className="text-app-text-muted">Đang giữ</span>
              <span className="font-plex-mono font-semibold">
                {session.positionQty > 0
                  ? `${formatAmount(session.positionQty)} ${session.symbol.replace(/USDT$/, "")} @ ${fmtPrice(session.avgCost ?? 0)}`
                  : "Chưa có vị thế"}
              </span>
            </div>
          </section>

          <ReplayResultsPanel result={session.result} currency={crypto ? "USDT" : "VND"} />
        </div>
      </div>
    </div>
  );
}
