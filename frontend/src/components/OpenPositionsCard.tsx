import type { Position } from "@/lib/api";
import { formatVN, signVN, tone } from "@/lib/format";

export default function OpenPositionsCard({ positions }: { positions: Position[] }) {
  return (
    <section className="flex min-h-0 flex-1 flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Vị thế đang mở</h2>
      </div>
      {positions.length === 0 ? (
        <p className="text-[12.5px] text-app-text-muted">Chưa có vị thế nào đang mở.</p>
      ) : (
        <table className="w-full text-[12.5px]">
          <thead>
            <tr className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">
              <th scope="col" className="p-0 pb-[7px] text-left font-medium">Mã</th>
              <th scope="col" className="p-0 pb-[7px] text-right font-medium">KL</th>
              <th scope="col" className="p-0 pb-[7px] text-right font-medium">Giá vốn</th>
              <th scope="col" className="p-0 pb-[7px] text-right font-medium">Lãi/lỗ</th>
            </tr>
          </thead>
          <tbody>
            {positions.map((p) => {
              const cost = p.avgCost * p.quantity;
              const pnlPct = cost > 0 ? (p.unrealizedPnl / cost) * 100 : 0;
              return (
                <tr key={p.symbol} className="border-t border-app-hairline">
                  <td className="py-2 font-plex-mono font-semibold">{p.symbol}</td>
                  <td className="py-2 text-right font-plex-mono text-app-text-3">{p.quantity.toLocaleString("vi-VN")}</td>
                  <td className="py-2 text-right font-plex-mono text-app-text-3">{formatVN(p.avgCost, 2)}</td>
                  <td className="py-2 text-right font-plex-mono font-semibold" style={{ color: tone(pnlPct) }}>
                    {signVN(pnlPct, 2)}% · {signVN(p.unrealizedPnl / 1_000_000, 1)} tr
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </section>
  );
}
