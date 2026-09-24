"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import WillBadge from "@/components/WillBadge";
import { settingsSections } from "@/lib/settingsSections";

// The 232px Settings column from design/screens/Settings-AI.dc.html.
export default function SettingsSubnav() {
  const pathname = usePathname();

  return (
    <div className="flex w-[232px] shrink-0 flex-col gap-[18px] border-r border-app-border bg-app-chrome p-[24px_16px]">
      <h1 className="m-0 px-2 font-display text-[22px] font-bold tracking-[-0.015em]">Cài đặt</h1>
      <nav aria-label="Mục cài đặt" className="flex flex-col gap-[2px]">
        {settingsSections.map((s) => {
          const href = `/settings/${s.slug}`;
          const active = pathname === href;
          return (
            <Link
              key={s.slug}
              href={href}
              aria-current={active ? "page" : undefined}
              className="flex h-10 items-center gap-2 rounded-[10px] px-3 text-[13.5px]"
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
      <div className="flex-grow" />
      <div className="flex flex-col gap-[6px] rounded-xl border border-app-border bg-app-surface-2 p-[13px]">
        <span className="text-[10px] uppercase tracking-[0.1em] text-app-text-muted">Tài khoản</span>
        <span className="text-[12.5px] text-app-text-2">Gói Cá nhân · miễn phí</span>
        <span className="flex items-center gap-2 text-xs font-semibold text-app-text-muted">
          So sánh các gói <WillBadge />
        </span>
      </div>
    </div>
  );
}
