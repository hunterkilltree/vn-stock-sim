import Link from "next/link";
import type { CryptoPairQuote } from "@/lib/api";
import { formatCompact, formatCryptoPrice, signVN, tone } from "@/lib/format";

export default function CryptoMoversTable({ title, rows }: { title: string; rows: CryptoPairQuote[] }) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-2">
      <h2 className="m-0 text-[14px] font-semibold">{title}</h2>
      <table className="w-full border-collapse text-[12.5px]">
        <thead>
          <tr className="border-b border-app-hairline text-[10.5px] uppercase tracking-[0.06em] text-app-text-muted">
            <th scope="col" className="py-[6px] text-left font-medium">Cặp</th>
            <th scope="col" className="py-[6px] text-right font-medium">Giá</th>
            <th scope="col" className="py-[6px] text-right font-medium">24h</th>
            <th scope="col" className="py-[6px] text-right font-medium">KL 24h</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.symbol}>
              <td className="py-[6.5px]">
                <Link href={`/crypto/${r.symbol}`} className="font-plex-mono font-semibold text-app-text">
                  {r.base}/USDT
                </Link>
              </td>
              <td className="py-[6.5px] text-right font-plex-mono">{formatCryptoPrice(r.lastPrice)}</td>
              <td className="py-[6.5px] text-right font-plex-mono font-semibold" style={{ color: tone(r.changePercent) }}>
                {signVN(r.changePercent, 1)}%
              </td>
              <td className="py-[6.5px] text-right font-plex-mono text-app-text-3">{formatCompact(r.quoteVolume24h)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
