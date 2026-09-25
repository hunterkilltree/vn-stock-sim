import type { BacktestRun } from "@/lib/api";
import { formatVN, signVN } from "@/lib/format";

const W = 1000;
const H = 220;

// Strategy equity vs buy-and-hold of the same capital (phase-k.md
// decision 14). The lines stretch to any width (preserveAspectRatio
// none, non-scaling strokes); the labels are HTML, so they stay legible
// on a phone instead of shrinking with the drawing.
export default function BacktestEquityChart({ run }: { run: BacktestRun }) {
  const pts = run.equity ?? [];
  if (pts.length < 2) {
    return <p className="m-0 text-[12.5px] text-app-text-muted">Không đủ dữ liệu để vẽ đường vốn.</p>;
  }
  // At most ~400 points: plenty for the width, light for the DOM.
  const stride = Math.max(1, Math.ceil(pts.length / 400));
  const sample = pts.filter((_, i) => i % stride === 0 || i === pts.length - 1);
  const values = sample.flatMap((p) => [p.equity, p.benchmark]);
  const hi = Math.max(...values);
  const lo = Math.min(...values);
  const x = (i: number) => (i / (sample.length - 1)) * W;
  const y = (v: number) => 6 + ((hi - v) / (hi - lo || 1)) * (H - 12);
  const line = (key: "equity" | "benchmark") => sample.map((p, i) => `${x(i).toFixed(1)},${y(p[key]).toFixed(1)}`).join(" ");
  const up = run.returnPercent >= 0;
  const color = up ? "#35C77F" : "#FF5C5C";
  const date = (t: number) => new Date(t * 1000).toLocaleDateString("vi-VN", { month: "2-digit", year: "numeric", timeZone: "UTC" });
  const money = (v: number) => `${formatVN(v / 1_000_000, v >= 1e9 ? 0 : 1)} tr`;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11.5px]">
        <span className="flex items-center gap-[6px] font-plex-mono" style={{ color }}>
          <span className="h-[2.5px] w-[14px] rounded" style={{ background: color }} />
          Chiến lược {signVN(run.returnPercent, 2)}%
        </span>
        <span className="flex items-center gap-[6px] font-plex-mono text-app-text-3">
          <span className="h-[2.5px] w-[14px] rounded bg-[#6F6C63]" />
          Mua và giữ {signVN(run.benchmarkReturnPercent, 2)}%
        </span>
      </div>
      <div className="flex gap-2">
        <div className="flex flex-col justify-between py-[2px] text-right font-plex-mono text-[10.5px] text-app-text-muted" style={{ height: H }}>
          <span>{money(hi)}</span>
          <span>{money(lo)}</span>
        </div>
        <svg
          viewBox={`0 0 ${W} ${H}`}
          preserveAspectRatio="none"
          width="100%"
          height={H}
          fill="none"
          role="img"
          aria-label={`Đường vốn chiến lược ${signVN(run.returnPercent, 2)}% so với mua và giữ ${signVN(run.benchmarkReturnPercent, 2)}%`}
          className="min-w-0 flex-1"
        >
          <line x1="0" y1={y(run.startingCapital)} x2={W} y2={y(run.startingCapital)} stroke="#2C2C28" strokeWidth="1" strokeDasharray="4 3" vectorEffect="non-scaling-stroke" />
          <polyline points={line("benchmark")} stroke="#6F6C63" strokeWidth="1.4" strokeDasharray="4 3" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
          <polyline points={line("equity")} stroke={color} strokeWidth="2" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
        </svg>
      </div>
      <div className="flex justify-between pl-12 font-plex-mono text-[10.5px] text-app-text-muted">
        <span>{date(sample[0].time)}</span>
        <span>{date(sample[sample.length - 1].time)}</span>
      </div>
    </div>
  );
}
