"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { runBacktestPageAction } from "@/lib/backtestActions";
import type { BacktestRule } from "@/lib/api";

export type BacktestDefaults = {
  symbol: string;
  rule: BacktestRule;
  params: Record<string, number>;
  years: number;
  startingCapital: number;
};

const RULES: { id: BacktestRule; label: string; note: string }[] = [
  { id: "rsi_reversion", label: "RSI hồi phục", note: "Mua khi RSI cắt lên ngưỡng vào, bán khi vượt ngưỡng ra hoặc chạm cắt lỗ" },
  { id: "ema_crossover", label: "EMA cắt nhau", note: "Mua khi EMA nhanh cắt lên EMA chậm, bán khi cắt xuống" },
];

const YEARS = [1, 3, 5];
const CAPITALS = [
  { amount: 100_000_000, label: "100 tr ₫" },
  { amount: 1_000_000_000, label: "1 tỷ ₫" },
];

// One numeric rule parameter, bounded like the engine's defaults.
function NumberField({ id, label, value, onChange, min, max }: { id: string; label: string; value: number; onChange: (v: number) => void; min: number; max: number }) {
  return (
    <label htmlFor={id} className="flex min-w-0 flex-col gap-[6px]">
      <span className="text-[11px] text-app-text-muted">{label}</span>
      <input
        id={id}
        type="number"
        inputMode="numeric"
        min={min}
        max={max}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="box-border h-11 w-full rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none"
      />
    </label>
  );
}

// The /backtest form (phase-k.md decision 14): symbol, rule and its
// parameters, range and capital. Runs on the existing engine and then
// navigates to the result by id, so every run has its own URL.
export default function BacktestForm({ defaults }: { defaults: BacktestDefaults }) {
  const router = useRouter();
  const [symbol, setSymbol] = useState(defaults.symbol);
  const [rule, setRule] = useState<BacktestRule>(defaults.rule);
  const p = defaults.params;
  const [fast, setFast] = useState(p.fast ?? 20);
  const [slow, setSlow] = useState(p.slow ?? 50);
  const [period, setPeriod] = useState(p.period ?? 14);
  const [entry, setEntry] = useState(p.entry ?? 35);
  const [exit, setExit] = useState(p.exit ?? 70);
  const [stop, setStop] = useState(p.stopLossPercent ?? 7);
  const [trend, setTrend] = useState(p.trendSma ?? 0);
  const [years, setYears] = useState(defaults.years);
  const [capital, setCapital] = useState(defaults.startingCapital);
  const [error, setError] = useState<string | null>(null);
  const [pending, start] = useTransition();

  function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const params: Record<string, number> =
      rule === "ema_crossover"
        ? { fast, slow }
        : { period, entry, exit, stopLossPercent: stop, trendSma: trend };
    if (rule === "ema_crossover" && fast >= slow) {
      setError("EMA nhanh phải ngắn hơn EMA chậm.");
      return;
    }
    if (rule === "rsi_reversion" && entry >= exit) {
      setError("Ngưỡng vào phải thấp hơn ngưỡng ra.");
      return;
    }
    start(async () => {
      const res = await runBacktestPageAction({ symbol, rule, params, years, startingCapital: capital });
      if (res.error || !res.id) {
        setError(res.error ?? "Kiểm thử thất bại.");
        return;
      }
      router.push(`/backtest?id=${res.id}`);
    });
  }

  const pill = (on: boolean) =>
    on
      ? { borderColor: "var(--app-border-strong)", background: "var(--app-border)", color: "var(--app-text)", fontWeight: 600 }
      : { borderColor: "var(--app-border)", background: "var(--app-surface-2)", color: "var(--app-text-muted)", fontWeight: 500 };

  return (
    <form onSubmit={submit} className="flex flex-col gap-4 rounded-2xl border border-app-border bg-app-surface p-[18px]">
      <h2 className="m-0 text-[15px] font-semibold">Thiết lập kiểm thử</h2>

      <label htmlFor="bt-symbol" className="flex flex-col gap-[6px]">
        <span className="text-[11px] text-app-text-muted">Mã cổ phiếu</span>
        <input
          id="bt-symbol"
          value={symbol}
          onChange={(e) => setSymbol(e.target.value.toUpperCase())}
          maxLength={10}
          className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm uppercase text-app-text outline-none"
        />
      </label>

      <div role="radiogroup" aria-label="Quy tắc giao dịch" className="flex flex-col gap-2">
        <span className="text-[11px] text-app-text-muted">Quy tắc</span>
        {RULES.map((r) => (
          <button
            key={r.id}
            type="button"
            role="radio"
            aria-checked={rule === r.id}
            onClick={() => setRule(r.id)}
            className="flex flex-col items-start gap-[2px] rounded-xl border p-[11px_13px] text-left"
            style={{
              borderColor: rule === r.id ? "var(--app-accent)" : "var(--app-border)",
              background: rule === r.id ? "var(--app-accent-surface)" : "var(--app-surface-2)",
            }}
          >
            <span className="text-[13px] font-semibold text-app-text">{r.label}</span>
            <span className="text-[11.5px] leading-[1.45] text-app-text-muted">{r.note}</span>
          </button>
        ))}
      </div>

      {rule === "ema_crossover" ? (
        <div className="grid grid-cols-2 gap-3">
          <NumberField id="bt-fast" label="EMA nhanh (phiên)" value={fast} onChange={setFast} min={2} max={100} />
          <NumberField id="bt-slow" label="EMA chậm (phiên)" value={slow} onChange={setSlow} min={3} max={250} />
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          <NumberField id="bt-period" label="RSI (phiên)" value={period} onChange={setPeriod} min={2} max={50} />
          <NumberField id="bt-stop" label="Cắt lỗ (%, 0 = không)" value={stop} onChange={setStop} min={0} max={50} />
          <NumberField id="bt-entry" label="Ngưỡng vào (RSI)" value={entry} onChange={setEntry} min={5} max={60} />
          <NumberField id="bt-exit" label="Ngưỡng ra (RSI)" value={exit} onChange={setExit} min={40} max={95} />
          <label htmlFor="bt-trend" className="col-span-2 flex flex-col gap-[6px]">
            <span className="text-[11px] text-app-text-muted">Lọc xu hướng</span>
            <select
              id="bt-trend"
              value={trend}
              onChange={(e) => setTrend(Number(e.target.value))}
              className="box-border h-11 rounded-[10px] border border-app-border bg-app-surface-2 px-3 text-[13px] text-app-text outline-none"
            >
              <option value={0}>Không lọc</option>
              <option value={50}>Chỉ mua khi giá trên SMA 50</option>
              <option value={200}>Chỉ mua khi giá trên SMA 200</option>
            </select>
          </label>
        </div>
      )}

      <div className="flex flex-col gap-[6px]">
        <span className="text-[11px] text-app-text-muted">Khoảng thời gian</span>
        <div className="grid grid-cols-3 gap-2">
          {YEARS.map((y) => (
            <button key={y} type="button" aria-pressed={years === y} onClick={() => setYears(y)} className="h-10 rounded-[10px] border text-[12.5px]" style={pill(years === y)}>
              {y} năm
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-[6px]">
        <span className="text-[11px] text-app-text-muted">Vốn ban đầu</span>
        <div className="grid grid-cols-2 gap-2">
          {CAPITALS.map((c) => (
            <button
              key={c.amount}
              type="button"
              aria-pressed={capital === c.amount}
              onClick={() => setCapital(c.amount)}
              className="h-10 rounded-[10px] border font-plex-mono text-[12.5px]"
              style={pill(capital === c.amount)}
            >
              {c.label}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <p role="alert" className="m-0 rounded-lg border border-app-warn-border bg-app-warn-surface px-3 py-2 text-xs text-app-warn-text">
          {error}
        </p>
      )}
      <button
        type="submit"
        disabled={pending || !symbol.trim()}
        className="h-12 rounded-[11px] bg-app-accent text-[14.5px] font-semibold text-app-accent-ink disabled:opacity-60"
      >
        {pending ? "Đang kiểm thử…" : "Chạy kiểm thử"}
      </button>
      <p className="m-0 text-[11px] leading-[1.5] text-app-text-faint">
        Nến ngày, mua/bán toàn bộ vốn ở giá đóng cửa, chưa tính phí giao dịch. Quá khứ không bảo đảm tương lai.
      </p>
    </form>
  );
}
