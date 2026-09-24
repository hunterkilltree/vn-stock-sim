import type { ReactNode } from "react";
import type { Bar, IndicatorPoint } from "@/lib/api";
import { formatVN } from "@/lib/format";

type Fill = { barIndex: number; side: string };

type Props = {
  bars: Bar[];
  sma20: IndicatorPoint[];
  rsi?: IndicatorPoint[];
  // Replay: the chart has `slots` candle positions but only the first
  // `revealed` are drawn; the rest is the "CHƯA MỞ" mask.
  slots?: number;
  revealed?: number;
  fills?: Fill[];
  stopLoss?: number;
  scale?: number;
  decimals?: number;
  ariaLabel: string;
  headLeft?: ReactNode;
  headRight?: ReactNode;
  footer?: ReactNode;
};

const W = 354; // Mobile-Detail/Mobile-Replay viewBox width
const PLOT_W = 308; // candles stop here; price labels sit to the right
const PRICE_H = 196; // candle area; volume sits below it
const H = 250;

// The phone chart from design/screens/Mobile-Detail.dc.html and
// Mobile-Replay.dc.html, drawn in their own 354-unit space (phase-j.md
// decision 7) so labels stay legible, instead of shrinking the 924-unit
// desktop drawing to a third of its size.
export default function CompactChart({
  bars,
  sma20,
  rsi,
  slots,
  revealed,
  fills = [],
  stopLoss,
  scale = 1000,
  decimals = 2,
  ariaLabel,
  headLeft,
  headRight,
  footer,
}: Props) {
  if (bars.length < 2) {
    return <p className="py-10 text-center text-sm text-app-text-muted">Chưa có dữ liệu biểu đồ.</p>;
  }
  const toK = (v: number) => v / scale;
  const n = slots ?? bars.length;
  const step = PLOT_W / n;
  const bw = Math.max(1, step * 0.62);
  const x = (i: number) => i * step + step / 2;
  const index = new Map(bars.map((b, i) => [b.time, i]));

  const highs = bars.map((b) => toK(b.high));
  const lows = bars.map((b) => toK(b.low));
  const stop = stopLoss ? toK(stopLoss) : null;
  const hi = Math.max(...highs, stop ?? -Infinity) * 1.01;
  const lo = Math.min(...lows, stop ?? Infinity) * 0.99;
  const py = (v: number) => 8 + ((hi - v) / (hi - lo || 1)) * (PRICE_H - 16);

  const grid = Array.from({ length: 4 }, (_, i) => {
    const v = lo + ((hi - lo) * (i + 0.5)) / 4;
    return { y: py(v), label: formatVN(v, decimals === 2 ? 1 : decimals) };
  });
  const maxV = Math.max(...bars.map((b) => b.volume), 1);
  const sma = sma20
    .map((p) => {
      const i = index.get(p.time);
      return i === undefined ? null : `${x(i).toFixed(1)},${py(toK(p.value)).toFixed(1)}`;
    })
    .filter(Boolean)
    .join(" ");

  const maskX = revealed !== undefined ? revealed * step : null;
  const last = bars[bars.length - 1];
  const lastY = py(toK(last.close));

  const rsiLast = rsi && rsi.length > 0 ? rsi[rsi.length - 1].value : null;
  const rsiPts =
    rsi && rsi.length > 1
      ? rsi
          .map((p) => {
            const i = index.get(p.time);
            return i === undefined ? null : `${((i / (bars.length - 1)) * 260).toFixed(1)},${Math.min(29, Math.max(1, 6 + ((70 - p.value) / 40) * 18)).toFixed(1)}`;
          })
          .filter(Boolean)
          .join(" ")
      : "";

  return (
    <section className="flex flex-col gap-2 rounded-[14px] border border-app-border bg-app-surface p-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-[10px] font-plex-mono text-[10.5px] text-app-text-3">
          {headLeft ?? (
            <>
              <span>O {formatVN(toK(last.open), decimals)}</span>
              <span>H {formatVN(toK(last.high), decimals)}</span>
              <span>L {formatVN(toK(last.low), decimals)}</span>
            </>
          )}
        </div>
        {headRight ?? (
          <span className="flex shrink-0 items-center gap-[5px] text-[10px] text-app-text-muted">
            <span className="h-[2px] w-3 rounded-sm bg-app-accent" />
            SMA 20
          </span>
        )}
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} width="100%" fill="none" role="img" aria-label={ariaLabel} className="h-auto">
        {grid.map((g) => (
          <g key={g.y}>
            <line x1="0" y1={g.y.toFixed(1)} x2={PLOT_W} y2={g.y.toFixed(1)} stroke="#23231F" strokeWidth="1" />
            <text x={W} y={(g.y + 3.3).toFixed(1)} textAnchor="end" fill="#8A867E" fontFamily="'IBM Plex Mono', monospace" fontSize="9.5">
              {g.label}
            </text>
          </g>
        ))}
        {maskX !== null && maskX < PLOT_W && (
          <>
            <rect x={maskX.toFixed(1)} y="0" width={(PLOT_W - maskX).toFixed(1)} height={PRICE_H} fill="#121210" />
            <line x1={maskX.toFixed(1)} y1="0" x2={maskX.toFixed(1)} y2={PRICE_H} stroke="#4A3521" strokeWidth="1.3" strokeDasharray="4 4" />
            {PLOT_W - maskX > 50 && (
              <text
                x={((maskX + PLOT_W) / 2).toFixed(1)}
                y="100"
                textAnchor="middle"
                fill="#57534A"
                fontFamily="'Be Vietnam Pro', sans-serif"
                fontSize="9.5"
                fontWeight="600"
                letterSpacing="0.1em"
              >
                CHƯA MỞ
              </text>
            )}
          </>
        )}
        {bars.map((b, i) => {
          const o = toK(b.open);
          const c = toK(b.close);
          const col = c >= o ? "#35C77F" : "#FF5C5C";
          const top = py(Math.max(o, c));
          const cx = x(i);
          const vh = (b.volume / maxV) * 40;
          return (
            <g key={b.time}>
              <path d={`M${cx.toFixed(1)} ${py(highs[i]).toFixed(1)}V${py(lows[i]).toFixed(1)}`} stroke={col} strokeWidth="1.1" />
              <rect x={(cx - bw / 2).toFixed(1)} y={top.toFixed(1)} width={bw.toFixed(1)} height={Math.max(1.2, py(Math.min(o, c)) - top).toFixed(1)} fill={col} rx="0.6" />
              <rect x={(cx - bw / 2).toFixed(1)} y={(H - 2 - vh).toFixed(1)} width={bw.toFixed(1)} height={Math.max(1, vh).toFixed(1)} fill={col} opacity="0.6" rx="0.6" />
            </g>
          );
        })}
        <polyline points={sma} stroke="#E08A3C" strokeWidth="1.5" strokeLinejoin="round" fill="none" />
        {fills.map((f) => {
          const bar = bars[f.barIndex];
          if (!bar) return null;
          const cx = x(f.barIndex);
          const buy = f.side === "buy";
          const cy = buy ? py(lows[f.barIndex]) + 9 : py(highs[f.barIndex]) - 9;
          const s = 4.5;
          const d = buy ? `M${cx} ${cy - s}L${cx - s} ${cy + s}L${cx + s} ${cy + s}Z` : `M${cx} ${cy + s}L${cx - s} ${cy - s}L${cx + s} ${cy - s}Z`;
          return <path key={`${f.barIndex}-${f.side}`} d={d} fill={buy ? "#35C77F" : "#FF5C5C"} />;
        })}
        {stop !== null && (
          <line x1="0" y1={py(stop).toFixed(1)} x2={(maskX ?? PLOT_W).toFixed(1)} y2={py(stop).toFixed(1)} stroke="#FF5C5C" strokeWidth="1" strokeDasharray="3 3" />
        )}
        {maskX === null && <line x1="0" y1={lastY.toFixed(1)} x2={PLOT_W} y2={lastY.toFixed(1)} stroke="#35C77F" strokeWidth="1" strokeDasharray="3 3" />}
      </svg>
      {rsiLast !== null && (
        <div className="flex items-center gap-[10px] border-t border-app-hairline pt-2">
          <span className="shrink-0 text-[10px] uppercase tracking-[0.06em] text-app-text-muted">RSI 14</span>
          <svg viewBox="0 0 260 30" width="100%" height="30" preserveAspectRatio="none" fill="none" role="img" aria-label={`RSI 14 phiên đang ở mức ${formatVN(rsiLast, 1)}`}>
            {/* dashed lines at RSI 70 (y 6) and 30 (y 24) */}
            <line x1="0" y1="6" x2="260" y2="6" stroke="#2C2C28" strokeWidth="1" strokeDasharray="3 3" />
            <line x1="0" y1="24" x2="260" y2="24" stroke="#2C2C28" strokeWidth="1" strokeDasharray="3 3" />
            <polyline points={rsiPts} stroke="#C08BFF" strokeWidth="1.5" strokeLinejoin="round" fill="none" />
          </svg>
          <span className="shrink-0 font-plex-mono text-[11px] text-app-rsi">{formatVN(rsiLast, 1)}</span>
        </div>
      )}
      {footer}
    </section>
  );
}
