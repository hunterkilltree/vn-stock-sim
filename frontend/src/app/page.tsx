import Button from "@/components/Button";
import StockChart from "@/components/StockChart";
import { getBars, getIndicator, type Bar, type IndicatorPoint } from "@/lib/api";

const SIX_MONTHS_SECONDS = 180 * 24 * 60 * 60;

function lastSixMonthsRange(): { from: number; to: number } {
  const to = Math.floor(Date.now() / 1000);
  return { from: to - SIX_MONTHS_SECONDS, to };
}

// Hero layout follows luxalgo.com (badge + bold headline + subtext + two
// pill CTAs, then a dark card showing a live chart as the product
// preview) -- fetched with the browser pane and inspected before writing
// this, see RESUME.md. Unlike their static marketing graphic, this one is
// a real chart backed by our own mock VNM data, reusing the same
// StockChart component the stock detail page uses (theme="dark" for the
// dark-card treatment).
export default async function Home() {
  let bars: Bar[] = [];
  let sma20: IndicatorPoint[] = [];
  try {
    const { from, to } = lastSixMonthsRange();
    const [barsRes, smaRes] = await Promise.all([
      getBars("VNM", "1D", from, to),
      getIndicator("VNM", "1D", "sma", 20, from, to),
    ]);
    bars = barsRes.data;
    sma20 = smaRes.data;
  } catch {
    // leave bars empty; the preview card below handles this.
  }

  return (
    <main className="mx-auto w-full max-w-5xl px-8 py-20 lg:py-28">
      <div className="mx-auto max-w-2xl text-center">
        <span className="inline-flex items-center rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1 text-xs font-medium text-neutral-600">
          Simulation-first · no real money
        </span>
        <h1 className="mt-6 text-4xl font-medium tracking-tight text-neutral-900 sm:text-5xl lg:text-6xl">
          Trade the past before you trade the future.
        </h1>
        <p className="mt-6 text-lg text-neutral-500">
          Charts, indicators, and paper trading for every stock on HOSE, HNX, and
          UPCOM. Practice on real market structure, with zero financial risk.
        </p>
        <div className="mt-8 flex items-center justify-center gap-3">
          <Button href="/stocks" variant="primary">
            Browse Stocks
          </Button>
          <Button href="/chart" variant="secondary">
            View Chart Demo
          </Button>
        </div>
      </div>

      <div className="mt-16 overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-950 p-6 shadow-xl lg:p-8">
        <p className="mb-4 text-sm text-neutral-400">
          VNM · Vietnam Dairy Products JSC · 6mo daily
        </p>
        {bars.length > 0 ? (
          <StockChart bars={bars} sma20={sma20} theme="dark" heightClassName="h-[320px] w-full lg:h-[420px]" />
        ) : (
          <p className="py-16 text-center text-sm text-neutral-500">
            Chart preview unavailable — the backend may be starting up.
          </p>
        )}
      </div>
    </main>
  );
}
