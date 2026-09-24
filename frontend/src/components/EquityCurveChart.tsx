"use client";

import { useMemo, useState } from "react";
import type { EquityPoint } from "@/lib/api";
import { signVN } from "@/lib/format";

type Props = {
  points: EquityPoint[];
  startingCapital: number;
};

// Pulled out of the component body -- eslint-config-next's
// react-hooks/purity rule flags Date.now() called directly during
// render, the same class of fix this repo already applied once before
// (see RESUME.md's 2026-09-21 entry).
function nowMillis(): number {
  return Date.now();
}

const RANGES: { label: string; days: number | null }[] = [
  { label: "7 ngày", days: 7 },
  { label: "30 ngày", days: 30 },
  { label: "Tất cả", days: null },
];

const W = 944;
const H = 200;
const PAD = 14;

// Portfolio.dc.html's equity-curve panel, ported to real data instead of
// the design's fake random walk (see phase-e.md item 1/4). V1 has no
// historical VN-Index series aligned to the account's own fill
// timestamps, so the design's benchmark comparison line is deliberately
// dropped here rather than faked -- a documented simplification, not a
// missing feature nobody decided about.
export default function EquityCurveChart({ points, startingCapital }: Props) {
  const [range, setRange] = useState<number | null>(null);

  const filtered = useMemo(() => {
    if (range === null) return points;
    const cutoff = nowMillis() - range * 24 * 60 * 60 * 1000;
    const kept = points.filter((p) => new Date(p.timestamp).getTime() >= cutoff);
    // Always keep at least the last point before the cutoff too, so a
    // short-range filter on a sparse (real, not-daily) series doesn't
    // render an empty chart just because the last fill happened outside
    // the window.
    if (kept.length === 0 && points.length > 0) return [points[points.length - 1]];
    return kept;
  }, [points, range]);

  const pct = useMemo(
    () => filtered.map((p) => (startingCapital > 0 ? ((p.nav - startingCapital) / startingCapital) * 100 : 0)),
    [filtered, startingCapital],
  );

  const latestPct = pct.length > 0 ? pct[pct.length - 1] : 0;

  if (filtered.length === 0) {
    return (
      <section className="flex shrink-0 flex-col gap-3 lg:h-[306px] rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
        <h2 className="m-0 text-[15px] font-semibold">Đường giá trị tài khoản</h2>
        <p className="text-[12.5px] text-app-text-muted">Chưa có dữ liệu -- đặt lệnh đầu tiên để bắt đầu theo dõi.</p>
      </section>
    );
  }

  const hi = Math.max(...pct, 0) + 1.5;
  const lo = Math.min(...pct, 0) - 1.5;
  const ex = (i: number) => (filtered.length > 1 ? (i / (filtered.length - 1)) * W : W / 2);
  const ey = (v: number) => PAD + ((hi - v) / (hi - lo || 1)) * (H - PAD * 2);

  const line = pct.map((v, i) => `${ex(i).toFixed(1)},${ey(v).toFixed(1)}`).join(" ");
  const area =
    `M0,${ey(pct[0]).toFixed(1)} ` +
    pct.map((v, i) => `L${ex(i).toFixed(1)},${ey(v).toFixed(1)}`).join(" ") +
    ` L${W},${(H - PAD).toFixed(1)} L0,${(H - PAD).toFixed(1)} Z`;

  const gridLines = [0, 1, 2, 3].map((i) => {
    const v = lo + ((hi - lo) * i) / 3;
    const y = ey(v);
    return { y, label: signVN(v, 1) + "%" };
  });

  const color = latestPct >= 0 ? "#35C77F" : "#FF5C5C";
  // Phone copy in Mobile-Portfolio.dc.html's 322-wide space, no axis
  // labels (phase-j.md decision 9); the zero line marks the starting capital.
  const PW = 322;
  const PH = 96;
  // A single point (a new account) is drawn as a flat line across.
  const series = pct.length === 1 ? [pct[0], pct[0]] : pct;
  const px = (i: number) => (i / (series.length - 1)) * PW;
  const pyp = (v: number) => 6 + ((hi - v) / (hi - lo || 1)) * (PH - 12);
  const phoneLine = series.map((v, i) => `${px(i).toFixed(1)},${pyp(v).toFixed(1)}`).join(" ");
  const phoneArea = `M0,${pyp(series[0]).toFixed(1)} ${series.map((v, i) => `L${px(i).toFixed(1)},${pyp(v).toFixed(1)}`).join(" ")} L${PW},${PH} L0,${PH} Z`;
  const first = new Date(filtered[0].timestamp);
  const last = new Date(filtered[filtered.length - 1].timestamp);

  return (
    <section className="flex shrink-0 flex-col gap-3 lg:h-[306px] rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-col gap-[3px]">
          <h2 className="m-0 text-[15px] font-semibold">Đường giá trị tài khoản</h2>
          <span className="text-[11.5px] text-app-text-muted">So với vốn ban đầu · dữ liệu thực từ lịch sử khớp lệnh</span>
        </div>
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-[6px] font-plex-mono text-[11px]" style={{ color }}>
            <span className="h-[2.5px] w-[14px] rounded" style={{ background: color }} />
            Tài khoản {signVN(latestPct, 2)}%
          </span>
          <div className="flex gap-[5px]">
            {RANGES.map((r) => (
              <button
                key={r.label}
                type="button"
                onClick={() => setRange(r.days)}
                className="h-9 whitespace-nowrap rounded-lg border px-[11px] text-[11.5px] lg:h-[30px]"
                style={
                  range === r.days
                    ? { borderColor: "var(--app-border-strong)", background: "var(--app-border)", color: "var(--app-text)" }
                    : { borderColor: "var(--app-border)", background: "var(--app-surface)", color: "var(--app-text-muted)" }
                }
              >
                {r.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <svg viewBox={`0 0 ${PW} ${PH}`} width="100%" height={PH} preserveAspectRatio="none" fill="none" aria-hidden="true" className="lg:hidden">
        <line x1="0" y1={pyp(0).toFixed(1)} x2={PW} y2={pyp(0).toFixed(1)} stroke="var(--app-hairline)" strokeWidth={1} strokeDasharray="4 3" />
        <path d={phoneArea} fill={color} opacity="0.1" />
        <polyline points={phoneLine} stroke={color} strokeWidth="1.9" strokeLinejoin="round" />
      </svg>
      <svg viewBox={`0 0 1000 ${H}`} width="100%" height={H} fill="none" className="hidden lg:block" role="img" aria-label={`Đường giá trị tài khoản ${signVN(latestPct, 2)}% so với vốn ban đầu`}>
        {gridLines.map((g, i) => (
          <g key={i}>
            <line x1="0" y1={g.y} x2="944" y2={g.y} stroke="var(--app-hairline)" strokeWidth={1} />
            <text x="1000" y={g.y + 3.5} textAnchor="end" fill="var(--app-text-muted)" fontFamily="'IBM Plex Mono', monospace" fontSize="10.5">
              {g.label}
            </text>
          </g>
        ))}
        <path d={area} fill={color} opacity="0.1" />
        <polyline points={line} stroke={color} strokeWidth="2" strokeLinejoin="round" />
      </svg>

      <div className="flex justify-between font-plex-mono text-[10.5px] text-app-text-muted">
        <span>{first.toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", timeZone: "UTC" })}</span>
        <span>{last.toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", timeZone: "UTC" })}</span>
      </div>
    </section>
  );
}
