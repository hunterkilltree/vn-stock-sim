import type { SectorGroup } from "@/lib/api";
import { signVN, tone } from "@/lib/format";

// tile() color-intensity formula, ported verbatim from
// design/screens/Main.dc.html's script block (phase-c.md decision 6).
function tileStyle(pct: number): { background: string; borderColor: string } {
  const alpha = (0.12 + Math.min(Math.abs(pct), 5) / 5 * 0.33).toFixed(2);
  const background =
    pct > 0 ? `rgba(53, 199, 127, ${alpha})` : pct < 0 ? `rgba(255, 92, 92, ${alpha})` : "rgba(240, 194, 67, 0.16)";
  const borderColor =
    pct > 0 ? "rgba(53, 199, 127, 0.4)" : pct < 0 ? "rgba(255, 92, 92, 0.4)" : "rgba(240, 194, 67, 0.45)";
  return { background, borderColor };
}

// Legend includes Trần/Sàn (ceiling/floor) per the design, but this V1
// heatmap only ever renders the continuous up/down/reference gradient
// (tileStyle above) -- detecting a real ceiling/floor-limit tile would
// need comparing each tile's live price against symbol.Detail's
// Ceiling/Floor bands, which the heatmap endpoint doesn't carry
// per-tile. Documented gap, not a silent omission -- see phase-c.md.
const LEGEND: { label: string; color: string }[] = [
  { label: "Trần", color: "#C08BFF" },
  { label: "Tăng", color: "#35C77F" },
  { label: "Tham chiếu", color: "#F0C243" },
  { label: "Giảm", color: "#FF5C5C" },
  { label: "Sàn", color: "#4FD3E8" },
];

export default function SectorHeatmap({ sectors }: { sectors: SectorGroup[] }) {
  return (
    <section className="flex min-h-0 flex-1 flex-col gap-[14px] rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Bản đồ nhiệt thị trường</h2>
        <div className="flex items-center gap-[14px] text-[11px] text-app-text-muted">
          {LEGEND.map((l) => (
            <span key={l.label} className="flex items-center gap-[6px]">
              <span className="h-[10px] w-[10px] rounded-[3px]" style={{ background: l.color }} />
              {l.label}
            </span>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-3">
        {sectors.map((sec) => (
          <div key={sec.sector} className="flex flex-col gap-[7px]">
            <div className="flex items-baseline gap-2">
              <span className="text-xs font-semibold text-app-text-2">{sec.sector}</span>
              <span className="font-plex-mono text-[11px]" style={{ color: tone(sec.avgChangePercent) }}>
                {signVN(sec.avgChangePercent, 2)}%
              </span>
            </div>
            <div className="grid grid-cols-6 gap-2">
              {sec.tickers.map((t) => {
                const s = tileStyle(t.changePercent);
                return (
                  <div
                    key={t.symbol}
                    className="box-border flex h-[46px] flex-col justify-between rounded-lg border p-[6px_9px]"
                    style={{ background: s.background, borderColor: s.borderColor }}
                  >
                    <span className="font-plex-mono text-[13px] font-semibold">{t.symbol}</span>
                    <span className="font-plex-mono text-[11.5px] text-app-text">{signVN(t.changePercent, 2)}%</span>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
