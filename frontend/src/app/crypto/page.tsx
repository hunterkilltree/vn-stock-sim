import Link from "next/link";
import SidebarNav from "@/components/SidebarNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import SectorHeatmap from "@/components/SectorHeatmap";
import CryptoStatCard from "@/components/CryptoStatCard";
import CryptoMoversTable from "@/components/CryptoMoversTable";
import CryptoSearch from "@/components/CryptoSearch";
import OpenCryptoWalletButton from "@/components/OpenCryptoWalletButton";
import WillBadge from "@/components/WillBadge";
import {
  getCryptoHeatmap,
  getCryptoMovers,
  getCryptoOverview,
  getCryptoPairs,
  getPortfolioPositionsByID,
  getPortfolioSummaryByID,
  type CryptoOverview,
  type CryptoPairQuote,
  type Portfolio,
  type PortfolioSummary,
  type Position,
  type SectorGroup,
} from "@/lib/api";
import { formatAmount, formatCompact, formatCryptoPrice, formatVN, signVN, tone } from "@/lib/format";
import { getActivePortfolio, getSessionToken, getSessionUser } from "@/lib/session";

export const metadata = { title: "Thị trường crypto — VN Stock Sim" };

function nowLabel(): string {
  const now = new Date();
  const opts = { timeZone: "Asia/Ho_Chi_Minh" } as const;
  return `${now.toLocaleDateString("vi-VN", { ...opts, day: "2-digit", month: "2-digit", year: "numeric" })} · ${now.toLocaleTimeString("vi-VN", { ...opts, hour: "2-digit", minute: "2-digit" })}`;
}

// design/screens/Crypto-Main.dc.html (phase-i.md). Market data comes from
// backend/internal/crypto (Binance, with a mock fallback); the wallet is
// the user's active crypto portfolio (decision 9).
export default async function CryptoMainPage() {
  let overview: CryptoOverview | null = null;
  let heat: SectorGroup[] = [];
  let up: CryptoPairQuote[] = [];
  let down: CryptoPairQuote[] = [];
  let pairs: CryptoPairQuote[] = [];
  let error: string | null = null;
  try {
    const [o, h, u, d, p] = await Promise.all([getCryptoOverview(), getCryptoHeatmap(), getCryptoMovers("up", 5), getCryptoMovers("down", 5), getCryptoPairs()]);
    overview = o;
    heat = h.data;
    up = u.data;
    down = d.data;
    pairs = p.data;
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load crypto data";
  }

  const user = await getSessionUser();
  let wallet: Portfolio | null = null;
  let summary: PortfolioSummary | null = null;
  let positions: Position[] = [];
  if (user) {
    try {
      const token = await getSessionToken();
      if (token) {
        wallet = (await getActivePortfolio(token, "crypto")).active;
        if (wallet) {
          [summary, positions] = await Promise.all([
            getPortfolioSummaryByID(wallet.id, token),
            getPortfolioPositionsByID(wallet.id, token).then((r) => r.data),
          ]);
        }
      }
    } catch {
      // wallet card falls back to its empty state
    }
  }
  const source = overview?.btc.source === "binance" ? "Binance" : "dữ liệu mô phỏng";
  const pnlPct = wallet && summary ? ((summary.totalEquity - wallet.startingCapital) / wallet.startingCapital) * 100 : 0;

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <SidebarNav mode="crypto" cashBalance={summary?.cashBalance} />
      <div className="flex min-w-0 flex-1 flex-col gap-5 p-[24px_28px]">
        <header className="flex items-end justify-between gap-6">
          <div className="flex flex-col gap-[5px]">
            <h1 className="m-0 font-display text-[27px] font-bold tracking-[-0.015em]">Thị trường crypto</h1>
            <div className="flex items-center gap-2 text-[12.5px] text-app-text-3">
              <span className="h-[7px] w-[7px] rounded-full bg-price-up" />
              <span>{nowLabel()} · giao dịch 24/7, không có phiên và không có biên độ trần–sàn · {source}</span>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <CryptoSearch pairs={pairs.map((p) => ({ symbol: p.symbol, base: p.base, name: p.name }))} />
            <Link href="/crypto/replay" className="flex h-11 items-center gap-2 rounded-[11px] bg-app-accent px-4 text-[13.5px] font-semibold text-app-accent-ink">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
              </svg>
              <span>Vào Replay</span>
            </Link>
            <AccountMenuButton placement="below" />
          </div>
        </header>

        {error && (
          <p className="rounded-xl border border-app-warn-border bg-app-warn-surface p-4 text-sm text-app-warn-text">
            Could not reach the API — is the backend running? ({error})
          </p>
        )}

        {overview && (
          <div className="grid grid-cols-4 gap-4">
            <CryptoStatCard
              name="BTC / USDT"
              value={formatCryptoPrice(overview.btc.lastPrice)}
              changePercent={overview.btc.changePercent}
              changeText={`${signVN(overview.btc.changePercent, 2)}% trong 24 giờ`}
              foot={`KL 24h ${formatCompact(overview.btc.quoteVolume24h, "USDT")}`}
              sparkline={overview.btcSparkline}
            />
            <CryptoStatCard
              name="ETH / USDT"
              value={formatCryptoPrice(overview.eth.lastPrice)}
              changePercent={overview.eth.changePercent}
              changeText={`${signVN(overview.eth.changePercent, 2)}% trong 24 giờ`}
              foot={`KL 24h ${formatCompact(overview.eth.quoteVolume24h, "USDT")}`}
              sparkline={overview.ethSparkline}
            />
            <CryptoStatCard
              name="Vốn hoá"
              value={formatCompact(overview.totalMarketCap, "USD")}
              changePercent={overview.totalMarketCapChangePercent}
              changeText={`${signVN(overview.totalMarketCapChangePercent, 2)}% trong 24 giờ`}
              foot={`Tổng ${overview.pairCount} coin của ứng dụng`}
            />
            <CryptoStatCard
              name="Thống trị BTC"
              value={`${formatVN(overview.btcDominancePercent, 2)}%`}
              changePercent={overview.btc.changePercent - overview.totalMarketCapChangePercent}
              changeText="so với tổng vốn hoá"
              foot={`ETH ${formatVN(overview.ethDominancePercent, 1)}% · phần còn lại ${formatVN(100 - overview.btcDominancePercent - overview.ethDominancePercent, 1)}% (trong ${overview.pairCount} coin)`}
            />
          </div>
        )}

        <div className="flex min-h-0 flex-1 gap-5">
          <div className="flex min-w-0 flex-1 flex-col gap-5">
            <SectorHeatmap sectors={heat} market="crypto" />
            <section className="flex shrink-0 gap-6 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
              <CryptoMoversTable title="Tăng mạnh nhất 24h" rows={up} />
              <CryptoMoversTable title="Giảm mạnh nhất 24h" rows={down} />
            </section>
          </div>

          <div className="flex w-[372px] shrink-0 flex-col gap-4">
            {!user ? (
              <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
                <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Ví giấy</span>
                <p className="m-0 text-[13px] text-app-text-3">Đăng nhập để mở ví crypto giấy 10.000 USDT. Bạn vẫn có thể xem thị trường mà không cần tài khoản.</p>
                <Link href="/register" className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink">
                  Tạo tài khoản miễn phí
                </Link>
              </section>
            ) : !wallet || !summary ? (
              <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
                <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Ví giấy</span>
                <p className="m-0 text-[13px] text-app-text-3">Bạn chưa có ví crypto. Ví giấy tách biệt với danh mục cổ phiếu và bắt đầu với 10.000 USDT ảo.</p>
                <OpenCryptoWalletButton />
              </section>
            ) : (
              <>
                <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]" aria-label="Ví giấy">
                  <div className="flex items-center justify-between">
                    <span className="text-[11px] uppercase tracking-[0.09em] text-app-text-muted">Ví giấy · {wallet.name}</span>
                    <span className="text-[11px] text-app-text-muted">
                      Từ {new Date(wallet.createdAt).toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" })}
                    </span>
                  </div>
                  <span className="font-plex-mono text-[26px] font-semibold">{formatVN(summary.totalEquity, 2)} USDT</span>
                  <div className="flex items-center gap-2 text-[12px]">
                    <span className="font-plex-mono font-semibold" style={{ color: tone(pnlPct) }}>{signVN(pnlPct, 2)}%</span>
                    <span className="text-app-text-muted">so với {formatVN(wallet.startingCapital, 0)} USDT ban đầu</span>
                  </div>
                  <div className="grid grid-cols-3 gap-2 border-t border-app-hairline pt-3 text-[11px]">
                    <div className="flex flex-col gap-1">
                      <span className="text-app-text-muted">USDT rảnh</span>
                      <span className="font-plex-mono text-[13px] text-app-text">{formatVN(summary.cashBalance, 2)}</span>
                    </div>
                    <div className="flex flex-col gap-1">
                      <span className="text-app-text-muted">Đang nắm giữ</span>
                      <span className="font-plex-mono text-[13px] text-app-text">
                        {formatVN(summary.totalEquity > 0 ? (summary.marketValue / summary.totalEquity) * 100 : 0, 1)}%
                      </span>
                    </div>
                    <div className="flex flex-col gap-1">
                      <span className="text-app-text-muted">Lãi/lỗ mở</span>
                      <span className="font-plex-mono text-[13px]" style={{ color: tone(summary.unrealizedPnl) }}>{signVN(summary.unrealizedPnl, 2)}</span>
                    </div>
                  </div>
                </section>
                <section className="flex flex-col gap-2 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
                  <div className="flex items-center justify-between">
                    <h2 className="m-0 text-[14px] font-semibold">Đang nắm giữ</h2>
                    <span className="text-[11px] text-app-text-muted">Giao ngay · không đòn bẩy</span>
                  </div>
                  {positions.length === 0 ? (
                    <p className="m-0 text-[12.5px] text-app-text-muted">Chưa có coin nào. Mở một cặp để đặt lệnh giấy.</p>
                  ) : (
                    <table className="w-full border-collapse text-[12.5px]">
                      <thead>
                        <tr className="border-b border-app-hairline text-[10.5px] uppercase tracking-[0.06em] text-app-text-muted">
                          <th scope="col" className="py-[6px] text-left font-medium">Coin</th>
                          <th scope="col" className="py-[6px] text-right font-medium">Số lượng</th>
                          <th scope="col" className="py-[6px] text-right font-medium">Giá vốn</th>
                          <th scope="col" className="py-[6px] text-right font-medium">Lãi/lỗ</th>
                        </tr>
                      </thead>
                      <tbody>
                        {positions.map((p) => {
                          const pct = p.avgCost > 0 ? ((p.lastPrice - p.avgCost) / p.avgCost) * 100 : 0;
                          return (
                            <tr key={p.symbol}>
                              <td className="py-2">
                                <Link href={`/crypto/${p.symbol}`} className="font-plex-mono font-semibold text-app-text">{p.symbol.replace(/USDT$/, "")}</Link>
                              </td>
                              <td className="py-2 text-right font-plex-mono">{formatAmount(p.quantity, 6)}</td>
                              <td className="py-2 text-right font-plex-mono">{formatCryptoPrice(p.avgCost)}</td>
                              <td className="py-2 text-right font-plex-mono font-semibold" style={{ color: tone(pct) }}>
                                {signVN(pct, 1)}% · {signVN(p.unrealizedPnl, 2)}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  )}
                </section>
              </>
            )}

            <section className="flex flex-col gap-3 rounded-2xl border border-app-accent-border bg-app-accent-surface p-[18px_20px]">
              <span className="text-[11px] uppercase tracking-[0.09em] text-app-accent">Chế độ Replay</span>
              <p className="m-0 font-display text-[18px] font-semibold leading-[1.35]">&ldquo;Giao dịch quá khứ trước khi giao dịch tương lai.&rdquo;</p>
              <p className="m-0 text-[12.5px] leading-[1.5] text-app-text-3">
                Chạy lại cú sập tháng 5/2021 theo nến 1 giờ — thị trường không nghỉ, và bạn cũng không thấy nến kế tiếp.
              </p>
              <Link href="/crypto/replay" className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink">
                Bắt đầu phiên Replay
              </Link>
            </section>
            <section title="Quant hiện chỉ lọc cổ phiếu" className="flex items-center gap-3 rounded-2xl border border-app-border bg-app-surface p-[14px_16px] opacity-70">
              <div className="flex min-w-0 flex-1 flex-col gap-[1px]">
                <span className="flex items-center gap-2 text-[13px] font-semibold">
                  Hỏi Quant <WillBadge />
                </span>
                <span className="truncate text-[11.5px] text-app-text-muted">&ldquo;Coin nào có RSI 4h dưới 30 và KL tăng?&rdquo;</span>
              </div>
            </section>
          </div>
        </div>
      </div>
    </div>
  );
}
