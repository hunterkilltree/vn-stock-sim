import type { Allocation } from "@/lib/api";
import { formatVN, sectorColor } from "@/lib/format";

// Portfolio.dc.html's "Phân bổ theo ngành" panel, backed by real
// GET /portfolios/:id/allocation (phase-e.md item 3) instead of the
// design's hardcoded allocRaw sample.
export default function SectorAllocationCard({ allocation }: { allocation: Allocation[] }) {
  const maxPercent = Math.max(...allocation.map((a) => a.percent), 20);

  return (
    <section className="flex flex-col gap-[13px] rounded-2xl border border-app-border bg-app-surface p-[18px]">
      <h2 className="m-0 text-[15px] font-semibold">Phân bổ theo ngành</h2>
      {allocation.length === 0 ? (
        <p className="text-[12.5px] text-app-text-muted">Chưa có vị thế nào để phân bổ.</p>
      ) : (
        <div className="flex flex-col gap-[11px]">
          {allocation.map((a, i) => (
            <div key={a.sector} className="flex flex-col gap-[5px]">
              <div className="flex justify-between text-[12px]">
                <span className="text-app-text-2">{a.sector}</span>
                <span className="font-plex-mono text-app-text-3">{formatVN(a.percent, 1)}%</span>
              </div>
              <div className="h-[6px] rounded-[3px] bg-app-surface-3">
                <div
                  className="h-[6px] rounded-[3px]"
                  style={{ width: `${Math.min((a.percent / maxPercent) * 100, 100)}%`, background: sectorColor(a.sector, i) }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
