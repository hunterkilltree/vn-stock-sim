import type { SymbolDetail } from "@/lib/api";
import { formatMarketCapVN, formatVN, formatVolumeVN } from "@/lib/format";

// "Chi so co ban" grid from design/screens/Detail.dc.html. avgVolume20d
// is computed by the page from the already-fetched Bar[] (last 20 daily
// bars' volume, averaged) -- not a new backend field, see phase-d.md
// decision 3.
export default function FundamentalsGrid({ detail, avgVolume20d }: { detail: SymbolDetail; avgVolume20d: number | null }) {
  const rows: { k: string; v: string }[] = [
    { k: "Vốn hóa", v: formatMarketCapVN(detail.marketCap) },
    { k: "P/E", v: formatVN(detail.peRatio, 1) },
    { k: "P/B", v: formatVN(detail.pbRatio, 1) },
    { k: "ROE", v: `${formatVN(detail.roe, 1)}%` },
    { k: "EPS 4 quý", v: `${formatVN(detail.eps, 0)} ₫` },
    { k: "KL TB 20 phiên", v: avgVolume20d !== null ? formatVolumeVN(avgVolume20d) : "—" },
  ];

  return (
    <section className="flex flex-1 flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[16px_18px]">
      <h2 className="m-0 text-[13px] font-semibold">Chỉ số cơ bản</h2>
      <div className="grid grid-cols-2 gap-x-[10px] gap-y-3">
        {rows.map((r) => (
          <div key={r.k} className="flex flex-col gap-[3px]">
            <span className="text-[10.5px] text-app-text-muted">{r.k}</span>
            <span className="font-plex-mono text-[13.5px] font-semibold">{r.v}</span>
          </div>
        ))}
      </div>
    </section>
  );
}
