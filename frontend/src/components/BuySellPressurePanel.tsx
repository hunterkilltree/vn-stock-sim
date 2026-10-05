import type { Bar } from "@/lib/api";
import { formatVN, formatVolumeVN } from "@/lib/format";

// "Lực mua / bán": buy vs sell volume for the latest session and the last
// 20, as a split bar. The split is an estimate from each bar's close
// location (backend volumesplit.go), not tick-level aggressor data, and
// the panel says so.
function sum(bars: Bar[]) {
  let buy = 0;
  let sell = 0;
  for (const b of bars) {
    buy += b.buyVolume ?? 0;
    sell += b.sellVolume ?? 0;
  }
  return { buy, sell };
}

function Row({ label, buy, sell }: { label: string; buy: number; sell: number }) {
  const total = buy + sell;
  const buyPct = total > 0 ? (buy / total) * 100 : 50;
  return (
    <div className="flex flex-col gap-[6px]">
      <div className="flex items-center justify-between text-[11px] text-app-text-muted">
        <span>{label}</span>
        <span className="font-plex-mono">{formatVN(buyPct, 1)}% mua</span>
      </div>
      <div className="flex h-[8px] overflow-hidden rounded-full bg-app-hairline" role="img" aria-label={`${label}: ${formatVN(buyPct, 1)}% khối lượng mua`}>
        <div style={{ width: `${buyPct}%`, background: "#35C77F" }} />
        <div style={{ width: `${100 - buyPct}%`, background: "#FF5C5C" }} />
      </div>
      <div className="flex justify-between font-plex-mono text-xs">
        <span className="text-price-up">Mua {formatVolumeVN(buy)}</span>
        <span className="text-price-down">Bán {formatVolumeVN(sell)}</span>
      </div>
    </div>
  );
}

export default function BuySellPressurePanel({ bars }: { bars: Bar[] }) {
  const withSplit = bars.filter((b) => b.buyVolume !== undefined && b.sellVolume !== undefined);
  if (withSplit.length === 0) return null;
  const last = sum(withSplit.slice(-1));
  const recent = sum(withSplit.slice(-20));

  return (
    <section className="flex flex-col gap-[12px] rounded-2xl border border-app-border bg-app-surface p-[16px_18px]">
      <h2 className="m-0 text-[13px] font-semibold">Lực mua / bán</h2>
      <Row label="Phiên gần nhất" buy={last.buy} sell={last.sell} />
      <Row label="20 phiên" buy={recent.buy} sell={recent.sell} />
      <p className="m-0 text-[10px] text-app-text-muted">Ước tính từ vị trí giá đóng cửa trong biên độ phiên.</p>
    </section>
  );
}
