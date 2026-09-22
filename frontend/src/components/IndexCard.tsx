import type { IndexSnapshot } from "@/lib/api";
import { formatVN, signVN, tone } from "@/lib/format";

// One index card from design/screens/Main.dc.html's 4-up grid. The
// design generates a fake 30-point sparkline client-side; this renders
// Phase B's real 20-point IndexSnapshot.sparkline instead (phase-c.md
// decision 5) -- same viewBox math (112x36), normalized against the
// real series' own min/max rather than the design's fixed [0.08,0.92]
// synthetic range.
function sparklinePoints(values: number[]): string {
  if (values.length === 0) return "";
  const w = 112;
  const h = 36;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  return values
    .map((v, i) => {
      const x = (i / Math.max(values.length - 1, 1)) * w;
      const norm = (v - min) / span;
      const y = (1 - norm) * (h - 4) + 2;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
}

export default function IndexCard({ index }: { index: IndexSnapshot }) {
  const color = tone(index.changePercent);
  const up = index.changePercent >= 0;

  return (
    <section className="flex flex-col gap-[10px] rounded-2xl border border-app-border bg-app-surface p-[16px_18px]">
      <div className="flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">{index.name}</span>
        <span
          className="rounded-md px-2 py-[3px] font-plex-mono text-[11.5px] font-semibold"
          style={{ color, background: up ? "rgba(53, 199, 127, 0.14)" : "rgba(255, 92, 92, 0.14)" }}
        >
          {signVN(index.changePercent, 2)}%
        </span>
      </div>
      <div className="flex items-end justify-between gap-3">
        <div className="flex flex-col gap-[2px]">
          <span className="font-plex-mono text-[25px] font-semibold tracking-[-0.01em]">{formatVN(index.value, 2)}</span>
          <span className="font-plex-mono text-[12.5px]" style={{ color }}>
            {signVN(index.change, 2)}
          </span>
        </div>
        <svg width="112" height="36" viewBox="0 0 112 36" fill="none" aria-hidden="true">
          <polyline points={sparklinePoints(index.sparkline)} stroke={color} strokeWidth="1.8" strokeLinejoin="round" fill="none" />
        </svg>
      </div>
      <div className="border-t border-app-hairline pt-[10px] text-[11.5px] text-app-text-muted">
        {index.sparkline.length} phiên gần nhất
      </div>
    </section>
  );
}
