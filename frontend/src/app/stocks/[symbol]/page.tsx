import { notFound } from "next/navigation";
import { getSymbolDetail } from "@/lib/api";

type Props = { params: Promise<{ symbol: string }> };

export default async function StockDetailPage({ params }: Props) {
  const { symbol } = await params;
  let detail;
  try {
    detail = await getSymbolDetail(symbol);
  } catch {
    notFound();
  }

  const changeColor = detail.change >= 0 ? "text-green-600" : "text-red-600";

  return (
    <main className="mx-auto max-w-2xl p-8">
      <p className="text-sm text-gray-500">{detail.exchange} · {detail.sector}</p>
      <h1 className="mb-1 text-3xl font-semibold">{detail.symbol}</h1>
      <p className="mb-6 text-gray-600">{detail.companyName}</p>

      <div className="mb-8 flex items-baseline gap-3">
        <span className="text-4xl font-semibold">{detail.lastPrice.toLocaleString("vi-VN")}</span>
        <span className={`text-lg font-medium ${changeColor}`}>
          {detail.change >= 0 ? "+" : ""}
          {detail.change.toLocaleString("vi-VN")} ({detail.changePercent.toFixed(2)}%)
        </span>
      </div>

      <dl className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <dt className="text-gray-500">Market Cap</dt>
          <dd className="font-medium">{detail.marketCap.toLocaleString("vi-VN")} VND</dd>
        </div>
        <div>
          <dt className="text-gray-500">P/E</dt>
          <dd className="font-medium">{detail.peRatio}</dd>
        </div>
        <div>
          <dt className="text-gray-500">P/B</dt>
          <dd className="font-medium">{detail.pbRatio}</dd>
        </div>
        <div>
          <dt className="text-gray-500">EPS</dt>
          <dd className="font-medium">{detail.eps.toLocaleString("vi-VN")}</dd>
        </div>
        <div>
          <dt className="text-gray-500">Dividend Yield</dt>
          <dd className="font-medium">{detail.dividendYield}%</dd>
        </div>
      </dl>

      <p className="mt-8 text-sm text-gray-400">
        Candlestick chart + indicators not built yet — see RESUME.md plan step 2.
      </p>
    </main>
  );
}
