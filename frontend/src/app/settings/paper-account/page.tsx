import Link from "next/link";
import RiskSettingsForm from "@/components/RiskSettingsForm";
import { getRiskSettings, type RiskSettings } from "@/lib/api";
import { getActivePortfolio, getSessionToken, getSessionUser } from "@/lib/session";

export const metadata = { title: "Tài khoản giấy — Cài đặt — VN Stock Sim" };

// Settings → Tài khoản giấy (phase-risk-rules.md slice 1): the active
// portfolio's risk-rule mode and thresholds. No design exists for this
// section; it follows the AI settings page's panels.
export default async function PaperAccountSettingsPage() {
  const user = await getSessionUser();
  const token = user ? await getSessionToken() : null;
  let portfolio: { id: string; name: string } | null = null;
  let settings: RiskSettings | null = null;
  if (token) {
    try {
      const { active } = await getActivePortfolio(token);
      if (active) {
        portfolio = { id: active.id, name: active.name };
        settings = await getRiskSettings(active.id, token);
      }
    } catch {
      // shown as the error panel below
    }
  }

  return (
    <>
      <header className="flex flex-col gap-1">
        <h2 className="m-0 font-display text-[25px] font-bold tracking-[-0.015em]">Tài khoản giấy</h2>
        <span className="text-[12.5px] text-app-text-muted">Quy tắc quản trị rủi ro áp dụng cho từng lệnh mua của danh mục đang chọn</span>
      </header>
      {!user ? (
        <section className="flex max-w-[640px] flex-col items-start gap-3 rounded-[14px] border border-app-border bg-app-surface p-[18px]">
          <p className="m-0 text-[13px] text-app-text-3">Đăng nhập để đặt quy tắc rủi ro cho danh mục của bạn.</p>
          <Link href="/login" className="flex h-10 items-center rounded-[10px] bg-app-accent px-5 text-[13px] font-semibold text-app-accent-ink">
            Đăng nhập
          </Link>
        </section>
      ) : !portfolio || !settings ? (
        <p className="m-0 max-w-[640px] rounded-xl border border-app-warn-border bg-app-warn-surface p-4 text-sm text-app-warn-text">
          Không tải được cài đặt của danh mục.
        </p>
      ) : (
        <RiskSettingsForm key={portfolio.id} portfolioId={portfolio.id} portfolioName={portfolio.name} initial={settings} />
      )}
    </>
  );
}
