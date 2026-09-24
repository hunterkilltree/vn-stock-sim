import type { ReplaySession } from "@/lib/api";
import { formatCompact, formatVN, formatVolumeVN } from "@/lib/format";

type Props = {
  session: ReplaySession;
  scale?: number;
  decimals?: number;
};

// Replay.dc.html's candle chart, ported the same way DetailChart.tsx
// ports Detail.dc.html's geometry (real inline SVG, not a chart
// library -- needed here even more than on the Detail screen, since the
// "future hidden" mask and buy/sell markers have to live on the same
// canvas as the candles). One real deviation from the mockup's own
// script: the design spaces candles with a fixed *1.9 fudge factor tuned
// for its one fixed demo (44 revealed of 120 total); since totalBars is
// configurable here, this uses a plain proportional
// `step = plotW / totalBars` instead, which scales correctly for any
// session length rather than replicating a magic constant tuned for one
// specific number.
export default function ReplayChart({ session, scale = 1000, decimals = 2 }: Props) {
  const { bars, sma20, fills, stopLoss, totalBars, currentBar } = session;

  if (bars.length < 2) {
    return (
      <p className="py-16 text-center text-sm text-app-text-muted">
        Đang tải nến đầu tiên...
      </p>
    );
  }

  // scale 1000 shows VND in thousands (stocks); 1 shows USDT (crypto).
  const toK = (v: number) => v / scale;
  const fmtVol = (v: number) => (scale === 1 ? formatCompact(v) : formatVolumeVN(v));
  const plotW = 870;
  const step = plotW / totalBars;
  const bw = Math.max(1, step * 0.6);
  const x = (i: number) => i * step + step / 2;
  const timeIndex = new Map(bars.map((b, i) => [b.time, i]));

  const opens = bars.map((b) => toK(b.open));
  const highs = bars.map((b) => toK(b.high));
  const lows = bars.map((b) => toK(b.low));
  const closes = bars.map((b) => toK(b.close));

  const hi = Math.max(...highs) * 1.02;
  const lo = Math.min(...lows) * 0.97;
  const H = 372;
  const pad = 12;
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
    return { key: i, y: gy.toFixed(1), ty: (gy + 3.5).toFixed(1), label: formatVN(v, decimals === 2 ? 1 : decimals) };
  });

  const smaPoly = sma20
    .map((p) => {
      const i = timeIndex.get(p.time);
      if (i === undefined) return null;
      return `${x(i).toFixed(1)},${py(toK(p.value)).toFixed(1)}`;
    })
    .filter((s): s is string => s !== null)
    .join(" ");

  const maskX = currentBar * step;
  const maskW = Math.max(0, plotW - maskX);
  const remaining = totalBars - currentBar;

  const markers = fills.map((f) => {
    const cx = x(f.barIndex);
    const bar = bars[f.barIndex];
    if (!bar) return null;
    const up = f.side === "buy";
    const price = up ? lows[f.barIndex] : highs[f.barIndex];
    const cy = py(price) + (up ? 16 : -16);
    const s = 6.5;
    const tri = up
      ? `M${cx} ${cy - s}L${cx - s} ${cy + s}L${cx + s} ${cy + s}Z`
      : `M${cx} ${cy + s}L${cx - s} ${cy - s}L${cx + s} ${cy - s}Z`;
    return {
      key: `${f.barIndex}-${f.side}-${f.price}`,
      tri,
      col: up ? "#35C77F" : "#FF5C5C",
      tx: cx.toFixed(1),
      ty: (up ? cy + s + 12 : cy - s - 6).toFixed(1),
      label: `${up ? "M" : "B"} ${formatVN(toK(f.price), decimals === 2 ? 1 : decimals)}`,
    };
  });

  const maxV = Math.max(...bars.map((b) => b.volume), 1);
  const vol = bars.map((b, i) => {
    const h = (b.volume / maxV) * 40;
    return {
      key: b.time,
      x: (x(i) - bw / 2).toFixed(1),
      y: (42 - h).toFixed(1),
      w: bw.toFixed(1),
      h: Math.max(1.5, h).toFixed(1),
      col: closes[i] >= opens[i] ? "#35C77F" : "#FF5C5C",
    };
  });

  const stopY = stopLoss ? py(toK(stopLoss)) : null;
  const lastBar = bars[bars.length - 1];

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-[10px] rounded-2xl border border-app-border bg-app-surface p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-[14px] font-plex-mono text-[11.5px] text-app-text-3">
          <span className="text-app-text-2">
            {new Date(lastBar.time * 1000).toLocaleString(
              "vi-VN",
              session.market === "crypto"
                ? { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "UTC" }
                : { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" },
            )}
          </span>
          <span>O {formatVN(toK(lastBar.open), decimals)}</span>
          <span>H {formatVN(toK(lastBar.high), decimals)}</span>
          <span>L {formatVN(toK(lastBar.low), decimals)}</span>
          <span className={closes[closes.length - 1] >= opens[opens.length - 1] ? "text-price-up" : "text-price-down"}>
            C {formatVN(toK(lastBar.close), decimals)}
          </span>
        </div>
        <div className="flex items-center gap-[10px]">
          <span className="flex items-center gap-[6px] text-[11px] text-app-text-muted">
            <span className="h-[9px] w-[9px] rounded-[2px] bg-price-up" />
            Mua
          </span>
          <span className="flex items-center gap-[6px] text-[11px] text-app-text-muted">
            <span className="h-[9px] w-[9px] rounded-[2px] bg-price-down" />
            Bán
          </span>
        </div>
      </div>

      <svg
        viewBox="0 0 924 430"
        width="100%"
        height="430"
        fill="none"
        role="img"
        aria-label={`Biểu đồ nến của phiên Replay: ${currentBar} nến đã hiện, phần còn lại của lịch sử bị che`}
      >
        {grid.map((g) => (
          <g key={g.key}>
            <line x1="0" y1={g.y} x2="870" y2={g.y} stroke="#23231F" strokeWidth="1" />
            <text x="924" y={g.ty} textAnchor="end" fill="#8A867E" fontFamily="'IBM Plex Mono', monospace" fontSize="10.5">
              {g.label}
            </text>
          </g>
        ))}

        {maskW > 0 && (
          <>
            <rect x={maskX.toFixed(1)} y="0" width={maskW.toFixed(1)} height="372" fill="#121210" />
            <line x1={maskX.toFixed(1)} y1="0" x2={maskX.toFixed(1)} y2="372" stroke="#4A3521" strokeWidth="1.4" strokeDasharray="5 4" />
            <text
              x={(maskX + maskW / 2).toFixed(1)}
              y="186"
              textAnchor="middle"
              fill="#57534A"
              fontFamily="'Be Vietnam Pro', sans-serif"
              fontSize="14"
              fontWeight="600"
              letterSpacing="0.14em"
            >
              TƯƠNG LAI BỊ CHE
            </text>
            <text x={(maskX + maskW / 2).toFixed(1)} y="208" textAnchor="middle" fill="#3E3B34" fontFamily="'Be Vietnam Pro', sans-serif" fontSize="11.5">
              {remaining} nến còn lại
            </text>
          </>
        )}

        {candles.map((c) => (
          <g key={c.key}>
            <path d={c.wick} stroke={c.col} strokeWidth="1.4" />
            <rect x={c.x} y={c.y} width={c.w} height={c.h} fill={c.col} rx="0.8" />
          </g>
        ))}

        <polyline points={smaPoly} stroke="#E08A3C" strokeWidth="1.6" strokeLinejoin="round" fill="none" />

        {markers.map(
          (m) =>
            m && (
              <g key={m.key}>
                <path d={m.tri} fill={m.col} />
                <text x={m.tx} y={m.ty} textAnchor="middle" fill={m.col} fontFamily="'IBM Plex Mono', monospace" fontSize="10" fontWeight="600">
                  {m.label}
                </text>
              </g>
            ),
        )}

        {stopY !== null && (
          <>
            <line x1="0" y1={stopY.toFixed(1)} x2={maskX > 0 ? maskX.toFixed(1) : "870"} y2={stopY.toFixed(1)} stroke="#FF5C5C" strokeWidth="1" strokeDasharray="4 4" />
            <text x="6" y={(stopY - 6).toFixed(1)} fill="#FF5C5C" fontFamily="'IBM Plex Mono', monospace" fontSize="10">
              Cắt lỗ {formatVN(toK(stopLoss ?? 0), decimals)}
            </text>
          </>
        )}

        <g>
          {vol.map((v) => (
            <rect key={v.key} x={v.x} y={388 + Number(v.y)} width={v.w} height={v.h} fill={v.col} opacity="0.6" rx="0.8" />
          ))}
          <line x1="0" y1="430" x2="870" y2="430" stroke="#23231F" strokeWidth="1" />
        </g>
      </svg>

      <div className="text-[10.5px] text-app-text-muted">KL {fmtVol(lastBar.volume)}</div>
    </section>
  );
}
