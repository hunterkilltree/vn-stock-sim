"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/navItems";
import { LogoMark, AvatarPlaceholder } from "@/components/icons";

// Collapsed 72px icon-only rail -- used on Detail/Replay/Settings/
// Quant-Chart per the design canvas, where chart screen space matters
// more than labeled nav (FULL-APP-PLAN.md section 1). Same nav item list
// as SidebarNav so the two shells never diverge -- see phase-a.md.
export default function RailNav() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[72px] shrink-0 flex-col items-center border-r border-app-border bg-app-bg py-6">
      <Link href="/stocks" className="mb-8 text-app-accent" title="VN Stock Sim">
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
                title={`${item.label} — coming in a later phase`}
                className="flex h-11 w-11 cursor-not-allowed items-center justify-center rounded-lg text-neutral-600"
              >
                <Icon />
              </div>
            );
          }

          return (
            <Link
              key={item.href}
              href={item.href}
              title={item.label}
              className={`flex h-11 w-11 items-center justify-center rounded-lg transition-colors ${
                active
                  ? "bg-app-surface text-app-accent"
                  : "text-neutral-300 hover:bg-app-hover hover:text-app-fg"
              }`}
            >
              <Icon />
            </Link>
          );
        })}
      </nav>

      <button
        type="button"
        title="Account menu — built in a later phase"
        className="flex h-11 w-11 cursor-not-allowed items-center justify-center rounded-lg border border-app-border text-neutral-400"
      >
        <AvatarPlaceholder />
      </button>
    </aside>
  );
}
