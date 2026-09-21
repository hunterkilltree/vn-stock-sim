import Link from "next/link";
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
    <main className="mx-auto max-w-3xl p-8">
      <h1 className="mb-4 text-2xl font-semibold">Vietnamese Stocks</h1>
      {error && (
        <p className="mb-4 rounded bg-red-50 p-3 text-sm text-red-700">
          Could not reach the API at {process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"}
          {" — is the backend running (`go run ./cmd/api` in backend/)? "}
          ({error})
        </p>
      )}
      <table className="w-full border-collapse text-left text-sm">
        <thead>
          <tr className="border-b font-medium">
            <th className="py-2">Symbol</th>
            <th className="py-2">Company</th>
            <th className="py-2">Exchange</th>
            <th className="py-2">Sector</th>
          </tr>
        </thead>
        <tbody>
          {symbols.map((s) => (
            <tr key={s.symbol} className="border-b last:border-0">
              <td className="py-2">
                <Link className="font-medium text-blue-600 hover:underline" href={`/stocks/${s.symbol}`}>
                  {s.symbol}
                </Link>
              </td>
              <td className="py-2">{s.companyName}</td>
              <td className="py-2">{s.exchange}</td>
              <td className="py-2">{s.sector}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </main>
  );
}
