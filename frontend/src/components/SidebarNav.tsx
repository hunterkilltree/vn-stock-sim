"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems, settingsItem } from "@/lib/navItems";
import WillBadge from "@/components/WillBadge";
import { formatVN } from "@/lib/format";

type Props = {
  // Only known when the caller has an authenticated session's portfolio
  // summary -- omitted entirely (not a fake "0") for a guest, per
  // phase-c.md decision 5.
  cashBalance?: number;
};

function NavIcon({ d }: { d: string }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={d} />
    </svg>
  );
}

// Full 236px sidebar shell -- used on Main/Portfolio/Quant per
// design/DESIGN-SYSTEM.md section 4. Rebuilt in Phase C against the real
// spec (design/screens/Main.dc.html): two-line logo lockup, the real
// 10-item nav list, a separate Settings link outside the loop, and a
// balance card wired to real portfolio data instead of the mockup's
// static sample. See phase-c.md.
export default function SidebarNav({ cashBalance }: Props) {
  const pathname = usePathname();

  return (
    <aside className="flex w-[236px] shrink-0 flex-col gap-5 bg-app-chrome px-4 py-[22px]">
      <Link href="/stocks" className="flex items-center gap-[10px] px-[6px] text-app-text">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[9px] bg-app-accent">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#131210" strokeWidth="2.2" strokeLinecap="round" aria-hidden="true">
            <path d="M7 4v5M7 15v5M7 9h0M17 3v6M17 17v4" />
            <rect x="4" y="9" width="6" height="6" fill="#131210" stroke="none" />
            <rect x="14" y="9" width="6" height="8" fill="#131210" stroke="none" />
          </svg>
        </span>
        <span className="flex flex-col gap-[1px]">
          <span className="font-display text-[17px] font-bold tracking-[-0.01em]">VN Stock Sim</span>
          <span className="text-[10px] uppercase tracking-[0.1em] text-app-text-muted">Mô phỏng · HOSE</span>
        </span>
      </Link>

      <nav className="flex flex-col gap-[2px]">
        {navItems.map((item) => {
          const active = pathname === item.href;

          if (item.kind !== "built") {
            return (
              <div
                key={item.href}
                title={item.kind === "will" ? "Chưa có thiết kế" : "Sắp ra mắt"}
                className="flex h-10 cursor-not-allowed items-center gap-[11px] rounded-[10px] px-3 text-app-text-3"
              >
                <NavIcon d={item.d} />
                <span className="text-[13.5px] font-medium">{item.label}</span>
                {item.kind === "will" && <WillBadge />}
              </div>
            );
          }

          return (
            <Link
              key={item.href}
              href={item.href}
              className="flex h-10 items-center gap-[11px] rounded-[10px] px-3"
              style={{
                color: active ? "var(--app-text)" : "var(--app-text-3)",
                background: active ? "var(--app-surface-3)" : "transparent",
              }}
            >
              <NavIcon d={item.d} />
              <span className="text-[13.5px] font-medium">{item.label}</span>
            </Link>
          );
        })}
      </nav>

      <div className="flex-grow" />

      <Link href={settingsItem.href} className="flex h-10 items-center gap-[11px] rounded-[10px] px-3 text-app-text-3">
        <NavIcon d={settingsItem.d} />
        <span className="text-[13.5px] font-medium">{settingsItem.label}</span>
      </Link>

      {cashBalance !== undefined && (
        <div className="flex flex-col gap-[6px] rounded-xl border border-app-border bg-app-surface-2 p-[14px]">
          <span className="text-[10px] uppercase tracking-[0.1em] text-app-text-muted">Số dư ảo</span>
          <span className="font-plex-mono text-[18px] font-semibold text-app-text">{formatVN(cashBalance, 0)} ₫</span>
          <span className="text-[11.5px] text-app-text-muted">Không dùng tiền thật</span>
        </div>
      )}
    </aside>
  );
}
