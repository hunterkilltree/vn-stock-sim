import Link from "next/link";
import SidebarNav from "@/components/SidebarNav";
import QuantChat from "@/components/QuantChat";
import QuantHistoryList from "@/components/QuantHistoryList";
import { getSessionUser } from "@/lib/session";

export const metadata = { title: "Trợ lý Quant — VN Stock Sim" };

// design/screens/Quant.dc.html, built in Phase H (phase-h.md). Signed-in
// only: Quant reads the user's watchlist/positions and every question is
// billed to the user's own model key (decision 14).
export default async function QuantPage({ searchParams }: PageProps<"/quant">) {
  const user = await getSessionUser();
  const q = (await searchParams).q;
  const initialQuestion = typeof q === "string" ? q.slice(0, 500) : "";

  return (
    // Viewport-high from lg, like the design's fixed artboard: the thread
    // scrolls inside, the question box stays in view. Phones scroll the
    // page instead, with the question box pinned above the tab bar
    // (phase-j.md decision 10).
    <div className="flex flex-1 bg-app-bg text-app-text lg:h-dvh lg:overflow-hidden">
      <SidebarNav>{user && <QuantHistoryList />}</SidebarNav>
      {user ? (
        <QuantChat initialQuestion={initialQuestion} />
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
          <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Trợ lý Quant</span>
          <p className="m-0 max-w-md text-[13.5px] text-app-text-3">
            Đăng nhập để lọc cổ phiếu và dựng chiến lược bằng tiếng Việt, với khoá AI của riêng bạn.
          </p>
          <Link
            href="/login"
            className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent px-6 text-[13.5px] font-semibold text-app-accent-ink"
          >
            Đăng nhập
          </Link>
        </div>
      )}
    </div>
  );
}
