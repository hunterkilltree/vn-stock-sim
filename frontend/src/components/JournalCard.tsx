import type { PortfolioStats } from "@/lib/api";
import { formatVN, signVN } from "@/lib/format";

// Portfolio.dc.html's "Sổ giao dịch" journal grid, computed from real
// FIFO-reconstructed closed trades (backend/internal/portfolio's Stats
// -- see phase-e.md item 2) instead of the design's hardcoded jRaw
// sample. The design's "Từ Replay" cell is dropped -- Replay Mode
// doesn't exist yet (Phase F) -- and replaced with a real closed-trade
// count instead of leaving a feature-that-doesn't-exist-yet number in
// the grid.
export default function JournalCard({ stats }: { stats: PortfolioStats }) {
  const rows: { k: string; v: string; color?: string }[] = [
    { k: "Lãi trung bình", v: `${signVN(stats.avgWinPercent, 2)}%`, color: "#35C77F" },
    { k: "Lỗ trung bình", v: `${signVN(stats.avgLossPercent, 2)}%`, color: "#FF5C5C" },
    { k: "Thời gian giữ TB", v: `${formatVN(stats.avgHoldingDays, 1)} ngày` },
    {
      k: "Lệnh tốt nhất",
      v: stats.bestTrade ? `${stats.bestTrade.symbol} ${signVN(stats.bestTrade.pnlPercent, 1)}%` : "—",
      color: stats.bestTrade ? "#35C77F" : undefined,
    },
    {
      k: "Lệnh tệ nhất",
      v: stats.worstTrade ? `${stats.worstTrade.symbol} ${signVN(stats.worstTrade.pnlPercent, 1)}%` : "—",
      color: stats.worstTrade ? "#FF5C5C" : undefined,
    },
    { k: "Lệnh đã đóng", v: `${stats.closedTradeCount}` },
  ];

  return (
    <section className="flex flex-col gap-[13px] rounded-2xl border border-app-border bg-app-surface p-[18px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Sổ giao dịch</h2>
        <span className="text-[12px] text-app-text-muted">{stats.closedTradeCount} lệnh đã đóng</span>
      </div>
      <div className="grid grid-cols-2 gap-[13px_10px]">
        {rows.map((r) => (
          <div key={r.k} className="flex flex-col gap-[3px]">
            <span className="text-[10.5px] text-app-text-muted">{r.k}</span>
            <span className="font-plex-mono text-[14.5px] font-semibold" style={{ color: r.color ?? "var(--app-text)" }}>
              {r.v}
            </span>
          </div>
        ))}
      </div>
      <p className="m-0 border-t border-app-hairline pt-3 text-[12px] leading-[1.55] text-app-text-3">
        Mọi lệnh mô phỏng được ghi lại để đối chiếu về sau — dựa trên lịch sử khớp lệnh thực tế, không phải số liệu mẫu.
      </p>
    </section>
  );
}
