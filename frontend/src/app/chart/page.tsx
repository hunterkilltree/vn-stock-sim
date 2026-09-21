import TradingViewWidget from "@/components/TradingViewWidget";

export const metadata = { title: "Chart — VN Stock Sim" };

// Step 1 of the TradingView integration plan (see RESUME.md): confirm a
// TradingView chart renders in the app using their public embed widget,
// defaulting to BTC. Not wired to our own /api/v1/market/bars data yet --
// that requires the separately-gated self-hosted Charting Library and a
// custom Datafeed (charting-library-integration.md).
export default function ChartPage() {
  return (
    <main className="mx-auto w-full max-w-5xl p-8 lg:p-12">
      <h1 className="mb-1 text-2xl font-medium tracking-tight text-neutral-900 lg:text-3xl">
        TradingView Chart
      </h1>
      <p className="mb-6 text-sm text-neutral-500">
        BTC/USD via TradingView&apos;s public Advanced Chart widget — proof of concept. This
        proves the chart renders end-to-end before wiring the licensed Charting Library +
        custom Datafeed against our own VN symbol data — see RESUME.md.
      </p>
      <div className="overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-950 p-6">
        <TradingViewWidget symbol="BITSTAMP:BTCUSD" theme="dark" />
      </div>
    </main>
  );
}
