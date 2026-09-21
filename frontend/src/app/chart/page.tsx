import TradingViewWidget from "@/components/TradingViewWidget";

export const metadata = { title: "Chart — VN Stock Sim" };

// Step 1 of the TradingView integration plan (see RESUME.md): confirm a
// TradingView chart renders in the app using their public embed widget,
// defaulting to BTC. Not wired to our own /api/v1/market/bars data yet --
// that requires the separately-gated self-hosted Charting Library and a
// custom Datafeed (charting-library-integration.md).
export default function ChartPage() {
  return (
    <main className="mx-auto max-w-5xl p-8">
      <h1 className="mb-4 text-2xl font-semibold">TradingView Chart (proof of concept)</h1>
      <p className="mb-6 text-sm text-gray-500">
        BTC/USD via TradingView&apos;s public Advanced Chart widget. This proves the chart
        renders end-to-end before wiring the licensed Charting Library + custom Datafeed
        against our own VN symbol data — see RESUME.md.
      </p>
      <TradingViewWidget symbol="BITSTAMP:BTCUSD" />
    </main>
  );
}
