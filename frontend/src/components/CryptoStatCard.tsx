import { signVN, tone } from "@/lib/format";

// One of Crypto-Main.dc.html's four cards (BTC, ETH, market cap,
// dominance) -- same shape as the stock IndexCard.
function points(values: number[]): string {
  if (values.length < 2) return "";
  const min = Math.min(...values);
  const span = Math.max(...values) - min || 1;
  return values.map((v, i) => `${((i / (values.length - 1)) * 112).toFixed(1)},${((1 - (v - min) / span) * 32 + 2).toFixed(1)}`).join(" ");
}

type Props = { name: string; value: string; changePercent: number; changeText: string; foot: string; sparkline?: number[] };

export default function CryptoStatCard({ name, value, changePercent, changeText, foot, sparkline }: Props) {
  const color = tone(changePercent);
  const up = changePercent >= 0;
  return (
    <section className="flex flex-col gap-[10px] rounded-2xl border border-app-border bg-app-surface p-[16px_18px]">
      <div className="flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">{name}</span>
        <span className="rounded-md px-2 py-[3px] font-plex-mono text-[11.5px] font-semibold" style={{ color, background: up ? "rgba(53, 199, 127, 0.14)" : "rgba(255, 92, 92, 0.14)" }}>
          {signVN(changePercent, 2)}%
        </span>
      </div>
      <div className="flex items-end justify-between gap-3">
        <div className="flex shrink-0 flex-col gap-[2px]">
          <span className="whitespace-nowrap font-plex-mono text-[23px] font-semibold tracking-[-0.01em]">{value}</span>
          <span className="font-plex-mono text-[12px]" style={{ color }}>{changeText}</span>
        </div>
        {sparkline && sparkline.length > 1 && (
          <svg height="36" viewBox="0 0 112 36" preserveAspectRatio="none" fill="none" aria-hidden="true" className="min-w-0 max-w-[112px] flex-1">
            <polyline points={points(sparkline)} stroke={color} strokeWidth="1.8" strokeLinejoin="round" fill="none" />
          </svg>
        )}
      </div>
      <div className="border-t border-app-hairline pt-[10px] text-[11.5px] text-app-text-muted">{foot}</div>
    </section>
  );
}
