import Link from "next/link";
import RailNav from "@/components/RailNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import ReplaySession from "@/components/ReplaySession";
import { getSessionUser } from "@/lib/session";

export const metadata = { title: "Chế độ Replay — VN Stock Sim" };

// Rebuilt in Phase F against design/screens/Replay.dc.html (see
// phase-f.md). RailNav shell (Replay is a chart-space-first screen, same
// nav choice as Detail/Settings/Quant-Chart). The whole session is
// interactive from the moment it starts (advance/orders/keyboard/auto-
// play), so this page is a thin Server Component wrapper that only
// handles the guest gate -- the actual work is ReplaySession.tsx, a
// Client Component.
export default async function ReplayPage({ searchParams }: PageProps<"/replay">) {
  const user = await getSessionUser();
  // "Chạy thử bằng Replay" from Quant links here with ?symbol=.
  const raw = (await searchParams).symbol;
  const initialSymbol = typeof raw === "string" && /^[A-Za-z0-9]{1,10}$/.test(raw) ? raw.toUpperCase() : "HPG";

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <RailNav account={<AccountMenuButton placement="right" />} />
      <div className="flex min-w-0 flex-1 flex-col p-[20px_24px]">
        {user ? (
          <ReplaySession initialSymbol={initialSymbol} />
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
            <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Chế độ Replay</span>
            <p className="m-0 max-w-md text-[13.5px] text-app-text-3">
              Đăng nhập để chơi lại lịch sử từng nến một -- mỗi phiên có tài khoản giấy riêng, không ảnh hưởng đến tài khoản giao dịch chính của bạn.
            </p>
            <Link
              href="/register"
              className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent px-6 text-[13.5px] font-semibold text-app-accent-ink"
            >
              Tạo tài khoản miễn phí
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
