"use client";

import Link from "next/link";
import { useState, useTransition } from "react";
import WillBadge from "@/components/WillBadge";
import { strategyRules } from "@/components/QuantAssistantCard";
import { formatThousandsVN, formatVN, formatVolumeVN, signVN } from "@/lib/format";
import { addToWatchlistAction, runBacktestAction } from "@/lib/quantActions";
import type { BacktestResult, QuantScreen, QuantStrategy } from "@/lib/quantTypes";

type Props = {
  screen: QuantScreen | null;
  strategy: QuantStrategy | null;
  years: number;
};

function csvFor(screen: QuantScreen): string {
  const head = ["Mã", "Sàn", "Giá (nghìn ₫)", "RSI(14)", "ROE (%)", "P/E", "KLTB20"];
  const rows = screen.matches.map((r) => [r.symbol, r.exchange, (r.price / 1000).toFixed(2), r.rsi14.toFixed(1), r.roe.toFixed(1), r.pe.toFixed(1), String(r.avgVolume20)]);
  return [head, ...rows].map((line) => line.map((c) => `"${c.replace(/"/g, '""')}"`).join(",")).join("\n");
}

// Quant.dc.html's right panel: "Kết quả lọc", "Chiến lược nháp", and the
// WILL-badged "Mã Python" tab (phase-h.md decision 13).
export default function QuantResultsPanel({ screen, strategy, years }: Props) {
  const [tab, setTab] = useState<"results" | "strategy">("results");
  const [notice, setNotice] = useState<string | null>(null);
  const [multi, setMulti] = useState<{ symbol: string; result: BacktestResult | null; error: string | null }[]>([]);
  const [pending, startTransition] = useTransition();

  const matches = screen?.matches ?? [];

  function exportCsv() {
    if (!screen) return;
    const blob = new Blob(["﻿" + csvFor(screen)], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "quant-ket-qua-loc.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  function addToWatchlist() {
    startTransition(async () => {
      const res = await addToWatchlistAction(matches.map((m) => m.symbol));
      setNotice(res.error ?? `Đã thêm ${res.added} mã vào danh sách theo dõi.`);
    });
  }

  function backtestAll() {
    if (!strategy) return;
    startTransition(async () => {
      const out = [];
      for (const m of matches.slice(0, 10)) {
        const res = await runBacktestAction(m.symbol, strategy, years);
        out.push({ symbol: m.symbol, result: res.result, error: res.error });
      }
      setMulti(out);
      setTab("strategy");
    });
  }

  const tabs = [
    { id: "results" as const, label: `Kết quả lọc · ${matches.length}` },
    { id: "strategy" as const, label: "Chiến lược nháp" },
  ];

  return (
    <aside className="flex w-full flex-col overflow-hidden rounded-[14px] lg:w-[416px] lg:shrink-0 border border-app-border bg-app-surface">
      <div role="tablist" className="flex gap-1 border-b border-app-hairline p-[6px]">
        {tabs.map((t) => (
          <button
            key={t.id}
            role="tab"
            type="button"
            aria-selected={tab === t.id}
            onClick={() => setTab(t.id)}
            className="h-9 flex-grow rounded-[9px] text-[12.5px]"
            style={{
              background: tab === t.id ? "var(--app-surface-3)" : "transparent",
              color: tab === t.id ? "var(--app-text)" : "var(--app-text-muted)",
              fontWeight: tab === t.id ? 600 : 500,
            }}
          >
            {t.label}
          </button>
        ))}
        <span role="tab" aria-disabled="true" className="flex h-9 flex-grow cursor-not-allowed items-center justify-center gap-2 text-[12.5px] text-app-text-faint">
          Mã Python <WillBadge />
        </span>
      </div>

      {tab === "results" ? (
        <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
          {!screen ? (
            <p className="m-0 text-[12.5px] text-app-text-muted">Hỏi Quant một điều kiện lọc để xem các mã đạt ở đây.</p>
          ) : (
            <>
              <div className="flex items-start justify-between gap-3">
                <div className="flex flex-col gap-[2px]">
                  <h2 className="m-0 text-[15px] font-semibold">{matches.length} mã đạt điều kiện</h2>
                  <span className="text-[11.5px] text-app-text-muted">{screen.sortedBy === "rsi14" ? "Sắp theo RSI tăng dần" : "Sắp theo mã"}</span>
                </div>
                <button
                  type="button"
                  onClick={exportCsv}
                  disabled={!matches.length}
                  className="h-8 rounded-lg border border-app-border px-3 text-[12px] text-app-text-3 disabled:opacity-50"
                >
                  Xuất CSV
                </button>
              </div>
              <div className="min-h-0 flex-1 overflow-auto">
                <table className="w-full border-collapse text-[12.5px]">
                  <thead>
                    <tr className="border-b border-app-hairline text-[11px] text-app-text-muted">
                      <th scope="col" className="py-2 text-left font-medium">Mã</th>
                      <th scope="col" className="py-2 text-right font-medium">Giá</th>
                      <th scope="col" className="py-2 text-right font-medium">RSI</th>
                      <th scope="col" className="py-2 text-right font-medium">ROE</th>
                      <th scope="col" className="py-2 text-right font-medium">KLTB20</th>
                    </tr>
                  </thead>
                  <tbody>
                    {matches.map((r) => (
                      <tr key={r.symbol} className="border-b border-app-hairline">
                        <td className="py-[8.5px]">
                          <Link href={`/stocks/${r.symbol}`} className="font-plex-mono font-semibold text-app-text">
                            {r.symbol}
                          </Link>
                        </td>
                        <td className="py-[8.5px] text-right font-plex-mono">{formatThousandsVN(r.price)}</td>
                        <td className="py-[8.5px] text-right font-plex-mono font-semibold" style={{ color: r.rsi14 < 30 ? "#C08BFF" : "#F0C243" }}>
                          {formatVN(r.rsi14, 1)}
                        </td>
                        <td className="py-[8.5px] text-right font-plex-mono">{formatVN(r.roe, 1)}%</td>
                        <td className="py-[8.5px] text-right font-plex-mono text-app-text-3">{formatVolumeVN(r.avgVolume20)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {!matches.length && <p className="text-[12.5px] text-app-text-muted">Không có mã nào đạt đủ các điều kiện.</p>}
              </div>
              <div className="flex flex-col gap-2 border-t border-app-hairline pt-3">
                <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Điều kiện đang áp dụng</span>
                <div className="flex flex-wrap gap-[6px]">
                  {screen.labels.map((l) => (
                    <span key={l} className="rounded-md bg-app-surface-3 px-2 py-[3px] font-plex-mono text-[11.5px] text-app-text-2">
                      {l}
                    </span>
                  ))}
                </div>
              </div>
              <div className="flex flex-col gap-2">
                <button
                  type="button"
                  onClick={backtestAll}
                  disabled={!strategy || !matches.length || pending}
                  title={strategy ? undefined : "Hãy nhờ Quant dựng một chiến lược trước"}
                  className="h-10 rounded-[10px] bg-app-accent text-[13px] font-semibold text-app-accent-ink disabled:opacity-50"
                >
                  {pending ? "Đang xử lý…" : `Kiểm thử chiến lược trên ${Math.min(matches.length, 10)} mã`}
                </button>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={addToWatchlist}
                    disabled={!matches.length || pending}
                    className="h-10 flex-1 rounded-[10px] border border-app-border text-[13px] text-app-text disabled:opacity-50"
                  >
                    Thêm vào theo dõi
                  </button>
                  {matches[0] && (
                    <Link
                      href={`/replay?symbol=${encodeURIComponent(matches[0].symbol)}`}
                      className="flex h-10 flex-1 items-center justify-center rounded-[10px] border border-app-border text-[13px] text-app-text"
                    >
                      Mở bằng Replay
                    </Link>
                  )}
                </div>
                {notice && <p className="m-0 text-[12px] text-app-text-3" role="status">{notice}</p>}
              </div>
            </>
          )}
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-4">
          {!strategy ? (
            <p className="m-0 text-[12.5px] text-app-text-muted">Chưa có chiến lược nháp. Thử: “Tạo chiến lược mua khi RSI cắt lên 35, bán khi RSI vượt 70 hoặc lỗ 7%”.</p>
          ) : (
            <>
              <span className="font-display text-[16px] font-semibold">“{strategy.name || "Chiến lược của bạn"}”</span>
              {strategyRules(strategy).map((r) => (
                <div key={r.tag} className="flex items-center gap-[10px]">
                  <span className="w-[52px] shrink-0 rounded-md py-[3px] text-center text-[11px] font-semibold" style={{ color: r.color, background: r.bg }}>
                    {r.tag}
                  </span>
                  <span className="font-plex-mono text-[12.5px] text-app-text-2">{r.expr}</span>
                </div>
              ))}
              {multi.length > 0 && (
                <table className="mt-2 w-full border-collapse text-[12.5px]">
                  <thead>
                    <tr className="border-b border-app-hairline text-[11px] text-app-text-muted">
                      <th scope="col" className="py-2 text-left font-medium">Mã</th>
                      <th scope="col" className="py-2 text-right font-medium">Lợi nhuận {years} năm</th>
                      <th scope="col" className="py-2 text-right font-medium">Thắng</th>
                      <th scope="col" className="py-2 text-right font-medium">Lệnh</th>
                    </tr>
                  </thead>
                  <tbody>
                    {multi.map((m) => (
                      <tr key={m.symbol} className="border-b border-app-hairline">
                        <td className="py-2 font-plex-mono font-semibold">{m.symbol}</td>
                        {m.result ? (
                          <>
                            <td className="py-2 text-right font-plex-mono" style={{ color: (m.result.returnPercent ?? 0) >= 0 ? "#35C77F" : "#FF5C5C" }}>
                              {signVN(m.result.returnPercent ?? 0, 2)}%
                            </td>
                            <td className="py-2 text-right font-plex-mono">{formatVN(m.result.winRate ?? 0, 1)}%</td>
                            <td className="py-2 text-right font-plex-mono">{m.result.totalTrades ?? 0}</td>
                          </>
                        ) : (
                          <td colSpan={3} className="py-2 text-right text-price-down">{m.error}</td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </>
          )}
        </div>
      )}
    </aside>
  );
}
