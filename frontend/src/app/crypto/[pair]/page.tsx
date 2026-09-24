import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import RailNav from "@/components/RailNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import DetailChart from "@/components/DetailChart";
import CryptoOrderTicket from "@/components/CryptoOrderTicket";
import CryptoOrderBook from "@/components/CryptoOrderBook";
import {
  getCryptoBars,
  getCryptoOrderBook,
  getCryptoPair,
  getPortfolioPositionsByID,
  getPortfolioSummaryByID,
  type Bar,
  type CryptoInterval,
  type CryptoPairDetail,
  type DepthLevel,
} from "@/lib/api";
import { macd, rsi, sma } from "@/lib/indicators";
import { formatAmount, formatCompact, formatCryptoPrice, formatVN, signVN, tone } from "@/lib/format";
import { getActivePortfolio, getSessionToken, getSessionUser } from "@/lib/session";

type Props = {
  params: Promise<{ pair: string }>;
  searchParams: Promise<{ tf?: string }>;
};

// Crypto-Detail.dc.html's timeframe pills; "4 giờ" is the design's default.
const FRAMES: { key: CryptoInterval; label: string; seconds: number }[] = [
  { key: "5m", label: "5 phút", seconds: 300 },
  { key: "15m", label: "15 phút", seconds: 900 },
  { key: "1h", label: "1 giờ", seconds: 3600 },
  { key: "4h", label: "4 giờ", seconds: 14_400 },
  { key: "1d", label: "1 ngày", seconds: 86_400 },
  { key: "1w", label: "1 tuần", seconds: 604_800 },
];

const SHOWN_BARS = 120;
const WARMUP_BARS = 60; // so SMA 50 / MACD start at the chart's left edge

const INDICATOR_CHIPS: { label: string; color: string }[] = [
  { label: "SMA 20", color: "#E08A3C" },
  { label: "SMA 50", color: "#7FA2FF" },
  { label: "RSI 14", color: "#C08BFF" },
  { label: "MACD", color: "#4FD3E8" },
];

function rangeFor(seconds: number): { from: number; to: number } {
  const to = Math.floor(Date.now() / 1000);
  return { from: to - (SHOWN_BARS + WARMUP_BARS) * seconds, to };
}

// Chart price labels: 2 decimals above 1 USDT, ~4 significant digits below.
function chartDecimals(price: number): number {
  if (price >= 1 || price <= 0) return 2;
  return Math.min(10, Math.max(4, 3 - Math.floor(Math.log10(price))));
}

export async function generateMetadata({ params }: Props) {
  const { pair } = await params;
  return { title: `${pair.toUpperCase()} — VN Stock Sim` };
}

// design/screens/Crypto-Detail.dc.html (phase-i.md). Indicators are
// computed from the fetched bars (lib/indicators.ts) because the
// backend's indicator endpoints serve the stock market only.
export default async function CryptoDetailPage({ params, searchParams }: Props) {
  const { pair } = await params;
  const { tf: tfParam } = await searchParams;
  const frame = FRAMES.find((f) => f.key === tfParam) ?? FRAMES[3];

  let detail: CryptoPairDetail;
  try {
    detail = await getCryptoPair(pair);
  } catch {
    notFound();
  }
  if (detail.symbol !== pair) redirect(`/crypto/${detail.symbol}${tfParam ? `?tf=${tfParam}` : ""}`); // canonical: /crypto/BTCUSDT

  const { from, to } = rangeFor(frame.seconds);
  let all: Bar[] = [];
  let book: { bids: DepthLevel[]; asks: DepthLevel[]; source: string } = { bids: [], asks: [], source: "mock" };
  const [barsRes, bookRes] = await Promise.allSettled([getCryptoBars(detail.symbol, frame.key, from, to), getCryptoOrderBook(detail.symbol, 4)]);
  if (barsRes.status === "fulfilled") all = barsRes.value.data;
  if (bookRes.status === "fulfilled") book = bookRes.value.data;

  const bars = all.slice(-SHOWN_BARS);
  const firstTime = bars[0]?.time ?? 0;
  const shown = <T extends { time: number }>(pts: T[]) => pts.filter((p) => p.time >= firstTime);
  const sma20 = shown(sma(all, 20));
  const sma50 = shown(sma(all, 50));
  const rsi14 = shown(rsi(all, 14));
  const macdPts = shown(macd(all));

  // null = guest, undefined = signed in without a crypto wallet.
  let usdtBalance: number | null | undefined = null;
  let positionQty = 0;
  let walletName: string | null = null;
  const user = await getSessionUser();
  if (user) {
    usdtBalance = undefined;
    try {
      const token = await getSessionToken();
      if (token) {
        const { active } = await getActivePortfolio(token, "crypto");
        if (active) {
          walletName = active.name;
          const [summary, positions] = await Promise.all([
            getPortfolioSummaryByID(active.id, token),
            getPortfolioPositionsByID(active.id, token).then((r) => r.data),
          ]);
          usdtBalance = summary.cashBalance;
          positionQty = positions.find((p) => p.symbol === detail.symbol)?.quantity ?? 0;
        }
      }
    } catch {
      // ticket falls back to its no-wallet state
    }
  }

  const changeColor = tone(detail.changePercent);
  const live = detail.source === "binance";
  const chips = [
    { k: "Cao 24h", v: formatCryptoPrice(detail.high24h), c: "var(--app-text-2)" },
    { k: "Thấp 24h", v: formatCryptoPrice(detail.low24h), c: "var(--app-text-2)" },
    { k: "KL 24h", v: formatCompact(detail.quoteVolume24h, "USDT"), c: "var(--app-text-2)" },
    { k: "Cách ATH", v: `${signVN(detail.distanceFromAthPercent, 2)}%`, c: tone(detail.distanceFromAthPercent) },
  ];
  const facts = [
    { k: "Vốn hoá", v: formatCompact(detail.marketCap, "USD") },
    { k: "KL 24h", v: formatCompact(detail.quoteVolume24h, "USDT") },
    { k: "Cung lưu hành", v: formatCompact(detail.circulatingSupply, detail.base) },
    { k: "Tổng cung tối đa", v: detail.maxSupply > 0 ? formatCompact(detail.maxSupply, detail.base) : "Không giới hạn" },
    { k: "Đỉnh lịch sử", v: formatCryptoPrice(detail.allTimeHigh) },
    { k: "Biến động 30 ngày", v: `${formatVN(detail.volatility30dPercentDay, 2)}%/ngày` },
  ];

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <RailNav mode="crypto" account={<AccountMenuButton placement="right" />} />

      <div className="flex min-w-0 flex-1 flex-col gap-4 p-[20px_24px]">
        <header className="flex flex-wrap items-center justify-between gap-6">
          <div className="flex flex-wrap items-center gap-[22px]">
            <div className="flex flex-col gap-[3px]">
              <div className="flex items-center gap-[9px]">
                <h1 className="m-0 font-display text-[26px] font-bold tracking-[-0.01em]">
                  {detail.base} / {detail.quote}
                </h1>
                <span className="rounded-[5px] border border-app-border px-[7px] py-[2px] text-[10.5px] text-app-text-3">Giao ngay</span>
              </div>
              <span className="text-[12.5px] text-app-text-muted">
                {detail.name} · {detail.category} · {live ? "giá từ Binance" : "dữ liệu mô phỏng"}
              </span>
            </div>
            <div className="flex items-baseline gap-[10px]">
              <span className="font-plex-mono text-[30px] font-semibold tracking-[-0.02em]" style={{ color: changeColor }}>
                {formatCryptoPrice(detail.lastPrice)}
              </span>
              <span className="font-plex-mono text-sm font-semibold" style={{ color: changeColor }}>
                {detail.change >= 0 ? "+" : "−"}
                {formatCryptoPrice(Math.abs(detail.change))} ({signVN(detail.changePercent, 2)}%)
              </span>
            </div>
            <div className="flex gap-2">
              {chips.map((c) => (
                <div key={c.k} className="box-border flex flex-col gap-[2px] rounded-[9px] border border-app-border bg-app-surface px-[11px] py-[6px]">
                  <span className="text-[9.5px] uppercase tracking-[0.08em] text-app-text-muted">{c.k}</span>
                  <span className="font-plex-mono text-[12.5px] font-semibold" style={{ color: c.c }}>
                    {c.v}
                  </span>
                </div>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-[10px]">
            <span className="box-border flex h-11 items-center gap-[7px] rounded-[11px] border border-app-border bg-app-surface px-[13px] text-xs text-app-text-3">
              <span className="h-[7px] w-[7px] rounded-full bg-price-up" />
              Thị trường mở 24/7
            </span>
            <Link
              href={`/crypto/replay?symbol=${detail.symbol}`}
              className="box-border flex h-11 items-center gap-2 rounded-[11px] border border-app-accent-border bg-app-accent-surface px-[15px] text-[13px] font-semibold text-app-accent"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
              </svg>
              <span>Replay cặp này</span>
            </Link>
          </div>
        </header>

        <div className="flex min-h-0 flex-1 gap-5">
          <div className="flex min-w-0 flex-1 flex-col gap-[14px]">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <nav aria-label="Khung thời gian" className="flex gap-[6px]">
                {FRAMES.map((f) => {
                  const on = f.key === frame.key;
                  return (
                    <Link
                      key={f.key}
                      href={`/crypto/${detail.symbol}?tf=${f.key}`}
                      aria-current={on ? "page" : undefined}
                      className="box-border flex h-[34px] items-center rounded-[9px] border px-[13px] text-[12.5px]"
                      style={{
                        borderColor: on ? "var(--app-border-strong)" : "var(--app-border)",
                        background: on ? "var(--app-border)" : "var(--app-surface)",
                        color: on ? "var(--app-text)" : "var(--app-text-muted)",
                        fontWeight: on ? 600 : 500,
                      }}
                    >
                      {f.label}
                    </Link>
                  );
                })}
              </nav>
              <div className="flex items-center gap-2">
                {INDICATOR_CHIPS.map((c) => (
                  <span
                    key={c.label}
                    className="flex h-7 items-center gap-[7px] rounded-lg border border-app-border bg-app-surface-2 px-[11px] text-[11.5px] text-app-text-2"
                  >
                    <span className="h-2 w-2 rounded-[2px]" style={{ background: c.color }} />
                    {c.label}
                  </span>
                ))}
              </div>
            </div>

            <DetailChart bars={bars} sma20={sma20} sma50={sma50} rsi={rsi14} macd={macdPts} scale={1} decimals={chartDecimals(detail.lastPrice)} />

            <section className="rounded-2xl border border-app-border bg-app-surface p-4">
              <div className="mb-3 flex items-center justify-between">
                <h2 className="m-0 text-[15px] font-semibold">Thông tin {detail.base}</h2>
                {walletName && positionQty > 0 && (
                  <span className="text-[12px] text-app-text-muted">
                    Đang nắm giữ <span className="font-plex-mono text-app-text">{formatAmount(positionQty)} {detail.base}</span> trong {walletName}
                  </span>
                )}
              </div>
              <dl className="m-0 grid grid-cols-3 gap-x-6 gap-y-3">
                {facts.map((f) => (
                  <div key={f.k} className="flex flex-col gap-[3px]">
                    <dt className="text-[11px] text-app-text-muted">{f.k}</dt>
                    <dd className="m-0 font-plex-mono text-[13px] font-semibold">{f.v}</dd>
                  </div>
                ))}
              </dl>
            </section>
          </div>

          <div className="flex w-[344px] shrink-0 flex-col gap-4">
            <CryptoOrderTicket
              key={detail.symbol}
              symbol={detail.symbol}
              base={detail.base}
              lastPrice={detail.lastPrice}
              usdtBalance={usdtBalance}
              positionQty={positionQty}
            />
            <CryptoOrderBook bids={book.bids} asks={book.asks} base={detail.base} source={book.source} />
          </div>
        </div>
      </div>
    </div>
  );
}
