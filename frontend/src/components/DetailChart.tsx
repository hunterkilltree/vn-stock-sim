import type { Bar, IndicatorPoint, IndicatorMultiPoint } from "@/lib/api";
import { formatVN, formatVolumeVN } from "@/lib/format";

type Props = {
  bars: Bar[];
  sma20: IndicatorPoint[];
  sma50: IndicatorPoint[];
  rsi: IndicatorPoint[];
  macd: IndicatorMultiPoint[];
};

// Full inline-SVG chart (candles + SMA20/50 + volume + RSI(14) +
// MACD(12,26,9)), geometry ported directly from
// design/screens/Detail.dc.html's script block -- design/README.md
// notes this math "is real and can be lifted directly." Real backend
// data throughout, not the design's synthetic random walk. All prices
// (candles, SMA, MACD) are converted to thousands of VND before any
// geometry math, matching the Detail screen's own "Gia (nghin d)"
// display convention (see lib/format.ts's formatThousandsVN) -- RSI
// (0-100) and volume (raw share counts) are left unconverted.
export default function DetailChart({ bars, sma20, sma50, rsi, macd }: Props) {
  if (bars.length < 2) {
    return (
      <p className="py-16 text-center text-sm text-app-text-muted">
        Chart data unavailable right now — the backend may be starting up or unreachable.
      </p>
    );
  }

  const toK = (v: number) => v / 1000;
  const n = bars.length;
  const plotW = 870;
  const step = plotW / n;
  const bw = Math.max(1, step * 0.62);
  const x = (i: number) => i * step + step / 2;
  const timeIndex = new Map(bars.map((b, i) => [b.time, i]));

  const opens = bars.map((b) => toK(b.open));
  const highs = bars.map((b) => toK(b.high));
  const lows = bars.map((b) => toK(b.low));
  const closes = bars.map((b) => toK(b.close));

  const hi = Math.max(...highs) * 1.008;
  const lo = Math.min(...lows) * 0.992;
  const H = 372;
  const pad = 10;
  const py = (v: number) => pad + ((hi - v) / (hi - lo)) * (H - pad * 2);

  const candles = bars.map((b, i) => {
    const up = closes[i] >= opens[i];
    const col = up ? "#35C77F" : "#FF5C5C";
    const top = py(Math.max(opens[i], closes[i]));
    const bot = py(Math.min(opens[i], closes[i]));
    const cx = x(i);
    return {
      key: b.time,
      col,
      wick: `M${cx.toFixed(1)} ${py(highs[i]).toFixed(1)}V${py(lows[i]).toFixed(1)}`,
      x: (cx - bw / 2).toFixed(1),
      y: top.toFixed(1),
      w: bw.toFixed(1),
      h: Math.max(1.6, bot - top).toFixed(1),
    };
  });

  const grid = Array.from({ length: 5 }, (_, i) => {
    const v = lo + ((hi - lo) * i) / 4;
    const gy = py(v);
    return { key: i, y: gy.toFixed(1), ty: (gy + 3.5).toFixed(1), label: formatVN(v, 1) };
  });

  const poly = (points: IndicatorPoint[], scale: (v: number) => number, convert: (v: number) => number = toK) =>
    points
      .map((p) => {
        const i = timeIndex.get(p.time);
        if (i === undefined) return null;
        return `${x(i).toFixed(1)},${scale(convert(p.value)).toFixed(1)}`;
      })
      .filter((s): s is string => s !== null)
      .join(" ");

  const maxV = Math.max(...bars.map((b) => b.volume), 1);
  const vol = bars.map((b, i) => {
    const h = (b.volume / maxV) * 62;
    return {
      key: b.time,
      x: (x(i) - bw / 2).toFixed(1),
      y: (72 - h).toFixed(1),
      w: bw.toFixed(1),
      h: Math.max(1.5, h).toFixed(1),
      col: closes[i] >= opens[i] ? "#35C77F" : "#FF5C5C",
    };
  });

  const rsiScale = (v: number) => 96 - 8 - (v / 100) * 80;
  const rsiLast = rsi.length > 0 ? rsi[rsi.length - 1].value : null;

  const macdAbs = Math.max(1, ...macd.flatMap((m) => [Math.abs(toK(m.values.macd ?? 0)), Math.abs(toK(m.values.histogram ?? 0))]));
  const mScale = (v: number) => 48 - (v / macdAbs) * 40;
  const hist = macd
    .map((m) => {
      const i = timeIndex.get(m.time);
      if (i === undefined) return null;
      const v = toK(m.values.histogram ?? 0);
      const y0 = mScale(0);
      const y1 = mScale(v);
      return {
        key: m.time,
        x: (x(i) - bw / 2).toFixed(1),
        y: Math.min(y0, y1).toFixed(1),
        w: bw.toFixed(1),
        h: Math.max(1.2, Math.abs(y1 - y0)).toFixed(1),
        col: v >= 0 ? "#35C77F" : "#FF5C5C",
      };
    })
    .filter((h): h is NonNullable<typeof h> => h !== null);
  const macdPoly = macd
    .map((m) => {
      const i = timeIndex.get(m.time);
      if (i === undefined) return null;
      return `${x(i).toFixed(1)},${mScale(toK(m.values.macd ?? 0)).toFixed(1)}`;
    })
    .filter((s): s is string => s !== null)
    .join(" ");
  const signalPoly = macd
    .map((m) => {
      const i = timeIndex.get(m.time);
      if (i === undefined) return null;
      return `${x(i).toFixed(1)},${mScale(toK(m.values.signal ?? 0)).toFixed(1)}`;
    })
    .filter((s): s is string => s !== null)
    .join(" ");
  const macdLast = macd.length > 0 ? toK(macd[macd.length - 1].values.macd ?? 0) : null;

  const lastClose = closes[closes.length - 1];
  const lastY = py(lastClose);
  const lastBar = bars[bars.length - 1];

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-[10px] rounded-2xl border border-app-border bg-app-surface p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-[14px] font-plex-mono text-[11.5px] text-app-text-3">
          <span>O {formatVN(toK(lastBar.open), 2)}</span>
          <span>H {formatVN(toK(lastBar.high), 2)}</span>
          <span>L {formatVN(toK(lastBar.low), 2)}</span>
          <span className="text-price-up">C {formatVN(toK(lastBar.close), 2)}</span>
          <span className="text-app-text-muted">KL {formatVolumeVN(lastBar.volume)}</span>
        </div>
        <span className="rounded-[5px] border border-app-border px-2 py-[2px] text-[10px] uppercase tracking-[0.06em] text-app-text-muted">
          Dữ liệu mẫu
        </span>
      </div>

      <svg viewBox="0 0 924 372" width="100%" height="372" fill="none" role="img" aria-label={`Biểu đồ nến ngày với hai đường trung bình động SMA 20 và SMA 50`}>
        {grid.map((g) => (
          <g key={g.key}>
            <line x1="0" y1={g.y} x2="870" y2={g.y} stroke="#23231F" strokeWidth="1" />
            <text x="924" y={g.ty} textAnchor="end" fill="#8A867E" fontFamily="'IBM Plex Mono', monospace" fontSize="10.5">
              {g.label}
            </text>
          </g>
        ))}
        {candles.map((c) => (
          <g key={c.key}>
            <path d={c.wick} stroke={c.col} strokeWidth="1.3" />
            <rect x={c.x} y={c.y} width={c.w} height={c.h} fill={c.col} rx="0.8" />
          </g>
        ))}
        <polyline points={poly(sma20, py)} stroke="#E08A3C" strokeWidth="1.6" strokeLinejoin="round" fill="none" />
        <polyline points={poly(sma50, py)} stroke="#7FA2FF" strokeWidth="1.6" strokeLinejoin="round" fill="none" />
        <line x1="0" y1={lastY.toFixed(1)} x2="870" y2={lastY.toFixed(1)} stroke="#35C77F" strokeWidth="1" strokeDasharray="3 3" />
        <rect x="874" y={(lastY - 8.5).toFixed(1)} width="50" height="17" fill="#35C77F" rx="3" />
        <text x="899" y={(lastY + 3.7).toFixed(1)} textAnchor="middle" fill="#0F0F0E" fontFamily="'IBM Plex Mono', monospace" fontSize="10.5" fontWeight="600">
          {formatVN(lastClose, 2)}
        </text>
      </svg>

      <div className="flex flex-col gap-1">
        <span className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">Khối lượng</span>
        <svg viewBox="0 0 924 74" width="100%" height="74" fill="none" role="img" aria-label="Biểu đồ khối lượng giao dịch theo phiên">
          {vol.map((v) => (
            <rect key={v.key} x={v.x} y={v.y} width={v.w} height={v.h} fill={v.col} opacity="0.65" rx="0.8" />
          ))}
          <line x1="0" y1="72" x2="870" y2="72" stroke="#23231F" strokeWidth="1" />
        </svg>
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-[10px]">
          <span className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">RSI (14)</span>
          {rsiLast !== null && <span className="font-plex-mono text-[11px] text-app-rsi">{formatVN(rsiLast, 1)}</span>}
        </div>
        <svg viewBox="0 0 924 96" width="100%" height="96" fill="none" role="img" aria-label="Chỉ báo RSI 14 phiên với vùng quá mua 70 và quá bán 30">
          <rect x="0" y="16" width="870" height="48" fill="#1F1F1C" />
          <line x1="0" y1="16" x2="870" y2="16" stroke="#2C2C28" strokeWidth="1" strokeDasharray="3 3" />
          <line x1="0" y1="64" x2="870" y2="64" stroke="#2C2C28" strokeWidth="1" strokeDasharray="3 3" />
          <text x="924" y="20" textAnchor="end" fill="#8A867E" fontFamily="'IBM Plex Mono', monospace" fontSize="10">70</text>
          <text x="924" y="68" textAnchor="end" fill="#8A867E" fontFamily="'IBM Plex Mono', monospace" fontSize="10">30</text>
          <polyline points={poly(rsi, rsiScale, (v) => v)} stroke="#C08BFF" strokeWidth="1.6" strokeLinejoin="round" fill="none" />
        </svg>
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-[10px]">
          <span className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">MACD (12, 26, 9)</span>
          {macdLast !== null && (
            <span className="font-plex-mono text-[11px] text-price-up">
              {macdLast >= 0 ? "+" : "−"}
              {formatVN(Math.abs(macdLast), 2)}
            </span>
          )}
        </div>
        <svg viewBox="0 0 924 96" width="100%" height="96" fill="none" role="img" aria-label="Chỉ báo MACD với đường tín hiệu và biểu đồ histogram">
          <line x1="0" y1="48" x2="870" y2="48" stroke="#23231F" strokeWidth="1" />
          {hist.map((h) => (
            <rect key={h.key} x={h.x} y={h.y} width={h.w} height={h.h} fill={h.col} opacity="0.55" rx="0.8" />
          ))}
          <polyline points={macdPoly} stroke="#4FD3E8" strokeWidth="1.6" strokeLinejoin="round" fill="none" />
          <polyline points={signalPoly} stroke="#E08A3C" strokeWidth="1.4" strokeLinejoin="round" fill="none" />
        </svg>
      </div>
    </section>
  );
}
