import Link from "next/link";
import SidebarNav from "@/components/SidebarNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import BacktestForm, { type BacktestDefaults } from "@/components/BacktestForm";
import BacktestEquityChart from "@/components/BacktestEquityChart";
import { getBacktest, getBacktests, type BacktestRun, type BacktestRule } from "@/lib/api";
import { formatThousandsVN, formatVN, signVN, tone } from "@/lib/format";
import { getSessionToken, getSessionUser } from "@/lib/session";

export const metadata = { title: "Kiểm thử lịch sử — VN Stock Sim" };

const RULE_LABEL: Record<BacktestRule, string> = { ema_crossover: "EMA cắt nhau", rsi_reversion: "RSI hồi phục" };

function ruleSummary(run: BacktestRun): string {
  const p = run.params ?? {};
  if (run.ruleType === "ema_crossover") return `EMA ${p.fast ?? 20}/${p.slow ?? 50}`;
  const parts = [`RSI ${p.period ?? 14}: vào ${p.entry ?? 35}, ra ${p.exit ?? 70}`];
  if (p.stopLossPercent) parts.push(`cắt lỗ ${p.stopLossPercent}%`);
  if (p.trendSma) parts.push(`trên SMA ${p.trendSma}`);
  return parts.join(" · ");
}

const date = (t: number) => new Date(t * 1000).toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" });
const years = (run: BacktestRun) => Math.max(1, Math.round((Date.parse(run.to) - Date.parse(run.from)) / (365.25 * 24 * 3600 * 1000)));

// "Kiểm thử lịch sử" (phase-k.md decision 14): no design exists, so this
// follows Quant's strategy card and Portfolio's panels. A run has its own
// URL (?id=); ?symbol= prefills the form (Detail's and Quant's links).
export default async function BacktestPage({ searchParams }: PageProps<"/backtest">) {
  const sp = await searchParams;
  const user = await getSessionUser();
  const token = user ? await getSessionToken() : null;

  let run: BacktestRun | null = null;
  let history: BacktestRun[] = [];
  let error: string | null = null;
  if (token) {
    try {
      history = (await getBacktests(token)).data.slice().reverse();
      if (typeof sp.id === "string") run = await getBacktest(sp.id, token);
    } catch {
      error = "Không tải được kết quả kiểm thử.";
    }
  }

  const symbolParam = typeof sp.symbol === "string" && /^[A-Za-z0-9]{2,10}$/.test(sp.symbol) ? sp.symbol.toUpperCase() : null;
  const defaults: BacktestDefaults = run
    ? { symbol: run.symbol, rule: run.ruleType, params: run.params ?? {}, years: years(run), startingCapital: run.startingCapital }
    : { symbol: symbolParam ?? "FPT", rule: "rsi_reversion", params: {}, years: 3, startingCapital: 100_000_000 };

  const kpis = run
    ? [
        { k: "Lợi nhuận", v: `${signVN(run.returnPercent, 2)}%`, c: tone(run.returnPercent), note: `Mua và giữ ${signVN(run.benchmarkReturnPercent, 2)}%` },
        { k: "Vốn cuối kỳ", v: `${formatVN(run.finalCapital / 1_000_000, 1)} tr ₫`, c: "var(--app-text)", note: `Từ ${formatVN(run.startingCapital / 1_000_000, 0)} tr ₫` },
        { k: "Số lệnh", v: String(run.totalTrades), c: "var(--app-text)", note: run.totalTrades ? `Thắng ${formatVN(run.winRate, 1)}%` : "Chưa có lệnh đóng" },
        { k: "Sụt giảm tối đa", v: run.maxDrawdownPercent ? `−${formatVN(run.maxDrawdownPercent, 2)}%` : "0%", c: run.maxDrawdownPercent ? "#FF5C5C" : "var(--app-text)", note: "Từ đỉnh vốn" },
        { k: "Hệ số lợi nhuận", v: run.profitFactor ? formatVN(run.profitFactor, 2) : "—", c: "var(--app-text)", note: "Lãi gộp / lỗ gộp" },
      ]
    : [];

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <SidebarNav />
      <div className="flex min-w-0 flex-1 flex-col gap-[14px] px-[18px] pb-4 pt-[22px] lg:gap-5 lg:p-[24px_28px]">
        <header className="flex items-center justify-between gap-6 lg:items-end">
          <div className="flex min-w-0 flex-col gap-[5px]">
            <h1 className="m-0 font-display text-[23px] font-bold tracking-[-0.015em] lg:text-[27px]">Kiểm thử lịch sử</h1>
            <span className="text-[11.5px] text-app-text-muted lg:text-[12.5px]">Chạy một quy tắc giao dịch trên nến ngày trong quá khứ và so với mua và giữ</span>
          </div>
          <AccountMenuButton placement="below" />
        </header>

        {!user ? (
          <section className="flex flex-col items-center gap-3 rounded-2xl border border-app-border bg-app-surface p-8 text-center">
            <p className="m-0 max-w-md text-[13.5px] text-app-text-3">Đăng nhập để kiểm thử chiến lược trên dữ liệu lịch sử — mỗi lần chạy được lưu để so sánh.</p>
            <Link href="/login" className="flex h-11 items-center rounded-[11px] bg-app-accent px-6 text-[13.5px] font-semibold text-app-accent-ink">
              Đăng nhập
            </Link>
          </section>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col gap-5 lg:flex-row">
            <div className="order-2 flex w-full flex-col gap-4 lg:order-1 lg:w-[340px] lg:shrink-0">
              <BacktestForm key={run?.id ?? "new"} defaults={defaults} />
              {history.length > 0 && (
                <section className="flex flex-col gap-2 rounded-2xl border border-app-border bg-app-surface p-[18px]">
                  <h2 className="m-0 text-[14px] font-semibold">Các lần chạy trước</h2>
                  <ul className="m-0 flex list-none flex-col p-0">
                    {history.map((h) => (
                      <li key={h.id} className="border-t border-app-hairline first:border-t-0">
                        <Link
                          href={`/backtest?id=${h.id}`}
                          aria-current={h.id === run?.id ? "page" : undefined}
                          className="flex min-h-12 items-center justify-between gap-3 py-2 text-app-text"
                        >
                          <span className="flex min-w-0 flex-col gap-[1px]">
                            <span className="font-plex-mono text-[13px] font-semibold">
                              {h.symbol} <span className="font-sans text-[11.5px] font-normal text-app-text-muted">· {RULE_LABEL[h.ruleType]}</span>
                            </span>
                            <span className="truncate text-[11px] text-app-text-muted">
                              {h.from} → {h.to}
                            </span>
                          </span>
                          <span className="shrink-0 font-plex-mono text-[12.5px] font-semibold" style={{ color: tone(h.returnPercent) }}>
                            {signVN(h.returnPercent, 2)}%
                          </span>
                        </Link>
                      </li>
                    ))}
                  </ul>
                </section>
              )}
            </div>

            <div className="order-1 flex min-w-0 flex-1 flex-col gap-4 lg:order-2">
              {error && <p className="m-0 rounded-xl border border-app-warn-border bg-app-warn-surface p-4 text-sm text-app-warn-text">{error}</p>}
              {!run ? (
                <section className="flex flex-col gap-2 rounded-2xl border border-app-border bg-app-surface p-[18px]">
                  <h2 className="m-0 text-[15px] font-semibold">Chưa chọn lần chạy nào</h2>
                  <p className="m-0 text-[12.5px] leading-[1.55] text-app-text-muted">
                    Chọn mã và quy tắc rồi bấm “Chạy kiểm thử”. Kết quả gồm đường vốn so với mua và giữ, từng lệnh mua/bán, tỷ lệ thắng và mức sụt giảm lớn nhất.
                  </p>
                </section>
              ) : (
                <>
                  <section className="flex flex-col gap-1">
                    <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                      <Link href={`/stocks/${run.symbol}`} className="font-plex-mono text-[20px] font-semibold text-app-text">
                        {run.symbol}
                      </Link>
                      <span className="text-[13px] text-app-text-2">
                        {RULE_LABEL[run.ruleType]} · {ruleSummary(run)}
                      </span>
                    </div>
                    <span className="text-[11.5px] text-app-text-muted">
                      {run.from} → {run.to} · nến ngày · chưa tính phí giao dịch
                    </span>
                  </section>

                  <div className="grid grid-cols-2 gap-[9px] lg:grid-cols-5 lg:gap-[14px]">
                    {kpis.map((k, i) => (
                      <section
                        key={k.k}
                        className={`flex min-w-0 flex-col gap-2 rounded-[14px] border border-app-border bg-app-surface p-[12px_14px] lg:p-[15px_17px] ${i === 0 ? "col-span-2 lg:col-span-1" : ""}`}
                      >
                        <span className="text-[10px] uppercase tracking-[0.06em] text-app-text-muted lg:text-[10.5px]">{k.k}</span>
                        <span className="font-plex-mono text-[18px] font-semibold" style={{ color: k.c }}>
                          {k.v}
                        </span>
                        <span className="text-[11px] text-app-text-muted">{k.note}</span>
                      </section>
                    ))}
                  </div>

                  <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px]">
                    <h2 className="m-0 text-[15px] font-semibold">Đường vốn</h2>
                    <BacktestEquityChart run={run} />
                  </section>

                  <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px]">
                    <div className="flex items-center justify-between">
                      <h2 className="m-0 text-[15px] font-semibold">Các lệnh</h2>
                      <span className="text-[11.5px] text-app-text-muted">{run.trades?.length ?? 0} lượt mua/bán</span>
                    </div>
                    {!run.trades?.length ? (
                      <p className="m-0 text-[12.5px] text-app-text-muted">Quy tắc không phát sinh lệnh nào trong khoảng này.</p>
                    ) : (
                      <div className="-mx-[18px] overflow-x-auto px-[18px]">
                        <table className="w-full min-w-[480px] border-collapse text-[12.5px]">
                          <thead>
                            <tr className="text-[10.5px] uppercase tracking-[0.07em] text-app-text-muted">
                              <th scope="col" className="pb-2 text-left font-medium">Mua</th>
                              <th scope="col" className="pb-2 text-right font-medium">Giá mua</th>
                              <th scope="col" className="pb-2 text-left font-medium pl-4">Bán</th>
                              <th scope="col" className="pb-2 text-right font-medium">Giá bán</th>
                              <th scope="col" className="pb-2 text-right font-medium">Lãi/lỗ</th>
                            </tr>
                          </thead>
                          <tbody>
                            {run.trades.map((t) => (
                              <tr key={t.entryTime} className="border-t border-app-hairline">
                                <td className="py-2 font-plex-mono">{date(t.entryTime)}</td>
                                <td className="py-2 text-right font-plex-mono">{formatThousandsVN(t.entryPrice)}</td>
                                <td className="py-2 pl-4 font-plex-mono">{t.open ? <span className="text-app-text-muted">Đang giữ</span> : date(t.exitTime)}</td>
                                <td className="py-2 text-right font-plex-mono">{formatThousandsVN(t.exitPrice)}</td>
                                <td className="py-2 text-right font-plex-mono font-semibold" style={{ color: tone(t.returnPercent) }}>
                                  {signVN(t.returnPercent, 2)}%
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </section>
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
