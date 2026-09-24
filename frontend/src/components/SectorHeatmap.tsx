import Link from "next/link";
import type { SectorGroup } from "@/lib/api";
import { signVN, tone } from "@/lib/format";

// tile() color-intensity formula, ported verbatim from
// design/screens/Main.dc.html's script block (phase-c.md decision 6);
// the crypto variant is Crypto-Main.dc.html's (12% range, ±0.15%
// neutral band).
function tileStyle(pct: number, crypto = false): { background: string; borderColor: string } {
  const range = crypto ? 12 : 5;
  const neutral = crypto ? 0.15 : 0;
  const alpha = (0.12 + Math.min(Math.abs(pct), range) / range * 0.33).toFixed(2);
  const background =
    pct > neutral ? `rgba(53, 199, 127, ${alpha})` : pct < -neutral ? `rgba(255, 92, 92, ${alpha})` : "rgba(240, 194, 67, 0.16)";
  const borderColor =
    pct > neutral ? "rgba(53, 199, 127, 0.4)" : pct < -neutral ? "rgba(255, 92, 92, 0.4)" : "rgba(240, 194, 67, 0.45)";
  return { background, borderColor };
}

const CRYPTO_LEGEND = [
  { label: "Tăng", color: "#35C77F" },
  { label: "Đi ngang", color: "#F0C243" },
  { label: "Giảm", color: "#FF5C5C" },
];

// Legend includes Trần/Sàn (ceiling/floor) per the design, but this V1
// heatmap only ever renders the continuous up/down/reference gradient
// (tileStyle above) -- detecting a real ceiling/floor-limit tile would
// need comparing each tile's live price against symbol.Detail's
// Ceiling/Floor bands, which the heatmap endpoint doesn't carry
// per-tile. Documented gap, not a silent omission -- see phase-c.md.
const LEGEND: { label: string; color: string }[] = [
  { label: "Trần", color: "#C08BFF" },
  { label: "Tăng", color: "#35C77F" },
  { label: "Tham chiếu", color: "#F0C243" },
  { label: "Giảm", color: "#FF5C5C" },
  { label: "Sàn", color: "#4FD3E8" },
];

export default function SectorHeatmap({ sectors, market = "stock" }: { sectors: SectorGroup[]; market?: "stock" | "crypto" }) {
  const crypto = market === "crypto";
  const legend = crypto ? CRYPTO_LEGEND : LEGEND;
  return (
    <section className="flex min-h-0 flex-1 flex-col gap-[14px] rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-[2px]">
          <h2 className="m-0 text-[15px] font-semibold">{crypto ? "Bản đồ nhiệt 24 giờ" : "Bản đồ nhiệt thị trường"}</h2>
          {crypto && <span className="text-[11px] text-app-text-muted">Không có màu trần/sàn — crypto không giới hạn biên độ</span>}
        </div>
        <div className="flex items-center gap-[14px] text-[11px] text-app-text-muted">
          <Link href={crypto ? "/heatmap?market=crypto" : "/heatmap"} className="font-semibold text-app-accent">
            Xem toàn bộ →
          </Link>
          {legend.map((l) => (
            <span key={l.label} className="flex items-center gap-[6px]">
              <span className="h-[10px] w-[10px] rounded-[3px]" style={{ background: l.color }} />
              {l.label}
            </span>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-3">
        {/* The panel is drawn for 4 sectors x 6 tiles (largest first, as the
            backend orders them); the rest is on the full /heatmap page. */}
        {sectors.slice(0, 4).map((sec) => (
          <div key={sec.sector} className="flex flex-col gap-[7px]">
            <div className="flex items-baseline gap-2">
              <span className="text-xs font-semibold text-app-text-2">{sec.sector}</span>
              <span className="font-plex-mono text-[11px]" style={{ color: tone(sec.avgChangePercent) }}>
                {signVN(sec.avgChangePercent, 2)}%
              </span>
            </div>
            <div className="grid grid-cols-6 gap-2">
              {sec.tickers.slice(0, 6).map((t) => {
                const s = tileStyle(t.changePercent, crypto);
                return (
                  <Link
                    key={t.symbol}
                    href={crypto ? `/crypto/${t.symbol}USDT` : `/stocks/${t.symbol}`}
                    className="box-border flex h-[46px] flex-col justify-between rounded-lg border p-[6px_9px] text-app-text"
                    style={{ background: s.background, borderColor: s.borderColor }}
                  >
                    <span className="font-plex-mono text-[13px] font-semibold">{t.symbol}</span>
                    <span className="font-plex-mono text-[11.5px] text-app-text">{signVN(t.changePercent, crypto ? 1 : 2)}%</span>
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
