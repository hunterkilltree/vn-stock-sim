import { notFound } from "next/navigation";
import {
  getSymbolDetail,
  getBars,
  getIndicator,
  getMACD,
  getOrderBook,
  getInsight,
  getPortfolioSummary,
  type Bar,
  type IndicatorPoint,
  type IndicatorMultiPoint,
  type PriceLevel,
  type Insight,
} from "@/lib/api";
import RailNav from "@/components/RailNav";
import PriceBandChips from "@/components/PriceBandChips";
import TimeframePills, { TIMEFRAMES, type TimeframeKey } from "@/components/TimeframePills";
import DetailChart from "@/components/DetailChart";
import OrderTicket from "@/components/OrderTicket";
import OrderBookPanel from "@/components/OrderBookPanel";
import FundamentalsGrid from "@/components/FundamentalsGrid";
import AIInsightCard from "@/components/AIInsightCard";
import { formatThousandsVN, signVN, tone } from "@/lib/format";
import { getSessionToken, getSessionUser } from "@/lib/session";

type Props = {
  params: Promise<{ symbol: string }>;
  searchParams: Promise<{ tf?: string }>;
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
  const { tf: tfParam } = await searchParams;
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
  if (user) {
    try {
      const token = await getSessionToken();
      if (token) buyingPower = (await getPortfolioSummary(token)).cashBalance;
    } catch {
      // leave buyingPower null; OrderTicket falls back to its guest treatment.
    }
  }

  const changeColor = tone(detail.changePercent);

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <RailNav />

      <div className="flex min-w-0 flex-1 flex-col gap-4 p-[20px_24px]">
        <header className="flex items-center justify-between gap-6">
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
            <button
              type="button"
              disabled
              title="Sắp ra mắt"
              className="box-border flex h-11 cursor-not-allowed items-center gap-2 rounded-[11px] border border-app-border bg-app-surface px-[15px] text-[13px] font-medium text-app-text opacity-70"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
                <path d="M12 5v14M5 12h14" />
              </svg>
              <span>Thêm chỉ báo</span>
            </button>
            <button
              type="button"
              disabled
              title="Sắp ra mắt"
              className="box-border flex h-11 cursor-not-allowed items-center gap-2 rounded-[11px] border border-app-accent-border bg-app-accent-surface px-[15px] text-[13px] font-semibold text-app-accent opacity-80"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
              </svg>
              <span>Replay mã này</span>
            </button>
          </div>
        </header>

        <div className="flex min-h-0 flex-1 gap-5">
          <div className="flex min-w-0 flex-1 flex-col gap-[14px]">
            <div className="flex items-center justify-between gap-4">
              <TimeframePills symbol={symbol} active={tf} />
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

            <DetailChart bars={bars} sma20={sma20} sma50={sma50} rsi={rsi} macd={macd} />

            {insight && <AIInsightCard insight={insight} />}
          </div>

          <div className="flex w-[344px] shrink-0 flex-col gap-4">
            <OrderTicket symbol={detail.symbol} lastPrice={detail.lastPrice} buyingPower={buyingPower} />
            <OrderBookPanel bids={orderBook.bids} asks={orderBook.asks} />
            <FundamentalsGrid detail={detail} avgVolume20d={avgVolume20d} />
          </div>
        </div>
      </div>
    </div>
  );
}
