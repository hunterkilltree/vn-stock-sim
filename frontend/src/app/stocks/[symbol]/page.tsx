import { notFound } from "next/navigation";
import {
  getSymbolDetail,
  getBars,
  getIndicator,
  getMACD,
  getOrderBook,
  getInsight,
  getPortfolioSummaryByID,
  getWatchlist,
  type Bar,
  type IndicatorPoint,
  type IndicatorMultiPoint,
  type PriceLevel,
  type Insight,
} from "@/lib/api";
import RailNav from "@/components/RailNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import PriceBandChips from "@/components/PriceBandChips";
import TimeframePills, { TIMEFRAMES, type TimeframeKey } from "@/components/TimeframePills";
import DetailChart from "@/components/DetailChart";
import OrderTicket from "@/components/OrderTicket";
import OrderBookPanel from "@/components/OrderBookPanel";
import FundamentalsGrid from "@/components/FundamentalsGrid";
import AIInsightCard from "@/components/AIInsightCard";
import CompactChart from "@/components/CompactChart";
import MobileActionBar from "@/components/MobileActionBar";
import WatchlistButton from "@/components/WatchlistButton";
import Link from "next/link";
import { formatThousandsVN, formatVN, formatVolumeVN, signVN, tone } from "@/lib/format";
import { getActivePortfolio, getSessionToken, getSessionUser } from "@/lib/session";

type Props = {
  params: Promise<{ symbol: string }>;
  searchParams: Promise<{ tf?: string; side?: string }>;
};

const INDICATOR_CHIPS: { label: string; color: string }[] = [
  { label: "SMA 20", color: "#E08A3C" },
  { label: "SMA 50", color: "#7FA2FF" },
  { label: "RSI 14", color: "#C08BFF" },
  { label: "MACD", color: "#4FD3E8" },
];

// resolution/range per timeframe pill -- see phase-d.md decision 6.
const TF_RANGES: Record<TimeframeKey, { resolution: string; days: number }> = {
  "1d": { resolution: "5", days: 1 },
  "1w": { resolution: "60", days: 7 },
  "1m": { resolution: "1D", days: 30 },
  "3m": { resolution: "1D", days: 90 },
  "1y": { resolution: "1D", days: 365 },
  "5y": { resolution: "1D", days: 1825 },
};

// Pulled out of the component body: eslint-config-next's react-hooks/purity
// rule flags Date.now() called directly inside a component, even for a
// Server Component that only ever runs once per request (same fix
// already applied elsewhere in this repo, e.g. the home page hero).
function rangeFor(days: number): { from: number; to: number } {
  const to = Math.floor(Date.now() / 1000);
  return { from: to - days * 24 * 60 * 60, to };
}

export default async function StockDetailPage({ params, searchParams }: Props) {
  const { symbol } = await params;
  const { tf: tfParam, side: sideParam } = await searchParams;
  const initialSide = sideParam === "sell" ? "sell" : "buy";
  const tf: TimeframeKey = TIMEFRAMES.some((f) => f.key === tfParam) ? (tfParam as TimeframeKey) : "1m";
  const { resolution, days } = TF_RANGES[tf];

  let detail;
  try {
    detail = await getSymbolDetail(symbol);
  } catch {
    notFound();
  }

  const { from, to } = rangeFor(days);

  let bars: Bar[] = [];
  let sma20: IndicatorPoint[] = [];
  let sma50: IndicatorPoint[] = [];
  let rsi: IndicatorPoint[] = [];
  let macd: IndicatorMultiPoint[] = [];
  try {
    const [barsRes, sma20Res, sma50Res, rsiRes, macdRes] = await Promise.all([
      getBars(symbol, resolution, from, to),
      getIndicator(symbol, resolution, "sma", 20, from, to),
      getIndicator(symbol, resolution, "sma", 50, from, to),
      getIndicator(symbol, resolution, "rsi", 14, from, to),
      getMACD(symbol, resolution, from, to),
    ]);
    bars = barsRes.data;
    sma20 = sma20Res.data;
    sma50 = sma50Res.data;
    rsi = rsiRes.data;
    macd = macdRes.data;
  } catch {
    // leave chart series empty; DetailChart's own empty-state handles it.
  }

  const avgVolume20d =
    bars.length > 0 ? bars.slice(-20).reduce((sum, b) => sum + b.volume, 0) / Math.min(20, bars.length) : null;

  let orderBook: { bids: PriceLevel[]; asks: PriceLevel[] } = { bids: [], asks: [] };
  try {
    orderBook = (await getOrderBook(symbol)).data;
  } catch {
    // leave empty; OrderBookPanel renders an empty ladder rather than failing.
  }

  let insight: Insight | null = null;
  try {
    insight = await getInsight(symbol);
  } catch {
    // leave insight null.
  }

  const user = await getSessionUser();
  let buyingPower: number | null = null;
  let activePortfolioName: string | null = null;
  let watched = false;
  if (user) {
    try {
      const token = await getSessionToken();
      if (token) {
        watched = (await getWatchlist(token)).data.some((w) => w.symbol === detail.symbol);
        const { active } = await getActivePortfolio(token);
        if (active) {
          activePortfolioName = active.name;
          buyingPower = (await getPortfolioSummaryByID(active.id, token)).cashBalance;
        }
      }
    } catch {
      // leave buyingPower null; OrderTicket falls back to its guest treatment.
    }
  }

  const changeColor = tone(detail.changePercent);
  // Phone chart: the last 40 bars, as in Mobile-Detail.dc.html.
  const phoneBars = bars.slice(-40);
  const phoneFrom = phoneBars[0]?.time ?? 0;
  const ticketHref = (side: "buy" | "sell") => `/stocks/${detail.symbol}?tf=${tf}&side=${side}#dat-lenh`;

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <RailNav account={<AccountMenuButton placement="right" />} tabBar={false} />

      <div className="flex min-w-0 flex-1 flex-col gap-[14px] px-[18px] pt-[22px] lg:gap-4 lg:p-[20px_24px]">
        {/* Phone header + price block (Mobile-Detail.dc.html). */}
        <header className="flex items-center gap-3 lg:hidden">
          <Link
            href="/stocks"
            aria-label="Trở lại thị trường"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-app-border bg-app-surface text-app-text-3"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M15 5l-7 7 7 7" />
            </svg>
          </Link>
          <div className="flex min-w-0 flex-1 flex-col gap-[1px]">
            <div className="flex items-center gap-2">
              <h1 className="m-0 font-display text-[20px] font-bold">{detail.symbol}</h1>
              <span className="rounded-[5px] border border-app-border px-[6px] py-[1px] text-[10px] text-app-text-muted">{detail.exchange}</span>
            </div>
            <span className="truncate text-[11.5px] text-app-text-muted">{detail.companyName}</span>
          </div>
          <WatchlistButton symbol={detail.symbol} watched={watched} signedIn={!!user} />
          <AccountMenuButton placement="below" />
        </header>
        <div className="flex items-end justify-between gap-3 lg:hidden">
          <div className="flex flex-col gap-[3px]">
            <span className="font-plex-mono text-[32px] font-semibold tracking-[-0.02em]" style={{ color: changeColor }}>
              {formatThousandsVN(detail.lastPrice)}
            </span>
            <span className="font-plex-mono text-[13px]" style={{ color: changeColor }}>
              {signVN(detail.change / 1000, 2)} ({signVN(detail.changePercent, 2)}%)
            </span>
          </div>
          <div className="flex flex-col items-end gap-1 font-plex-mono text-[11px]">
            <span className="text-price-ceiling">Trần {formatThousandsVN(detail.ceiling)}</span>
            <span className="text-price-ref">TC {formatThousandsVN(detail.reference)}</span>
            <span className="text-price-floor">Sàn {formatThousandsVN(detail.floor)}</span>
          </div>
        </div>

        <header className="hidden items-center justify-between gap-6 lg:flex">
          <div className="flex items-center gap-[22px]">
            <div className="flex flex-col gap-[3px]">
              <div className="flex items-center gap-[9px]">
                <h1 className="m-0 font-display text-[26px] font-bold tracking-[-0.01em]">{detail.symbol}</h1>
                <span className="rounded-[5px] border border-app-border px-[7px] py-[2px] text-[10.5px] text-app-text-3">{detail.exchange}</span>
              </div>
              <span className="text-[12.5px] text-app-text-muted">
                {detail.companyName} · {detail.sector}
              </span>
            </div>
            <div className="flex items-baseline gap-[10px]">
              <span className="font-plex-mono text-[30px] font-semibold tracking-[-0.02em]" style={{ color: changeColor }}>
                {formatThousandsVN(detail.lastPrice)}
              </span>
              <span className="font-plex-mono text-sm font-semibold" style={{ color: changeColor }}>
                {signVN(detail.change / 1000, 2)} ({signVN(detail.changePercent, 2)}%)
              </span>
            </div>
            <PriceBandChips ceiling={detail.ceiling} reference={detail.reference} floor={detail.floor} />
          </div>
          <div className="flex items-center gap-[10px]">
            <WatchlistButton symbol={detail.symbol} watched={watched} signedIn={!!user} />
            <button
              type="button"
              disabled
              title="Sắp ra mắt"
              aria-label="Thêm chỉ báo"
              className="box-border flex h-11 cursor-not-allowed items-center gap-2 whitespace-nowrap rounded-[11px] border border-app-border bg-app-surface px-[13px] text-[13px] font-medium text-app-text opacity-70 xl:px-[15px]"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
                <path d="M12 5v14M5 12h14" />
              </svg>
              <span className="hidden xl:inline">Thêm chỉ báo</span>
            </button>
            <Link
              href={`/backtest?symbol=${detail.symbol}`}
              aria-label="Kiểm thử mã này"
              className="box-border flex h-11 items-center gap-2 whitespace-nowrap rounded-[11px] border border-app-border bg-app-surface px-[13px] text-[13px] font-medium text-app-text xl:px-[15px]"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M12 21a9 9 0 100-18 9 9 0 000 18zM12 7.5V12l3 2" />
              </svg>
              <span className="hidden xl:inline">Kiểm thử mã này</span>
            </Link>
            <Link
              href={`/replay?symbol=${detail.symbol}`}
              aria-label="Replay mã này"
              className="box-border flex h-11 items-center gap-2 whitespace-nowrap rounded-[11px] border border-app-accent-border bg-app-accent-surface px-[13px] text-[13px] font-semibold text-app-accent xl:px-[15px]"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
              </svg>
              <span className="hidden xl:inline">Replay mã này</span>
            </Link>
          </div>
        </header>

        <div className="flex min-h-0 flex-1 flex-col gap-5 lg:flex-row">
          <div className="flex min-w-0 flex-1 flex-col gap-[14px]">
            <div className="flex items-center justify-between gap-4">
              <div className="-mx-[18px] overflow-x-auto px-[18px] lg:mx-0 lg:px-0">
                <TimeframePills symbol={symbol} active={tf} />
              </div>
              {/* Chips from xl up: at lg they'd wrap beside the timeframes. */}
              <div className="hidden items-center gap-2 xl:flex">
                {INDICATOR_CHIPS.map((c) => (
                  <span
                    key={c.label}
                    className="flex h-7 items-center gap-[7px] whitespace-nowrap rounded-lg border border-app-border bg-app-surface-2 px-[11px] text-[11.5px] text-app-text-2"
                  >
                    <span className="h-2 w-2 rounded-[2px]" style={{ background: c.color }} />
                    {c.label}
                  </span>
                ))}
              </div>
            </div>

            <div className="lg:hidden">
              <CompactChart
                bars={phoneBars}
                sma20={sma20.filter((p) => p.time >= phoneFrom)}
                rsi={rsi.filter((p) => p.time >= phoneFrom)}
                ariaLabel={`Biểu đồ nến của ${detail.symbol} kèm đường trung bình động SMA 20`}
              />
            </div>
            <div className="grid grid-cols-[1fr_1fr_1fr_1.4fr] gap-2 lg:hidden">
              {[
                { k: "KL", v: bars.length > 0 ? formatVolumeVN(bars[bars.length - 1].volume) : "—" },
                { k: "P/E", v: formatVN(detail.peRatio, 1) },
                { k: "ROE", v: `${formatVN(detail.roe, 1)}%` },
                // Mobile-Detail's short form: "188 ngh.tỷ".
                { k: "Vốn hóa", v: detail.marketCap >= 1e12 ? `${formatVN(detail.marketCap / 1e12, 0)} ngh.tỷ` : `${formatVN(detail.marketCap / 1e9, 0)} tỷ` },
              ].map((x) => (
                <div key={x.k} className="flex min-w-0 flex-col gap-[3px] rounded-[11px] border border-app-hairline bg-app-surface p-[9px_10px]">
                  <span className="text-[9.5px] text-app-text-muted">{x.k}</span>
                  <span className="truncate font-plex-mono text-[12.5px] font-semibold">{x.v}</span>
                </div>
              ))}
            </div>

            <div className="hidden lg:block">
              <DetailChart bars={bars} sma20={sma20} sma50={sma50} rsi={rsi} macd={macd} />
            </div>

            {insight && <AIInsightCard insight={insight} />}
          </div>

          <div id="dat-lenh" className="flex w-full scroll-mt-4 flex-col gap-4 lg:w-[344px] lg:shrink-0">
            <OrderTicket
              key={initialSide}
              symbol={detail.symbol}
              lastPrice={detail.lastPrice}
              buyingPower={buyingPower}
              portfolioName={activePortfolioName}
              initialSide={initialSide}
            />
            <OrderBookPanel bids={orderBook.bids} asks={orderBook.asks} />
            <FundamentalsGrid detail={detail} avgVolume20d={avgVolume20d} />
          </div>
        </div>

        <MobileActionBar replayHref={`/replay?symbol=${detail.symbol}`} buyHref={ticketHref("buy")} sellHref={ticketHref("sell")} />
      </div>
    </div>
  );
}
