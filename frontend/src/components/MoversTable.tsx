import type { TickerChange } from "@/lib/api";
import { formatVN, formatVolumeVN, signVN, tone } from "@/lib/format";

export default function MoversTable({ title, rows }: { title: string; rows: TickerChange[] }) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-[10px]">
      <h2 className="m-0 text-[12.5px] font-semibold text-app-text-2">{title}</h2>
      <table className="w-full text-[12.5px]">
        <thead>
          <tr className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">
            <th scope="col" className="p-0 pb-[7px] text-left font-medium">Mã</th>
            <th scope="col" className="p-0 pb-[7px] text-right font-medium">Giá</th>
            <th scope="col" className="p-0 pb-[7px] text-right font-medium">+/−</th>
            <th scope="col" className="p-0 pb-[7px] text-right font-medium">KL</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.symbol} className="border-t border-app-hairline">
              <td className="py-[6.5px] font-plex-mono font-semibold">{r.symbol}</td>
              <td className="py-[6.5px] text-right font-plex-mono">{r.price !== undefined ? formatVN(r.price, 2) : "—"}</td>
              <td className="py-[6.5px] text-right font-plex-mono font-semibold" style={{ color: tone(r.changePercent) }}>
                {signVN(r.changePercent, 2)}%
              </td>
              <td className="py-[6.5px] text-right font-plex-mono text-app-text-muted">
                {r.volume !== undefined ? formatVolumeVN(r.volume) : "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
