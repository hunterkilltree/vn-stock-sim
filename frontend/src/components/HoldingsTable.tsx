import Link from "next/link";
import type { Position } from "@/lib/api";
import { formatVN, signVN, tone } from "@/lib/format";

function daysHeld(openSince?: string): string {
  if (!openSince) return "—";
  const days = Math.max(0, Math.floor((Date.now() - new Date(openSince).getTime()) / (24 * 60 * 60 * 1000)));
  return `${days} ngày`;
}

// Portfolio.dc.html's "Vị thế đang mở" table, the fuller Positions-tab
// version of OpenPositionsCard.tsx (which only shows 4 columns for the
// Main screen's sidebar-width card) -- this adds Giá hiện tại/Giá trị/
// %NAV/Nắm giữ to match the design's 8-column table. Company names
// (design shows "FPT  CTCP FPT") aren't fetched per-symbol here to avoid
// N extra requests -- deliberately deferred, same simplification class
// as MoversTable's symbol-only rows.
export default function HoldingsTable({ positions, totalEquity }: { positions: Position[]; totalEquity: number }) {
  return (
    <section className="flex min-h-0 flex-1 flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Vị thế đang mở</h2>
        <span className="text-[11.5px] text-app-text-muted">{positions.length} mã</span>
      </div>
      {positions.length === 0 ? (
        <p className="text-[12.5px] text-app-text-muted">Chưa có vị thế nào đang mở.</p>
      ) : (
        <>
          {/* Phones: Mobile-Portfolio.dc.html's holding cards (phase-j.md decision 9). */}
          <div className="flex flex-col gap-2 lg:hidden">
            {positions.map((p) => {
              const cost = p.avgCost * p.quantity;
              const pnlPct = cost > 0 ? (p.unrealizedPnl / cost) * 100 : 0;
              return (
                <Link
                  key={p.symbol}
                  href={`/stocks/${p.symbol}`}
                  className="flex min-h-14 items-center gap-3 rounded-[13px] border border-app-hairline bg-app-surface-2 p-[10px_14px] text-app-text"
                >
                  <span className="flex min-w-0 flex-1 flex-col gap-[3px]">
                    <span className="font-plex-mono text-[14px] font-semibold">{p.symbol}</span>
                    <span className="truncate text-[11px] text-app-text-muted">
                      {p.quantity.toLocaleString("vi-VN")} cp · vốn {formatVN(p.avgCost / 1000, 2)}
                    </span>
                  </span>
                  <span className="flex flex-col items-end gap-[3px]">
                    <span className="font-plex-mono text-[13.5px]">{formatVN(p.marketValue / 1_000_000, 1)} tr</span>
                    <span className="font-plex-mono text-[12px] font-semibold" style={{ color: tone(pnlPct) }}>
                      {signVN(pnlPct, 2)}%
                    </span>
                  </span>
                </Link>
              );
            })}
          </div>
          <table className="hidden w-full text-[12.5px] lg:table">
            <thead>
              <tr className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">
                <th scope="col" className="p-0 pb-[9px] text-left font-medium">Mã</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">Khối lượng</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">Giá vốn</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">Giá hiện tại</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">Giá trị</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">Lãi/lỗ</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">% NAV</th>
                <th scope="col" className="p-0 pb-[9px] text-right font-medium">Nắm giữ</th>
              </tr>
            </thead>
            <tbody>
              {positions.map((p) => {
                const cost = p.avgCost * p.quantity;
                const pnlPct = cost > 0 ? (p.unrealizedPnl / cost) * 100 : 0;
                const navPct = totalEquity > 0 ? (p.marketValue / totalEquity) * 100 : 0;
                return (
                  <tr key={p.symbol} className="border-t border-app-hairline">
                    <td className="py-[9.5px] font-plex-mono font-semibold">{p.symbol}</td>
                    <td className="py-[9.5px] text-right font-plex-mono text-app-text-3">{p.quantity.toLocaleString("vi-VN")}</td>
                    <td className="py-[9.5px] text-right font-plex-mono text-app-text-3">{formatVN(p.avgCost, 2)}</td>
                    <td className="py-[9.5px] text-right font-plex-mono">{formatVN(p.lastPrice, 2)}</td>
                    <td className="py-[9.5px] text-right font-plex-mono">{formatVN(p.marketValue / 1_000_000, 1)} tr</td>
                    <td className="py-[9.5px] text-right font-plex-mono font-semibold" style={{ color: tone(pnlPct) }}>
                      {signVN(pnlPct, 2)}%
                    </td>
                    <td className="py-[9.5px] text-right font-plex-mono text-app-text-3">{formatVN(navPct, 1)}%</td>
                    <td className="py-[9.5px] text-right text-app-text-muted">{daysHeld(p.openSince)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </>
      )}
    </section>
  );
}
