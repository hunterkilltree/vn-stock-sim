"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import type { SectorGroup, TickerChange } from "@/lib/api";
import { formatCompact, formatCryptoPrice, formatThousandsVN, signVN, tone } from "@/lib/format";
import { squarify, type Rect } from "@/lib/treemap";
import { MobileTileGrid } from "@/components/MobileMarket";

type Market = "stock" | "crypto";
type Props = { groups: SectorGroup[]; market: Market };

// Main.dc.html's tile() formula (stocks: intensity saturates at 5%,
// exactly 0 is reference-yellow) and Crypto-Main.dc.html's (12%, with a
// ±0.15% neutral band) -- the full heatmap is the dashboard panel scaled
// up, not a new visual language (phase-i.md decision 15).
function tileColors(pct: number, market: Market) {
  const range = market === "crypto" ? 12 : 5;
  const neutral = market === "crypto" ? 0.15 : 0;
  const a = (0.12 + (Math.min(Math.abs(pct), range) / range) * 0.33).toFixed(2);
  if (pct > neutral) return { bg: `rgba(53, 199, 127, ${a})`, bd: "rgba(53, 199, 127, 0.4)" };
  if (pct < -neutral) return { bg: `rgba(255, 92, 92, ${a})`, bd: "rgba(255, 92, 92, 0.4)" };
  return { bg: "rgba(240, 194, 67, 0.16)", bd: "rgba(240, 194, 67, 0.45)" };
}

// Treemap layout space; rendered with percentages, so it stretches to the
// panel. Sector headers take a fixed strip at the top of each block.
const W = 1000;
const H = 640;
const HEADER = 26;

type Placed = { group: SectorGroup; rect: Rect; tiles: { t: TickerChange; rect: Rect }[] };

export default function HeatmapDashboard({ groups, market }: Props) {
  const [sizeBy, setSizeBy] = useState<"cap" | "equal">("cap");
  const [hover, setHover] = useState<TickerChange | null>(null);

  const placed: Placed[] = useMemo(() => {
    const weight = (t: TickerChange) => (sizeBy === "cap" ? Math.max(t.marketCap ?? 0, 1) : 1);
    // Floor tiny caps so every tile stays visible and clickable.
    const allCaps = groups.flatMap((g) => g.tickers.map(weight));
    const floor = allCaps.reduce((a, b) => a + b, 0) * 0.004;
    const sectorWeights = groups.map((g) => g.tickers.reduce((a, t) => a + Math.max(weight(t), floor), 0));
    const sectorRects = squarify(sectorWeights, { x: 0, y: 0, w: W, h: H });
    return groups.map((g, i) => {
      const r = sectorRects[i];
      const body = { x: 0, y: 0, w: r.w, h: Math.max(r.h - HEADER, 1) };
      const tileRects = squarify(g.tickers.map((t) => Math.max(weight(t), floor)), body);
      return { group: g, rect: r, tiles: g.tickers.map((t, j) => ({ t, rect: tileRects[j] })) };
    });
  }, [groups, sizeBy]);

  const all = groups.flatMap((g) => g.tickers);
  const up = all.filter((t) => (market === "crypto" ? t.changePercent > 0.15 : t.changePercent > 0)).length;
  const down = all.filter((t) => (market === "crypto" ? t.changePercent < -0.15 : t.changePercent < 0)).length;
  const flat = all.length - up - down;
  const sorted = [...all].sort((a, b) => b.changePercent - a.changePercent);
  const bySector = [...groups].sort((a, b) => b.avgChangePercent - a.avgChangePercent);

  const priceText = (t: TickerChange) => (market === "crypto" ? `${formatCryptoPrice(t.price ?? 0)} USDT` : `${formatThousandsVN(t.price ?? 0)} nghìn ₫`);
  const hrefFor = (t: TickerChange) => (market === "crypto" ? `/crypto/${t.symbol}USDT` : `/stocks/${t.symbol}`);
  const capText = (t: TickerChange) => (market === "crypto" ? formatCompact(t.marketCap ?? 0, "USD") : formatCompact(t.marketCap ?? 0, "₫"));

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5 lg:flex-row">
      {/* Phones: one 3-column tile grid per sector, Mobile-Market's own
          heatmap -- the treemap can't label 40 tiles in 354px
          (phase-j.md decision 12). */}
      <section className="flex flex-col gap-4 lg:hidden" aria-label="Bản đồ nhiệt theo nhóm">
        {groups.map((g) => (
          <div key={g.sector} className="flex flex-col gap-2">
            <div className="flex items-baseline justify-between gap-2">
              <h2 className="m-0 truncate text-[13.5px] font-semibold text-app-text-2">{g.sector}</h2>
              <span className="shrink-0 font-plex-mono text-[12px] font-semibold" style={{ color: tone(g.avgChangePercent) }}>
                {signVN(g.avgChangePercent, 2)}%
              </span>
            </div>
            <MobileTileGrid
              range={market === "crypto" ? 12 : 5}
              decimals={market === "crypto" ? 1 : 2}
              tiles={g.tickers.map((t) => ({ symbol: t.symbol, changePercent: t.changePercent, href: hrefFor(t) }))}
            />
          </div>
        ))}
      </section>

      <section className="hidden min-w-0 flex-1 flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[16px_18px] lg:flex">
        <div className="flex items-center justify-between gap-4">
          <span className="text-[12px] text-app-text-muted">
            {all.length} {market === "crypto" ? "coin" : "mã"} · kích thước theo {sizeBy === "cap" ? "vốn hoá" : "số lượng (bằng nhau)"}
          </span>
          <div role="group" aria-label="Kích thước ô" className="flex gap-1 rounded-[10px] border border-app-border p-[3px]">
            {([
              ["cap", "Vốn hoá"],
              ["equal", "Bằng nhau"],
            ] as const).map(([k, label]) => (
              <button
                key={k}
                type="button"
                aria-pressed={sizeBy === k}
                onClick={() => setSizeBy(k)}
                className="h-7 rounded-[7px] px-3 text-[12px]"
                style={{
                  background: sizeBy === k ? "var(--app-surface-3)" : "transparent",
                  color: sizeBy === k ? "var(--app-text)" : "var(--app-text-muted)",
                }}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="relative min-h-[520px] flex-1" data-testid="treemap">
          {placed.map(({ group, rect, tiles }) => (
            <div
              key={group.sector}
              className="absolute box-border p-[2px]"
              style={{ left: `${(rect.x / W) * 100}%`, top: `${(rect.y / H) * 100}%`, width: `${(rect.w / W) * 100}%`, height: `${(rect.h / H) * 100}%` }}
            >
              <div className="flex h-full flex-col overflow-hidden rounded-[10px] border border-app-hairline bg-app-bg">
                <div className="flex h-[26px] shrink-0 items-center gap-2 px-2">
                  <span className="truncate text-[11.5px] font-semibold text-app-text-2">{group.sector}</span>
                  <span className="font-plex-mono text-[11px]" style={{ color: tone(group.avgChangePercent) }}>
                    {signVN(group.avgChangePercent, market === "crypto" ? 1 : 2)}%
                  </span>
                </div>
                <div className="relative flex-1">
                  {tiles.map(({ t, rect: tr }) => {
                    const c = tileColors(t.changePercent, market);
                    const bodyH = Math.max(rect.h - HEADER, 1);
                    const area = tr.w * tr.h;
                    const big = area > 9000;
                    // Layout units are ~0.8 px at 1440 wide: below ~58x40
                    // only the ticker fits, below ~30x20 nothing does.
                    const tiny = tr.w < 30 || tr.h < 20;
                    const small = !big && (tr.w < 58 || tr.h < 40);
                    return (
                      <Link
                        key={t.symbol}
                        href={hrefFor(t)}
                        title={`${t.symbol} · ${t.companyName ?? ""} · ${signVN(t.changePercent, 2)}%`}
                        onMouseEnter={() => setHover(t)}
                        onFocus={() => setHover(t)}
                        className="absolute box-border p-[1.5px]"
                        style={{
                          left: `${(tr.x / rect.w) * 100}%`,
                          top: `${(tr.y / bodyH) * 100}%`,
                          width: `${(tr.w / rect.w) * 100}%`,
                          height: `${(tr.h / bodyH) * 100}%`,
                        }}
                      >
                        <span
                          className="flex h-full w-full flex-col items-center justify-center overflow-hidden rounded-[6px] border text-center text-app-text"
                          style={{ background: c.bg, borderColor: hover?.symbol === t.symbol ? "var(--app-text)" : c.bd }}
                        >
                          {!tiny && (
                            <>
                              <span className={`font-plex-mono font-semibold ${big ? "text-[16px]" : small ? "text-[10.5px]" : "text-[12px]"}`}>{t.symbol}</span>
                              {!small && (
                                <span className={`font-plex-mono ${big ? "text-[13px]" : "text-[10.5px]"}`}>{signVN(t.changePercent, market === "crypto" ? 1 : 2)}%</span>
                              )}
                            </>
                          )}
                        </span>
                      </Link>
                    );
                  })}
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>

      <aside className="flex w-full flex-col gap-4 lg:w-[300px] lg:shrink-0 lg:overflow-y-auto">
        <section className="hidden flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-4 lg:flex">
          <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">{hover ? "Đang xem" : "Di chuột lên một ô để xem chi tiết"}</span>
          {hover && (
            <>
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-plex-mono text-[20px] font-semibold">{market === "crypto" ? `${hover.symbol}/USDT` : hover.symbol}</span>
                <span className="font-plex-mono text-[14px] font-semibold" style={{ color: tone(hover.changePercent) }}>
                  {signVN(hover.changePercent, 2)}%
                </span>
              </div>
              <span className="text-[12px] leading-[1.4] text-app-text-3">{hover.companyName}</span>
              <dl className="m-0 grid grid-cols-2 gap-y-2 text-[12px]">
                <dt className="text-app-text-muted">Giá</dt>
                <dd className="m-0 text-right font-plex-mono">{priceText(hover)}</dd>
                <dt className="text-app-text-muted">Vốn hoá</dt>
                <dd className="m-0 text-right font-plex-mono">{capText(hover)}</dd>
                <dt className="text-app-text-muted">{market === "crypto" ? "KL 24h" : "Khối lượng"}</dt>
                <dd className="m-0 text-right font-plex-mono">{formatCompact(hover.volume ?? 0, market === "crypto" ? "USDT" : "")}</dd>
                {hover.exchange && market === "stock" && (
                  <>
                    <dt className="text-app-text-muted">Sàn</dt>
                    <dd className="m-0 text-right">{hover.exchange}</dd>
                  </>
                )}
              </dl>
              <Link href={hrefFor(hover)} className="text-[12.5px] font-semibold text-app-accent">
                Xem biểu đồ →
              </Link>
            </>
          )}
        </section>

        <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-4">
          <h2 className="m-0 text-[14px] font-semibold">Độ rộng thị trường</h2>
          <div className="flex h-2 overflow-hidden rounded-full" aria-hidden="true">
            <span style={{ width: `${(up / Math.max(all.length, 1)) * 100}%`, background: "#35C77F" }} />
            <span style={{ width: `${(flat / Math.max(all.length, 1)) * 100}%`, background: "#F0C243" }} />
            <span style={{ width: `${(down / Math.max(all.length, 1)) * 100}%`, background: "#FF5C5C" }} />
          </div>
          <div className="flex justify-between font-plex-mono text-[12px]">
            <span className="text-price-up">{up} tăng</span>
            <span className="text-price-ref">{flat} {market === "crypto" ? "đi ngang" : "đứng"}</span>
            <span className="text-price-down">{down} giảm</span>
          </div>
          <div className="flex items-center gap-2 text-[10.5px] text-app-text-muted">
            <span className="font-plex-mono">−{market === "crypto" ? 12 : 5}%</span>
            <span
              className="h-2 flex-1 rounded-full"
              style={{ background: "linear-gradient(90deg, rgba(255,92,92,0.45), rgba(255,92,92,0.12), rgba(240,194,67,0.3), rgba(53,199,127,0.12), rgba(53,199,127,0.45))" }}
            />
            <span className="font-plex-mono">+{market === "crypto" ? 12 : 5}%</span>
          </div>
          {market === "crypto" && <span className="text-[11px] text-app-text-muted">Không có màu trần/sàn — crypto không giới hạn biên độ.</span>}
        </section>

        <section className="flex flex-col gap-2 rounded-2xl border border-app-border bg-app-surface p-4">
          <h2 className="m-0 text-[14px] font-semibold">{market === "crypto" ? "Nhóm coin" : "Ngành"}</h2>
          {bySector.map((g) => (
            <div key={g.sector} className="flex items-center justify-between gap-2 text-[12.5px]">
              <span className="truncate text-app-text-2">{g.sector}</span>
              <span className="flex shrink-0 items-center gap-2">
                <span className="font-plex-mono text-[11px] text-app-text-muted">
                  {g.up ?? 0}↑ {g.down ?? 0}↓
                </span>
                <span className="w-[60px] text-right font-plex-mono font-semibold" style={{ color: tone(g.avgChangePercent) }}>
                  {signVN(g.avgChangePercent, 2)}%
                </span>
              </span>
            </div>
          ))}
        </section>

        <section className="flex flex-col gap-2 rounded-2xl border border-app-border bg-app-surface p-4">
          {[
            { title: "Tăng mạnh nhất", rows: sorted.slice(0, 5) },
            { title: "Giảm mạnh nhất", rows: [...sorted].reverse().slice(0, 5) },
          ].map((col) => (
            <div key={col.title} className="flex flex-col gap-[6px]">
              <h3 className="m-0 text-[12.5px] font-semibold text-app-text-2">{col.title}</h3>
              {col.rows.map((t) => (
                <Link key={t.symbol} href={hrefFor(t)} className="flex justify-between py-[7px] text-[12.5px] text-app-text lg:py-0">
                  <span className="font-plex-mono font-semibold">{t.symbol}</span>
                  <span className="font-plex-mono" style={{ color: tone(t.changePercent) }}>
                    {signVN(t.changePercent, 2)}%
                  </span>
                </Link>
              ))}
            </div>
          ))}
        </section>
        <span className="px-1 text-[11px] leading-[1.5] text-app-text-faint">
          {market === "crypto"
            ? "Giá từ dữ liệu thị trường công khai của Binance; khi không kết nối được sẽ dùng dữ liệu mô phỏng."
            : "Giá (nghìn ₫) từ nguồn dữ liệu thị trường của ứng dụng; vốn hoá là số liệu tham khảo."}
        </span>
      </aside>
    </div>
  );
}
