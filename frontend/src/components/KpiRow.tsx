import type { PortfolioStats } from "@/lib/api";
import { formatVN, signVN, tone } from "@/lib/format";

// Portfolio.dc.html's 5-up KPI row, computed from real Stats (phase-e.md
// item 2) instead of the design's hardcoded kpiRaw sample.
export default function KpiRow({ stats }: { stats: PortfolioStats }) {
  const kpis: { k: string; v: string; color: string; note: string }[] = [
    { k: "Giá trị tài khoản", v: `${formatVN(stats.totalEquity, 0)} ₫`, color: "var(--app-text)", note: "Xem chi tiết ở tab Vị thế" },
    {
      k: "Lãi/lỗ tổng",
      v: `${signVN(stats.totalPnl / 1_000_000, 1)} tr ₫`,
      color: tone(stats.totalPnl),
      note: `${signVN(stats.totalPnlPercent, 2)}% từ khi mở tài khoản`,
    },
    {
      k: "Tỷ lệ thắng",
      v: stats.closedTradeCount > 0 ? `${formatVN(stats.winRate, 1)}%` : "—",
      color: "var(--app-text)",
      note: `${stats.wins} thắng / ${stats.losses} thua`,
    },
    {
      k: "Hệ số lợi nhuận",
      v: stats.profitFactor > 0 ? formatVN(stats.profitFactor, 2) : "—",
      color: "var(--app-text)",
      note: `Lãi TB ${signVN(stats.avgWinPercent, 1)}% · lỗ TB ${signVN(stats.avgLossPercent, 1)}%`,
    },
    {
      k: "Sụt giảm tối đa",
      v: stats.maxDrawdownPercent > 0 ? `−${formatVN(stats.maxDrawdownPercent, 2)}%` : "0%",
      color: stats.maxDrawdownPercent > 0 ? "#FF5C5C" : "var(--app-text)",
      note: `Trên ${stats.closedTradeCount} lệnh đã đóng`,
    },
  ];

  return (
    // Phones (Mobile-Portfolio.dc.html): account value full width, then
    // the other four as a 2 x 2 grid; five across from lg up.
    <div className="grid grid-cols-2 gap-[9px] lg:grid-cols-5 lg:gap-[14px]">
      {kpis.map((k, i) => (
        <section
          key={k.k}
          className={`flex min-w-0 flex-col gap-2 rounded-[14px] border border-app-border bg-app-surface p-[12px_14px] lg:p-[15px_17px] ${i === 0 ? "col-span-2 lg:col-span-1" : ""}`}
        >
          <span className="text-[10px] uppercase tracking-[0.06em] text-app-text-muted lg:text-[10.5px] lg:tracking-[0.08em]">{k.k}</span>
          <span
            className={`font-plex-mono font-semibold tracking-[-0.01em] ${i === 0 ? "text-[27px] lg:text-[19px]" : "text-[16px] lg:text-[19px]"}`}
            style={{ color: k.color }}
          >
            {k.v}
          </span>
          <span className="text-[11px] text-app-text-muted lg:text-[11.5px]">{k.note}</span>
        </section>
      ))}
    </div>
  );
}
