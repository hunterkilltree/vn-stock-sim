"use client";

import Link from "next/link";
import WillBadge from "@/components/WillBadge";
import { formatVN, signVN } from "@/lib/format";
import type { BacktestResult, QuantChatResponse, QuantStrategy } from "@/lib/quantTypes";

export function SparkIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="#E08A3C" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 3l2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5z" />
    </svg>
  );
}

// Quant.dc.html's Mua / Bán / Cắt lỗ rule rows, derived from the
// validated strategy the backend returned.
export function strategyRules(s: QuantStrategy): { tag: string; expr: string; color: string; bg: string }[] {
  const buy = { color: "#35C77F", bg: "rgba(53, 199, 127, 0.14)" };
  const sell = { color: "#FF5C5C", bg: "rgba(255, 92, 92, 0.14)" };
  const stop = { color: "#F0C243", bg: "rgba(240, 194, 67, 0.14)" };
  if (s.kind === "ema_crossover") {
    return [
      { tag: "Mua", expr: `EMA(${s.fast}) cắt lên EMA(${s.slow})`, ...buy },
      { tag: "Bán", expr: `EMA(${s.fast}) cắt xuống EMA(${s.slow})`, ...sell },
    ];
  }
  const rows = [
    { tag: "Mua", expr: `RSI(14) cắt lên ${formatVN(s.rsiEntry, 0)}${s.trendSma ? ` và giá > SMA(${s.trendSma})` : ""}`, ...buy },
    { tag: "Bán", expr: `RSI(14) > ${formatVN(s.rsiExit, 0)}`, ...sell },
  ];
  if (s.stopLossPercent > 0) {
    rows.push({ tag: "Cắt lỗ", expr: `giá ≤ giá vào × ${formatVN(1 - s.stopLossPercent / 100, 2)}`, ...stop });
  }
  return rows;
}

type Props = {
  response: QuantChatResponse;
  backtest?: BacktestResult | null;
  backtestError?: string | null;
  backtestSymbol: string | null;
  years: number;
  runningBacktest: boolean;
  onRunBacktest: () => void;
};

export default function QuantAssistantCard({ response, backtest, backtestError, backtestSymbol, years, runningBacktest, onRunBacktest }: Props) {
  const { screen, strategy } = response;
  const tokens = response.usage.inputTokens + response.usage.outputTokens;

  return (
    <div className="flex gap-3">
      <span className="mt-[2px] flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-[9px] bg-app-surface-3">
        <SparkIcon />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <p className="m-0 whitespace-pre-wrap text-[13.5px] leading-[1.6] text-app-text-2">{response.reply}</p>

        {screen && (
          <>
            <div className="flex flex-wrap gap-[7px]">
              {screen.labels.map((l) => (
                <span key={l} className="rounded-lg border border-app-border bg-app-surface-2 px-[10px] py-[5px] font-plex-mono text-[12px] text-app-text-2">
                  {l}
                </span>
              ))}
            </div>
            <div className="flex items-center gap-3 rounded-xl border border-app-border bg-app-surface p-[12px_14px]">
              <span className="font-plex-mono text-[22px] font-semibold text-app-accent">{screen.matches.length}</span>
              <span className="text-[12.5px] text-app-text-3">
                trên {screen.universe} mã {screen.exchange === "ALL" ? "trong dữ liệu" : screen.exchange} đạt cả {screen.labels.length} điều kiện
              </span>
              <div className="flex-1" />
              <div className="flex gap-[6px]">
                {screen.matches.slice(0, 3).map((m) => (
                  <Link key={m.symbol} href={`/stocks/${m.symbol}`} className="rounded-md bg-app-surface-3 px-2 py-[3px] font-plex-mono text-[11.5px] font-semibold text-app-text">
                    {m.symbol}
                  </Link>
                ))}
              </div>
            </div>
          </>
        )}

        {response.dropped.length > 0 && (
          <ul className="m-0 flex list-none flex-col gap-1 rounded-[10px] border border-app-warn-border bg-app-warn-surface p-[9px_12px] text-[11.5px] text-app-warn-text">
            {response.dropped.map((d) => (
              <li key={d}>Không áp dụng: {d}</li>
            ))}
          </ul>
        )}

        {strategy && (
          <div className="flex flex-col gap-3 rounded-[14px] border border-app-accent-border bg-app-surface p-4">
            <div className="flex flex-col gap-[2px]">
              <span className="text-[11px] uppercase tracking-[0.09em] text-app-accent">Đã dựng chiến lược nháp</span>
              <span className="font-display text-[16px] font-semibold">“{strategy.name || "Chiến lược của bạn"}”</span>
            </div>
            <div className="flex flex-col gap-2">
              {strategyRules(strategy).map((r) => (
                <div key={r.tag} className="flex items-center gap-[10px]">
                  <span className="w-[52px] shrink-0 rounded-md py-[3px] text-center text-[11px] font-semibold" style={{ color: r.color, background: r.bg }}>
                    {r.tag}
                  </span>
                  <span className="font-plex-mono text-[12.5px] text-app-text-2">{r.expr}</span>
                </div>
              ))}
            </div>

            {backtest ? (
              <div className="grid grid-cols-4 gap-2 border-t border-app-hairline pt-3">
                {[
                  { k: `Lợi nhuận ${years} năm`, v: `${signVN(backtest.returnPercent ?? 0, 2)}%`, c: (backtest.returnPercent ?? 0) >= 0 ? "#35C77F" : "#FF5C5C" },
                  { k: "Tỷ lệ thắng", v: `${formatVN(backtest.winRate ?? 0, 2)}%`, c: "var(--app-text)" },
                  { k: "Sụt giảm tối đa", v: `${signVN(-(backtest.maxDrawdownPercent ?? 0), 2)}%`, c: "#FF5C5C" },
                  { k: "Hệ số lợi nhuận", v: backtest.profitFactor ? formatVN(backtest.profitFactor, 2) : "—", c: "var(--app-text)" },
                ].map((b) => (
                  <div key={b.k} className="flex flex-col gap-1">
                    <span className="text-[10.5px] text-app-text-muted">{b.k}</span>
                    <span className="font-plex-mono text-[15px] font-semibold" style={{ color: b.c }}>
                      {b.v}
                    </span>
                  </div>
                ))}
              </div>
            ) : backtestError ? (
              <p className="m-0 text-[12px] text-price-down">{backtestError}</p>
            ) : null}

            <div className="flex items-start gap-[9px] rounded-[10px] border border-app-warn-border bg-app-warn-surface p-[10px_12px]">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#F0C243" strokeWidth="1.8" strokeLinecap="round" className="mt-[1px] shrink-0" aria-hidden="true">
                <path d="M12 9v5M12 17h0M12 3l9 17H3L12 3z" />
              </svg>
              <p className="m-0 text-[11.5px] leading-[1.5] text-app-warn-text">
                {backtest
                  ? `Kiểm thử trên ${backtest.symbol}, ${years} năm gần nhất, nến ngày, ${backtest.totalTrades ?? 0} lệnh đã đóng. `
                  : backtestSymbol
                    ? `Sẽ kiểm thử trên ${backtestSymbol}, ${years} năm gần nhất, nến ngày. `
                    : ""}
                Quá khứ không bảo đảm tương lai và Quant không đưa ra lời khuyên đầu tư — mọi lệnh chỉ chạy trên tài khoản giấy.
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={onRunBacktest}
                disabled={runningBacktest || !backtestSymbol}
                className="h-10 rounded-[10px] bg-app-accent px-4 text-[13px] font-semibold text-app-accent-ink disabled:opacity-60"
              >
                {runningBacktest ? "Đang kiểm thử…" : backtest ? "Kiểm thử lại" : "Mở trong Kiểm thử"}
              </button>
              {backtestSymbol && (
                <Link
                  href={`/replay?symbol=${encodeURIComponent(backtestSymbol)}`}
                  className="flex h-10 items-center rounded-[10px] border border-app-border px-4 text-[13px] font-medium text-app-text"
                >
                  Chạy thử bằng Replay
                </Link>
              )}
              <span
                title="Thư viện chiến lược chưa có"
                className="flex h-10 cursor-not-allowed items-center gap-2 rounded-[10px] border border-app-border px-4 text-[13px] text-app-text-muted"
              >
                Lưu chiến lược <WillBadge />
              </span>
            </div>
          </div>
        )}

        <span className="text-[11px] text-app-text-faint">
          {response.model} · {formatVN(response.latencyMs / 1000, 1)} giây · {formatVN(tokens, 0)} token
          {!response.temperatureApplied && " · mô hình bỏ qua mức sáng tạo"}
        </span>
      </div>
    </div>
  );
}
