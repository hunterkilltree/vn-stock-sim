import Link from "next/link";
import SidebarNav from "@/components/SidebarNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import HeatmapDashboard from "@/components/HeatmapDashboard";
import { getCryptoHeatmap, getHeatmap, type HeatmapPeriod, type SectorGroup } from "@/lib/api";

export const metadata = { title: "Bản đồ nhiệt — VN Stock Sim" };

const PERIODS: { id: HeatmapPeriod; label: string }[] = [
  { id: "1D", label: "1 ngày" },
  { id: "1W", label: "1 tuần" },
  { id: "1M", label: "1 tháng" },
  { id: "3M", label: "3 tháng" },
];
const EXCHANGES = [
  { id: "ALL", label: "Tất cả" },
  { id: "HOSE", label: "HOSE" },
  { id: "HNX", label: "HNX" },
  { id: "UPCOM", label: "UPCOM" },
];

function pill(on: boolean) {
  return {
    className: "flex h-8 items-center rounded-[8px] px-3 text-[12.5px]",
    style: {
      background: on ? "var(--app-surface-3)" : "transparent",
      color: on ? "var(--app-text)" : "var(--app-text-muted)",
      fontWeight: on ? 600 : 500,
    },
  };
}

// Full-screen heatmap (phase-i.md decision 15). design/SCREENS.md lists
// it as not designed -- only Main's dashboard panel exists -- so this is
// that panel's visual language scaled up. Filters are real query params,
// fetched server-side.
export default async function HeatmapPage({ searchParams }: PageProps<"/heatmap">) {
  const sp = await searchParams;
  const market = sp.market === "crypto" ? "crypto" : "stock";
  const period = PERIODS.some((p) => p.id === sp.period) ? (sp.period as HeatmapPeriod) : "1D";
  const exchange = EXCHANGES.some((e) => e.id === sp.exchange) ? String(sp.exchange) : "ALL";

  let groups: SectorGroup[] = [];
  let error: string | null = null;
  try {
    groups = (market === "crypto" ? await getCryptoHeatmap() : await getHeatmap(period, exchange)).data;
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load heatmap";
  }

  const href = (patch: Record<string, string>) => {
    const q = new URLSearchParams({ market, period, exchange, ...patch });
    if (q.get("market") === "crypto") {
      q.delete("period");
      q.delete("exchange");
    }
    return `/heatmap?${q.toString()}`;
  };

  return (
    <div className="flex flex-1 bg-app-bg text-app-text lg:h-dvh lg:overflow-hidden">
      <SidebarNav mode={market} />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-4 px-[18px] pb-4 pt-[22px] lg:p-[24px_28px]">
        <header className="flex items-center justify-between gap-6 lg:items-end">
          <div className="flex min-w-0 flex-col gap-[5px]">
            <h1 className="m-0 font-display text-[23px] font-bold tracking-[-0.015em] lg:text-[27px]">Bản đồ nhiệt</h1>
            <span className="text-[11.5px] text-app-text-muted lg:text-[12.5px]">
              {market === "crypto"
                ? "Biến động 24 giờ theo nhóm coin · giao dịch 24/7, không có biên độ trần–sàn"
                : `Biến động ${PERIODS.find((p) => p.id === period)?.label} theo ngành · ${exchange === "ALL" ? "HOSE, HNX, UPCOM" : exchange}`}
            </span>
          </div>
          <AccountMenuButton placement="below" />
        </header>

        {/* One scrolling row on phones instead of wrapping onto three lines. */}
        <div className="-mx-[18px] flex items-center gap-3 overflow-x-auto px-[18px] lg:mx-0 lg:flex-wrap lg:overflow-visible lg:px-0">
          <div role="group" aria-label="Thị trường" className="flex shrink-0 gap-1 rounded-[10px] border border-app-border p-[3px]">
            <Link href={href({ market: "stock" })} {...pill(market === "stock")} aria-current={market === "stock" ? "page" : undefined}>
              Cổ phiếu
            </Link>
            <Link href={href({ market: "crypto" })} {...pill(market === "crypto")} aria-current={market === "crypto" ? "page" : undefined}>
              Crypto
            </Link>
          </div>
          {market === "stock" ? (
            <>
              <div role="group" aria-label="Sàn" className="flex shrink-0 gap-1 rounded-[10px] border border-app-border p-[3px]">
                {EXCHANGES.map((e) => (
                  <Link key={e.id} href={href({ exchange: e.id })} {...pill(exchange === e.id)} aria-current={exchange === e.id ? "true" : undefined}>
                    {e.label}
                  </Link>
                ))}
              </div>
              <div role="group" aria-label="Khoảng thời gian" className="flex shrink-0 gap-1 rounded-[10px] border border-app-border p-[3px]">
                {PERIODS.map((p) => (
                  <Link key={p.id} href={href({ period: p.id })} {...pill(period === p.id)} aria-current={period === p.id ? "true" : undefined}>
                    {p.label}
                  </Link>
                ))}
              </div>
            </>
          ) : (
            <span className="flex h-9 shrink-0 items-center rounded-[10px] border border-app-border px-3 text-[12.5px] text-app-text-3">24 giờ</span>
          )}
        </div>

        {error ? (
          <p className="rounded-xl border border-app-warn-border bg-app-warn-surface p-4 text-sm text-app-warn-text">
            Could not reach the API — is the backend running? ({error})
          </p>
        ) : (
          <HeatmapDashboard key={`${market}-${period}-${exchange}`} groups={groups} market={market} />
        )}
      </div>
    </div>
  );
}
