import type { PriceLevel } from "@/lib/api";
import { formatThousandsVN } from "@/lib/format";

type Props = { bids: PriceLevel[]; asks: PriceLevel[] };

// "Buoc gia" panel from design/screens/Detail.dc.html -- 3 levels each
// side (not Phase B's full 6), a single descending price ladder: the 3
// closest asks on top (highest of the three first), then the 3 closest
// bids below, with the row nearest the last trade price highlighted
// amber (phase-d.md decision 7).
export default function OrderBookPanel({ bids, asks }: Props) {
  const topAsks = asks.slice(0, 3).reverse();
  const topBids = bids.slice(0, 3);
  const rows = [
    ...topAsks.map((a) => ({ bid: "", ask: a.volume, price: a.price, mid: false })),
    ...topBids.map((b, i) => ({ bid: b.volume, ask: "", price: b.price, mid: i === 0 })),
  ];

  return (
    <section className="flex flex-col gap-[10px] rounded-2xl border border-app-border bg-app-surface p-[16px_18px]">
      <h2 className="m-0 text-[13px] font-semibold">Bước giá</h2>
      <table className="w-full font-plex-mono text-xs">
        <thead>
          <tr className="font-sans text-[10px] uppercase tracking-[0.07em] text-app-text-muted">
            <th scope="col" className="p-0 pb-[6px] text-left font-medium">KL mua</th>
            <th scope="col" className="p-0 pb-[6px] text-center font-medium">Giá</th>
            <th scope="col" className="p-0 pb-[6px] text-right font-medium">KL bán</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.price} className="border-t border-app-hairline">
              <td className="py-[6px] text-price-up">{r.bid !== "" ? r.bid.toLocaleString("vi-VN") : ""}</td>
              <td className="py-[6px] text-center font-semibold" style={{ color: r.mid ? "#F0C243" : "var(--app-text-2)" }}>
                {formatThousandsVN(r.price)}
              </td>
              <td className="py-[6px] text-right text-price-down">{r.ask !== "" ? r.ask.toLocaleString("vi-VN") : ""}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
