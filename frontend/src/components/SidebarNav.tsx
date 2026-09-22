"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/navItems";
import { LogoMark, AvatarPlaceholder } from "@/components/icons";
import WillBadge from "@/components/WillBadge";

// Full 236px sidebar shell -- used on Main/Portfolio/Quant per
// design/DESIGN-SYSTEM.md section 4 ("Nav item (sidebar)"). Content
// redesign for the pages that use this happens in later phases (C, E,
// H); this is the shell only, corrected to match design/tokens.css
// exactly in phase-design-alignment.md (chrome background, not bg;
// active state is text on surface-3, not accent-colored text).
export default function SidebarNav() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[236px] shrink-0 flex-col gap-6 bg-app-chrome px-4 py-6">
      <Link href="/stocks" className="flex items-center gap-2 px-2 text-app-text">
        <LogoMark className="text-app-accent" />
        <span className="font-display text-base font-semibold tracking-tight">VN Stock Sim</span>
      </Link>

      <nav className="flex flex-1 flex-col gap-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = pathname === item.href;

          if (item.kind !== "built") {
            return (
              <div
                key={item.href}
                title="Designed, not built yet"
                className="flex h-10 cursor-not-allowed items-center gap-[11px] rounded-[11px] px-3 text-[13.5px] text-app-text-muted"
              >
                <Icon width={18} height={18} />
                <span className="flex-1">{item.label}</span>
                {item.kind === "will" && <WillBadge />}
              </div>
            );
          }

          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex h-10 items-center gap-[11px] rounded-[11px] px-3 text-[13.5px] transition-colors ${
                active ? "bg-app-surface-3 text-app-text" : "text-app-text-3 hover:bg-app-surface-3/50 hover:text-app-text"
              }`}
            >
              <Icon width={18} height={18} />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>

      <button
        type="button"
        title="Account menu — built in a later phase"
        className="flex h-10 cursor-not-allowed items-center gap-2 rounded-[11px] border border-app-border px-3 text-[13.5px] text-app-text-muted"
      >
        <AvatarPlaceholder width={18} height={18} />
        <span>Account</span>
      </button>
    </aside>
  );
}
