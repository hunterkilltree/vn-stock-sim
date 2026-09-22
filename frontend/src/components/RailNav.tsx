"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/navItems";
import { LogoMark, AvatarPlaceholder } from "@/components/icons";

// Collapsed 72px icon-only rail -- used on Detail/Replay/Settings/
// Quant-Chart per design/DESIGN-SYSTEM.md section 4 ("Icon rail item":
// 44x44, 11px radius, 19px icon). Corrected in phase-design-alignment.md
// to use the chrome background and the spec's active-state rule (text
// on surface-3, not accent-colored icon).
export default function RailNav() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[72px] shrink-0 flex-col items-center gap-6 bg-app-chrome py-6">
      <Link href="/stocks" className="text-app-accent" title="VN Stock Sim">
        <LogoMark />
      </Link>

      <nav className="flex flex-1 flex-col items-center gap-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = pathname === item.href;

          if (item.kind !== "built") {
            return (
              <div
                key={item.href}
                title={`${item.label} — designed, not built yet`}
                className="flex h-11 w-11 cursor-not-allowed items-center justify-center rounded-[11px] text-app-text-muted"
              >
                <Icon width={19} height={19} />
              </div>
            );
          }

          return (
            <Link
              key={item.href}
              href={item.href}
              title={item.label}
              aria-label={item.label}
              className={`flex h-11 w-11 items-center justify-center rounded-[11px] transition-colors ${
                active ? "bg-app-surface-3 text-app-text" : "text-app-text-3 hover:bg-app-surface-3/50 hover:text-app-text"
              }`}
            >
              <Icon width={19} height={19} />
            </Link>
          );
        })}
      </nav>

      <button
        type="button"
        title="Account menu — built in a later phase"
        aria-label="Account menu"
        className="flex h-11 w-11 cursor-not-allowed items-center justify-center rounded-[11px] border border-app-border text-app-text-muted"
      >
        <AvatarPlaceholder width={19} height={19} />
      </button>
    </aside>
  );
}
