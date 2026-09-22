import type { PortfolioSummary } from "@/lib/api";
import { formatVN, signVN, tone } from "@/lib/format";

type Props = {
  summary: PortfolioSummary;
  startingCapital: number;
  sinceDate: string; // already formatted, e.g. "01/01/2026"
};

export default function PaperAccountCard({ summary, startingCapital, sinceDate }: Props) {
  const returnAbs = summary.totalEquity - startingCapital;
  const returnPct = startingCapital > 0 ? (returnAbs / startingCapital) * 100 : 0;
  const stockWeightPct = summary.totalEquity > 0 ? (summary.marketValue / summary.totalEquity) * 100 : 0;
  const color = tone(returnPct);

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Tài khoản giấy</span>
        <span className="text-[11px] text-app-text-muted">Từ {sinceDate}</span>
      </div>
      <div className="flex items-end gap-[10px]">
        <span className="font-plex-mono text-[27px] font-semibold tracking-[-0.015em]">{formatVN(summary.totalEquity, 0)} ₫</span>
      </div>
      <div className="flex items-center gap-2">
        <span
          className="rounded-md px-2 py-[3px] font-plex-mono text-xs font-semibold"
          style={{ color, background: returnPct >= 0 ? "rgba(53, 199, 127, 0.16)" : "rgba(255, 92, 92, 0.16)" }}
        >
          {signVN(returnPct, 2)}%
        </span>
        <span className="text-xs text-app-text-3">{signVN(returnAbs / 1_000_000, 2)} tr so với vốn ban đầu</span>
      </div>
      <div className="grid grid-cols-3 gap-[10px] border-t border-app-hairline pt-3">
        <div className="flex flex-col gap-[3px]">
          <span className="text-[10.5px] text-app-text-muted">Tiền mặt</span>
          <span className="font-plex-mono text-[13.5px] font-semibold">{formatVN(summary.cashBalance / 1_000_000, 1)} tr</span>
        </div>
        <div className="flex flex-col gap-[3px]">
          <span className="text-[10.5px] text-app-text-muted">Tỷ trọng cổ phiếu</span>
          <span className="font-plex-mono text-[13.5px] font-semibold">{formatVN(stockWeightPct, 1)}%</span>
        </div>
        <div className="flex flex-col gap-[3px]">
          <span className="text-[10.5px] text-app-text-muted">Lãi/lỗ mở</span>
          <span className="font-plex-mono text-[13.5px] font-semibold" style={{ color: tone(summary.unrealizedPnl) }}>
            {signVN(summary.unrealizedPnl / 1_000_000, 1)} tr
          </span>
        </div>
      </div>
    </section>
  );
}
