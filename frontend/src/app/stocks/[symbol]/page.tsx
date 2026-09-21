import { notFound } from "next/navigation";
import { getSymbolDetail, getBars, getIndicator, getInsight, type Bar, type IndicatorPoint, type Insight } from "@/lib/api";
import StockChart from "@/components/StockChart";
import AIInsightCard from "@/components/AIInsightCard";

type Props = { params: Promise<{ symbol: string }> };

const SIX_MONTHS_SECONDS = 180 * 24 * 60 * 60;

// Pulled out of the component body: eslint-config-next's react-hooks/purity
// rule flags Date.now() called directly inside a component/hook, even
// though this is a Server Component that only ever runs once per request,
// not re-rendered like a Client Component.
function lastSixMonthsRange(): { from: number; to: number } {
  const to = Math.floor(Date.now() / 1000);
  return { from: to - SIX_MONTHS_SECONDS, to };
}

export default async function StockDetailPage({ params }: Props) {
  const { symbol } = await params;
  let detail;
  try {
    detail = await getSymbolDetail(symbol);
  } catch {
    notFound();
  }

  // Fetched separately from the detail lookup above: a chart data outage
  // shouldn't take down the whole page when the price/fundamentals load
  // fine, so failures here degrade to an empty chart instead of notFound().
  let bars: Bar[] = [];
  let sma20: IndicatorPoint[] = [];
  try {
    const { from, to } = lastSixMonthsRange();
    const [barsRes, smaRes] = await Promise.all([
      getBars(symbol, "1D", from, to),
      getIndicator(symbol, "1D", "sma", 20, from, to),
    ]);
    bars = barsRes.data;
    sma20 = smaRes.data;
  } catch {
    // leave bars/sma20 empty; the chart section below handles this.
  }

  // Also independent: the insight panel degrades to nothing (not an
  // error) if the backend can't produce one, rather than blocking the
  // rest of the page.
  let insight: Insight | null = null;
  try {
    insight = await getInsight(symbol);
  } catch {
    // leave insight null; the section below just doesn't render.
  }

  const changeColor = detail.change >= 0 ? "text-green-600" : "text-red-600";

  return (
    <main className="mx-auto w-full max-w-6xl p-8 lg:p-12">
      {/* Single column on small screens; on lg+ the info panel becomes a
          fixed-width sidebar next to a wide chart, instead of everything
          squeezed into one narrow centered column. */}
      <div className="lg:grid lg:grid-cols-[340px_1fr] lg:items-start lg:gap-12">
        <div>
          <p className="text-sm text-neutral-500">{detail.exchange} · {detail.sector}</p>
          <h1 className="mb-1 text-3xl font-medium tracking-tight text-neutral-900 lg:text-4xl">
            {detail.symbol}
          </h1>
          <p className="mb-6 text-neutral-500">{detail.companyName}</p>

          <div className="mb-8 flex items-baseline gap-3">
            <span className="text-4xl font-medium text-neutral-900">
              {detail.lastPrice.toLocaleString("vi-VN")}
            </span>
            <span className={`text-lg font-medium ${changeColor}`}>
              {detail.change >= 0 ? "+" : ""}
              {detail.change.toLocaleString("vi-VN")} ({detail.changePercent.toFixed(2)}%)
            </span>
          </div>

          <dl className="grid grid-cols-2 gap-x-6 gap-y-4 rounded-2xl border border-neutral-200 p-5 text-sm">
            <div>
              <dt className="text-neutral-500">Market Cap</dt>
              <dd className="font-medium text-neutral-900">{detail.marketCap.toLocaleString("vi-VN")} VND</dd>
            </div>
            <div>
              <dt className="text-neutral-500">P/E</dt>
              <dd className="font-medium text-neutral-900">{detail.peRatio}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">P/B</dt>
              <dd className="font-medium text-neutral-900">{detail.pbRatio}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">EPS</dt>
              <dd className="font-medium text-neutral-900">{detail.eps.toLocaleString("vi-VN")}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Dividend Yield</dt>
              <dd className="font-medium text-neutral-900">{detail.dividendYield}%</dd>
            </div>
          </dl>
        </div>

        <div className="mt-8 rounded-2xl border border-neutral-800 bg-neutral-950 p-6 lg:mt-0">
          <div className="mb-4 flex items-baseline justify-between">
            <h2 className="text-sm font-medium text-neutral-300">
              Price (6mo, daily) <span className="ml-2 text-blue-400">— SMA(20)</span>
            </h2>
          </div>
          {bars.length > 0 ? (
            <StockChart bars={bars} sma20={sma20} theme="dark" />
          ) : (
            <p className="py-16 text-center text-sm text-neutral-500">
              Chart data unavailable right now — the backend may be starting up or
              unreachable. Indicators beyond SMA/EMA are not implemented yet, see
              RESUME.md.
            </p>
          )}
        </div>
      </div>

      {insight && (
        <div className="mt-8">
          <AIInsightCard insight={insight} />
        </div>
      )}
    </main>
  );
}
