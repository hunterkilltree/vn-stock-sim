import Link from "next/link";
import type { ReactNode } from "react";
import { signVN, tone } from "@/lib/format";

// Phone building blocks from design/screens/Mobile-Market.dc.html, shared
// by the stock Market page, the Crypto overview and the phone heatmap
// (phase-j.md decisions 5, 12, 13). All of them are `lg:hidden` at the
// call site; from lg up the desktop panels render instead.

function spark(values: number[], w: number, h: number): string {
  if (values.length < 2) return "";
  const min = Math.min(...values);
  const span = Math.max(...values) - min || 1;
  return values.map((v, i) => `${((i / (values.length - 1)) * w).toFixed(1)},${((1 - (v - min) / span) * (h - 8) + 4).toFixed(1)}`).join(" ");
}

// The VN-Index hero card: name + % chip, big value, change, sparkline,
// and a footer row.
export function MobileHero({
  name,
  value,
  changePercent,
  changeText,
  sparkline,
  foot,
}: {
  name: string;
  value: string;
  changePercent: number;
  changeText: string;
  sparkline: number[];
  foot: ReactNode;
}) {
  const color = tone(changePercent);
  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-4">
      <div className="flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">{name}</span>
        <span
          className="rounded-md px-2 py-[3px] font-plex-mono text-[11.5px] font-semibold"
          style={{ color, background: changePercent >= 0 ? "rgba(53, 199, 127, 0.14)" : "rgba(255, 92, 92, 0.14)" }}
        >
          {signVN(changePercent, 2)}%
        </span>
      </div>
      <div className="flex items-end justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-[3px]">
          <span className="font-plex-mono text-[30px] font-semibold tracking-[-0.02em]">{value}</span>
          <span className="font-plex-mono text-[12.5px]" style={{ color }}>
            {changeText}
          </span>
        </div>
        <svg viewBox="0 0 120 46" preserveAspectRatio="none" height="46" fill="none" aria-hidden="true" className="min-w-0 max-w-[120px] flex-1">
          <polyline points={spark(sparkline, 120, 46)} stroke={color} strokeWidth="2" strokeLinejoin="round" fill="none" />
        </svg>
      </div>
      <div className="flex justify-between gap-2 border-t border-app-hairline pt-3 text-[11.5px] text-app-text-muted">{foot}</div>
    </section>
  );
}

export type Chip = { name: string; value: string; changePercent: number; href?: string };

// The row of three smaller index chips under the hero.
export function MobileChips({ chips }: { chips: Chip[] }) {
  return (
    <div className="flex gap-[10px]">
      {chips.map((c) => {
        const body = (
          <>
            <span className="truncate text-[10px] uppercase tracking-[0.07em] text-app-text-muted">{c.name}</span>
            <span className="truncate font-plex-mono text-[14px] font-semibold">{c.value}</span>
            <span className="font-plex-mono text-[11.5px]" style={{ color: tone(c.changePercent) }}>
              {signVN(c.changePercent, 2)}%
            </span>
          </>
        );
        const cls = "flex min-w-0 flex-1 flex-col gap-1 rounded-[13px] border border-app-border bg-app-surface p-[11px_12px] text-app-text";
        return c.href ? (
          <Link key={c.name} href={c.href} className={cls}>
            {body}
          </Link>
        ) : (
          <div key={c.name} className={cls}>
            {body}
          </div>
        );
      })}
    </div>
  );
}

export type Tile = { symbol: string; label?: string; changePercent: number; href: string };

// Mobile-Market's tile formula: alpha 0.12 + |p|/range * 0.33, capped at
// `range` percent (5 for stocks; crypto uses its wider 12, phase-i.md).
function tileStyle(p: number, range: number) {
  if (Math.abs(p) < 0.005) return { background: "rgba(240, 194, 67, 0.16)", borderColor: "rgba(240, 194, 67, 0.4)" };
  const a = (0.12 + (Math.min(Math.abs(p), range) / range) * 0.33).toFixed(2);
  return p > 0
    ? { background: `rgba(53, 199, 127, ${a})`, borderColor: "rgba(53, 199, 127, 0.4)" }
    : { background: `rgba(255, 92, 92, ${a})`, borderColor: "rgba(255, 92, 92, 0.4)" };
}

// The design's phone heatmap: a 3-column grid of 52px tiles.
export function MobileTileGrid({ tiles, range = 5, decimals = 2 }: { tiles: Tile[]; range?: number; decimals?: number }) {
  return (
    <div className="grid grid-cols-3 gap-2">
      {tiles.map((t) => (
        <Link
          key={t.symbol}
          href={t.href}
          className="box-border flex h-[52px] flex-col justify-between rounded-[10px] border p-[7px_10px] text-app-text"
          style={tileStyle(t.changePercent, range)}
        >
          <span className="font-plex-mono text-[13.5px] font-semibold">{t.label ?? t.symbol}</span>
          <span className="font-plex-mono text-[11.5px] text-[#E8E3D8]">{signVN(t.changePercent, decimals)}%</span>
        </Link>
      ))}
    </div>
  );
}

export type MoverRow = { symbol: string; label?: string; name: string; price: string; changePercent: number; href: string };

// "Tăng mạnh nhất" as full-width tappable cards.
export function MobileMoverList({ rows }: { rows: MoverRow[] }) {
  return (
    <div className="flex flex-col gap-2">
      {rows.map((r) => {
        const color = tone(r.changePercent);
        return (
          <Link
            key={r.symbol}
            href={r.href}
            className="box-border flex min-h-14 items-center gap-3 rounded-[13px] border border-app-hairline bg-app-surface p-[10px_14px] text-app-text"
          >
            <span className="flex min-w-0 flex-1 flex-col gap-[2px]">
              <span className="font-plex-mono text-[14px] font-semibold">{r.label ?? r.symbol}</span>
              <span className="truncate text-[11px] text-app-text-muted">{r.name}</span>
            </span>
            <span className="flex flex-col items-end gap-[2px]">
              <span className="font-plex-mono text-[14px]" style={{ color }}>
                {r.price}
              </span>
              <span className="font-plex-mono text-[11.5px]" style={{ color }}>
                {signVN(r.changePercent, 2)}%
              </span>
            </span>
          </Link>
        );
      })}
    </div>
  );
}

// Section heading row: title on the left, a link or caption on the right.
export function MobileSectionHead({ title, right }: { title: string; right?: ReactNode }) {
  return (
    <div className="flex items-center justify-between">
      <h2 className="m-0 text-[14.5px] font-semibold">{title}</h2>
      {right}
    </div>
  );
}
