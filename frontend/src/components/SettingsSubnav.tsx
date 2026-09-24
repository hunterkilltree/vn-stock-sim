"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import WillBadge from "@/components/WillBadge";
import { settingsSections } from "@/lib/settingsSections";

// The 232px Settings column from design/screens/Settings-AI.dc.html; on
// phones a title plus a scrolling row of section tabs (phase-j.md
// decision 11).
export default function SettingsSubnav() {
  const pathname = usePathname();

  return (
    <div className="flex w-full flex-col gap-3 border-b border-app-border bg-app-chrome px-[18px] pb-3 pt-[22px] lg:w-[232px] lg:shrink-0 lg:gap-[18px] lg:border-b-0 lg:border-r lg:p-[24px_16px]">
      <h1 className="m-0 font-display text-[22px] font-bold tracking-[-0.015em] lg:px-2">Cài đặt</h1>
      <nav aria-label="Mục cài đặt" className="-mx-[18px] flex gap-1 overflow-x-auto px-[18px] lg:mx-0 lg:flex-col lg:gap-[2px] lg:px-0">
        {settingsSections.map((s) => {
          const href = `/settings/${s.slug}`;
          const active = pathname === href;
          return (
            <Link
              key={s.slug}
              href={href}
              aria-current={active ? "page" : undefined}
              className="flex h-10 shrink-0 items-center gap-2 whitespace-nowrap rounded-[10px] px-3 text-[13.5px]"
              style={{
                fontWeight: active ? 600 : 500,
                color: active ? "var(--app-text)" : "var(--app-text-3)",
                background: active ? "var(--app-surface-3)" : "transparent",
              }}
            >
              <span>{s.label}</span>
              {!s.built && (
                <span className="ml-auto">
                  <WillBadge />
                </span>
              )}
            </Link>
          );
        })}
      </nav>
      <div className="hidden flex-grow lg:block" />
      <div className="hidden flex-col gap-[6px] rounded-xl lg:flex border border-app-border bg-app-surface-2 p-[13px]">
        <span className="text-[10px] uppercase tracking-[0.1em] text-app-text-muted">Tài khoản</span>
        <span className="text-[12.5px] text-app-text-2">Gói Cá nhân · miễn phí</span>
        <span className="flex items-center gap-2 text-xs font-semibold text-app-text-muted">
          So sánh các gói <WillBadge />
        </span>
      </div>
    </div>
  );
}
