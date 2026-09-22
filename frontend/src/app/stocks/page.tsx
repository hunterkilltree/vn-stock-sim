import SidebarNav from "@/components/SidebarNav";
import IndexCard from "@/components/IndexCard";
import SectorHeatmap from "@/components/SectorHeatmap";
import MoversTable from "@/components/MoversTable";
import PaperAccountCard from "@/components/PaperAccountCard";
import OpenPositionsCard from "@/components/OpenPositionsCard";
import ReplayPromoCard from "@/components/ReplayPromoCard";
import QuantPromptCard from "@/components/QuantPromptCard";
import Link from "next/link";
import {
  getIndices,
  getHeatmap,
  getMovers,
  getPortfolioSummary,
  getPortfolioPositions,
  getPortfolios,
  type IndexSnapshot,
  type SectorGroup,
  type TickerChange,
  type PortfolioSummary,
  type Position,
  type Portfolio,
} from "@/lib/api";
import { getSessionToken, getSessionUser } from "@/lib/session";

export const metadata = { title: "Tổng quan thị trường — VN Stock Sim" };

function formatDateVN(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" });
}

// VN_TIME_ZONE is IANA "Asia/Ho_Chi_Minh" (ICT, UTC+7) -- the Vietnamese
// exchanges' own timezone, not the server process's local one. Without
// an explicit timeZone, Date#toLocale*String falls back to the
// container's TZ (UTC in this repo's Docker image), which would render
// e.g. "07:45" and label it as if it were the VN session's "14:45" --
// wrong by exactly the UTC+7 offset. This is the same class of bug as
// the 2026-09-21 stock-price/chart UTC-freshness fix (see RESUME.md):
// a real timestamp rendered against the wrong reference, not a design
// choice.
const VN_TIME_ZONE = "Asia/Ho_Chi_Minh";

function nowSessionLabel(): string {
  const now = new Date();
  const date = now.toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", timeZone: VN_TIME_ZONE });
  const time = now.toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit", timeZone: VN_TIME_ZONE });
  return `Phiên ${date} · ${time} · Khớp lệnh liên tục`;
}

// Rebuilt in Phase C against design/screens/Main.dc.html (see
// phase-c.md) -- structure/spacing/colors follow that spec exactly;
// data comes entirely from the real Phase B backend endpoints, not the
// design's sample fixtures.
export default async function MarketOverviewPage() {
  let indices: IndexSnapshot[] = [];
  let sectors: SectorGroup[] = [];
  let gainers: TickerChange[] = [];
  let losers: TickerChange[] = [];
  let error: string | null = null;
  try {
    const [idxRes, heatRes, upRes, downRes] = await Promise.all([
      getIndices(),
      getHeatmap(),
      getMovers("up", 5),
      getMovers("down", 5),
    ]);
    indices = idxRes.data;
    sectors = heatRes.data;
    gainers = upRes.data;
    losers = downRes.data;
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load market data";
  }

  const user = await getSessionUser();
  let summary: PortfolioSummary | null = null;
  let positions: Position[] = [];
  let defaultPortfolio: Portfolio | null = null;
  if (user) {
    try {
      const token = await getSessionToken();
      if (token) {
        const [summaryRes, positionsRes, portfoliosRes] = await Promise.all([
          getPortfolioSummary(token),
          getPortfolioPositions(token),
          getPortfolios(token),
        ]);
        summary = summaryRes;
        positions = positionsRes.data;
        defaultPortfolio = portfoliosRes.data[0] ?? null;
      }
    } catch {
      // leave summary/positions/defaultPortfolio null; the right column
      // degrades to the sign-up-style prompt below rather than crashing.
    }
  }

  const initials = user ? user.displayName.slice(0, 2).toUpperCase() : "?";

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <SidebarNav cashBalance={summary?.cashBalance} />

      <div className="flex min-w-0 flex-1 flex-col gap-5 p-[24px_28px]">
        <header className="flex items-end justify-between gap-6">
          <div className="flex flex-col gap-[5px]">
            <h1 className="m-0 font-display text-[27px] font-bold tracking-[-0.015em]">Tổng quan thị trường</h1>
            <div className="flex items-center gap-2 text-[12.5px] text-app-text-3">
              <span className="h-[7px] w-[7px] rounded-full bg-price-up" />
              <span>{nowSessionLabel()}</span>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex h-11 items-center gap-2 rounded-[11px] border border-app-border bg-app-surface px-[14px]">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#8A867E" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
                <circle cx="11" cy="11" r="7" />
                <path d="M16.5 16.5L21 21" />
              </svg>
              <label htmlFor="q" className="sr-only">Tìm mã cổ phiếu</label>
              <input
                id="q"
                type="search"
                placeholder="Tìm mã: FPT, VCB, BTC…"
                className="w-[200px] border-0 bg-transparent text-[13px] text-app-text outline-none placeholder:text-app-text-muted"
              />
            </div>
            <button
              type="button"
              disabled
              title="Sắp ra mắt"
              className="flex h-11 cursor-not-allowed items-center gap-2 rounded-[11px] bg-app-accent px-4 text-[13.5px] font-semibold text-app-accent-ink opacity-70"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
              </svg>
              <span>Vào Replay</span>
            </button>
            <button
              type="button"
              aria-label="Tài khoản của bạn"
              disabled
              title="Menu tài khoản — sắp ra mắt"
              className="flex h-11 w-11 cursor-not-allowed items-center justify-center rounded-[11px] border border-app-border bg-app-surface-2 text-[13px] font-semibold text-app-text"
            >
              {initials}
            </button>
          </div>
        </header>

        {error && (
          <p className="rounded-xl border border-app-warn-border bg-app-warn-surface p-4 text-sm text-app-warn-text">
            Could not reach the API — is the backend running? ({error})
          </p>
        )}

        <div className="grid grid-cols-4 gap-4">
          {indices.map((ix) => (
            <IndexCard key={ix.name} index={ix} />
          ))}
        </div>

        <div className="flex min-h-0 flex-1 gap-5">
          <div className="flex min-w-0 flex-1 flex-col gap-5">
            <SectorHeatmap sectors={sectors} />
            <section className="flex h-[226px] shrink-0 gap-6 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
              <MoversTable title="Tăng mạnh nhất" rows={gainers} />
              <MoversTable title="Giảm mạnh nhất" rows={losers} />
            </section>
          </div>

          <div className="flex w-[372px] shrink-0 flex-col gap-4">
            {summary && defaultPortfolio ? (
              <>
                <PaperAccountCard
                  summary={summary}
                  startingCapital={defaultPortfolio.startingCapital}
                  sinceDate={formatDateVN(defaultPortfolio.createdAt)}
                />
                <OpenPositionsCard positions={positions} />
              </>
            ) : (
              <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
                <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Tài khoản giấy</span>
                <p className="m-0 text-[13px] text-app-text-3">
                  Đăng nhập để xem số dư ảo, vị thế, và bắt đầu giao dịch giấy. Bạn vẫn có thể xem thị trường và biểu đồ mà không cần tài khoản.
                </p>
                <Link
                  href="/register"
                  className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink"
                >
                  Tạo tài khoản miễn phí
                </Link>
              </section>
            )}

            <ReplayPromoCard />
            <QuantPromptCard />
          </div>
        </div>
      </div>
    </div>
  );
}
