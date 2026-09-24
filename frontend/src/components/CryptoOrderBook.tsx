import type { DepthLevel } from "@/lib/api";
import { formatAmount, formatCryptoPrice } from "@/lib/format";

type Props = { bids: DepthLevel[]; asks: DepthLevel[]; base: string; source: string };

// Crypto-Detail.dc.html's 8-level book: 4 asks (highest first) over 4
// bids, each with its size and a depth bar sized by the running total.
export default function CryptoOrderBook({ bids, asks, base, source }: Props) {
  const maxCum = Math.max(...bids.map((b) => b.cumulative), ...asks.map((a) => a.cumulative), 1e-12);
  const rows = [
    ...[...asks].reverse().map((l) => ({ ...l, side: "ask" as const })),
    ...bids.map((l) => ({ ...l, side: "bid" as const })),
  ];
  return (
    <section className="flex flex-col gap-2 rounded-2xl border border-app-border bg-app-surface p-4">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[15px] font-semibold">Sổ lệnh</h2>
        <span className="text-[11px] text-app-text-muted">{source === "binance" ? "Binance · 8 mức gần nhất" : "Dữ liệu mô phỏng · 8 mức"}</span>
      </div>
      <table className="w-full border-collapse text-[12px]">
        <thead>
          <tr className="text-[10.5px] text-app-text-muted">
            <th scope="col" className="py-1 text-left font-medium">Giá (USDT)</th>
            <th scope="col" className="py-1 text-right font-medium">Lượng ({base})</th>
            <th scope="col" className="py-1 text-right font-medium">Cộng dồn</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={`${r.side}-${r.price}`}>
              <td className="py-[5px] font-plex-mono" style={{ color: r.side === "ask" ? "#FF5C5C" : "#35C77F" }}>
                {formatCryptoPrice(r.price)}
              </td>
              <td className="py-[5px] text-right font-plex-mono text-app-text-2">{formatAmount(r.size, 4)}</td>
              <td className="relative py-[5px] text-right font-plex-mono text-app-text-3">
                <span
                  className="absolute inset-y-[3px] right-0 rounded-[3px]"
                  style={{ width: `${(r.cumulative / maxCum) * 100}%`, background: r.side === "ask" ? "rgba(255,92,92,0.12)" : "rgba(53,199,127,0.12)" }}
                  aria-hidden="true"
                />
                <span className="relative">{formatAmount(r.cumulative, 4)}</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
