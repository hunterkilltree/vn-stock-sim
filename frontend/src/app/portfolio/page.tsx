import Link from "next/link";
import SidebarNav from "@/components/SidebarNav";
import PortfolioTabs from "@/components/PortfolioTabs";
import {
  getPortfolioSummaryByID,
  getPortfolioPositionsByID,
  getEquityHistory,
  getAllocation,
  getPortfolioStats,
  getOrders,
  recentlyProcessed,
  type Portfolio,
  type PortfolioSummary,
  type Position,
  type EquityPoint,
  type Allocation,
  type PortfolioStats,
  type Order,
} from "@/lib/api";
import { getActivePortfolio, getSessionToken, getSessionUser } from "@/lib/session";
import AccountMenuButton from "@/components/AccountMenuButton";

export const metadata = { title: "Giao dịch giấy — VN Stock Sim" };

// Rebuilt in Phase E against design/screens/Portfolio.dc.html (see
// phase-e.md) -- KPIs, equity curve, positions, sector allocation,
// pending orders, and journal all come from real backend computation
// (backend/internal/portfolio's new equity-history/allocation/stats
// endpoints, backend/internal/order's existing list/cancel), not
// hardcoded sample numbers.
export default async function PortfolioPage() {
  const user = await getSessionUser();
  const token = user ? await getSessionToken() : null;

  let portfolio: Portfolio | null = null;
  let summary: PortfolioSummary | null = null;
  let positions: Position[] = [];
  let equityHistory: EquityPoint[] = [];
  let allocation: Allocation[] = [];
  let stats: PortfolioStats | null = null;
  let pendingOrders: Order[] = [];
  let recentOrders: Order[] = [];
  let error: string | null = null;

  if (token) {
    try {
      portfolio = (await getActivePortfolio(token)).active;
      if (portfolio) {
        const [summaryRes, positionsRes, equityRes, allocRes, statsRes, ordersRes] = await Promise.all([
          getPortfolioSummaryByID(portfolio.id, token),
          getPortfolioPositionsByID(portfolio.id, token),
          getEquityHistory(portfolio.id, token),
          getAllocation(portfolio.id, token),
          getPortfolioStats(portfolio.id, token),
          getOrders(token),
        ]);
        summary = summaryRes;
        positions = positionsRes.data;
        equityHistory = equityRes.data;
        allocation = allocRes.data;
        stats = statsRes;
        pendingOrders = ordersRes.data.filter((o) => o.status === "queued" && o.portfolioId === portfolio!.id);
        recentOrders = recentlyProcessed(ordersRes.data, portfolio.id);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load portfolio data";
    }
  }

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <SidebarNav cashBalance={summary?.cashBalance} />

      <div className="flex min-w-0 flex-1 flex-col gap-[14px] px-[18px] pb-4 pt-[22px] lg:gap-[18px] lg:p-[24px_28px]">
        <header className="flex items-center justify-between gap-6">
          <div className="flex min-w-0 flex-col gap-[5px]">
            <h1 className="m-0 font-display text-[23px] font-bold tracking-[-0.015em] lg:text-[27px]">
              <span className="lg:hidden">Danh mục giấy</span>
              <span className="hidden lg:inline">Giao dịch giấy</span>
            </h1>
            {portfolio && (
              <span className="truncate text-[11.5px] text-app-text-muted lg:text-[12.5px]">
                {portfolio.name} · {positions.length} mã
              </span>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-[10px]">
            <button
              type="button"
              disabled
              title="Sắp ra mắt"
              className="hidden h-11 cursor-not-allowed lg:block rounded-[11px] border border-app-border bg-app-surface px-[15px] text-[13px] font-medium text-app-text-muted"
            >
              Nạp lại tài khoản ảo
            </button>
            <Link
              href="/stocks"
              aria-label="Đặt lệnh mới"
              className="flex h-11 w-11 items-center justify-center gap-2 rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink lg:w-auto lg:px-4"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                <path d="M12 5v14M5 12h14" />
              </svg>
              <span className="hidden lg:inline">Đặt lệnh mới</span>
            </Link>
            {/* Not in Portfolio.dc.html -- phase-g.md decision 9. */}
            <AccountMenuButton placement="below" />
          </div>
        </header>

        {error && (
          <p className="rounded-xl border border-app-warn-border bg-app-warn-surface p-4 text-sm text-app-warn-text">
            Could not reach the API — is the backend running? ({error})
          </p>
        )}

        {!token || !portfolio || !stats ? (
          <section className="flex flex-1 flex-col items-center justify-center gap-3 rounded-2xl border border-app-border bg-app-surface p-8 text-center">
            <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Tài khoản giấy</span>
            <p className="m-0 max-w-md text-[13.5px] text-app-text-3">
              Đăng nhập để xem số dư ảo, vị thế, đường giá trị tài khoản, và lịch sử giao dịch giấy của bạn.
            </p>
            <Link
              href="/register"
              className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent px-6 text-[13.5px] font-semibold text-app-accent-ink"
            >
              Tạo tài khoản miễn phí
            </Link>
          </section>
        ) : (
          <PortfolioTabs
            stats={stats}
            equityHistory={equityHistory}
            positions={positions}
            allocation={allocation}
            pendingOrders={pendingOrders}
            recentOrders={recentOrders}
            startingCapital={portfolio.startingCapital}
          />
        )}
      </div>
    </div>
  );
}
