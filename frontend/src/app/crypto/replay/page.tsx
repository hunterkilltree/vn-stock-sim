import Link from "next/link";
import RailNav from "@/components/RailNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import ReplaySession from "@/components/ReplaySession";
import { getSessionUser } from "@/lib/session";

export const metadata = { title: "Replay crypto — VN Stock Sim" };

// design/screens/Crypto-Replay.dc.html (phase-i.md): the same Replay
// session as /replay, run on hourly bars with its own 10.000 USDT paper
// account. "Replay cặp này" on a pair's page links here with ?symbol=.
export default async function CryptoReplayPage({ searchParams }: PageProps<"/crypto/replay">) {
  const user = await getSessionUser();
  const raw = (await searchParams).symbol;
  const initialSymbol = typeof raw === "string" && /^[A-Za-z0-9]{2,12}$/.test(raw) ? raw.toUpperCase() : "BTCUSDT";

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <RailNav mode="crypto" account={<AccountMenuButton placement="right" />} />
      <div className="flex min-w-0 flex-1 flex-col px-[18px] pt-[22px] lg:p-[20px_24px]">
        {user ? (
          <ReplaySession initialSymbol={initialSymbol} market="crypto" />
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
            <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Replay crypto</span>
            <p className="m-0 max-w-md text-[13.5px] text-app-text-3">
              Đăng nhập để chơi lại thị trường crypto từng nến 1 giờ -- mỗi phiên có ví giấy 10.000 USDT riêng, không ảnh hưởng đến ví crypto chính của bạn.
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
