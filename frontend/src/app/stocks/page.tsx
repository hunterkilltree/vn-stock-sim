import Link from "next/link";
import SidebarNav from "@/components/SidebarNav";
import { searchSymbols, type Symbol } from "@/lib/api";

export const metadata = { title: "Stocks — VN Stock Sim" };

// Server Component: fetches the V1 stock browser list from the Go backend
// (GET /api/v1/symbols). See RESUME.md — backed by mock fixture data until
// a licensed market-data provider is wired up.
export default async function StocksPage() {
  let symbols: Symbol[] = [];
  let error: string | null = null;
  try {
    ({ data: symbols } = await searchSymbols());
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load symbols";
  }

  return (
    <div className="flex flex-1">
      <SidebarNav />
      <main className="mx-auto w-full max-w-5xl p-8 lg:p-12">
      <h1 className="mb-1 text-2xl font-medium tracking-tight text-neutral-900 lg:text-3xl">
        Vietnamese Stocks
      </h1>
      <p className="mb-6 text-sm text-neutral-500">HOSE, HNX, and UPCOM — mock fixture data for V1.</p>
      {error && (
        <p className="mb-6 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          Could not reach the API at {process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"}
          {" — is the backend running (`go run ./cmd/api` in backend/)? "}
          ({error})
        </p>
      )}
      <div className="overflow-hidden rounded-2xl border border-neutral-200">
        <table className="w-full border-collapse text-left text-sm lg:text-base">
          <thead>
            <tr className="border-b border-neutral-200 bg-neutral-50 text-neutral-500">
              <th className="px-4 py-3 font-medium">Symbol</th>
              <th className="px-4 py-3 font-medium">Company</th>
              <th className="px-4 py-3 font-medium">Exchange</th>
              <th className="px-4 py-3 font-medium">Sector</th>
            </tr>
          </thead>
          <tbody>
            {symbols.map((s) => (
              <tr key={s.symbol} className="border-b border-neutral-100 last:border-0 hover:bg-neutral-50">
                <td className="px-4 py-3">
                  <Link className="font-medium text-neutral-900 hover:underline" href={`/stocks/${s.symbol}`}>
                    {s.symbol}
                  </Link>
                </td>
                <td className="px-4 py-3 text-neutral-700">{s.companyName}</td>
                <td className="px-4 py-3 text-neutral-500">{s.exchange}</td>
                <td className="px-4 py-3 text-neutral-500">{s.sector}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      </main>
    </div>
  );
}
