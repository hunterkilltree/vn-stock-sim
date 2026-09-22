"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/navItems";
import { LogoMark, AvatarPlaceholder } from "@/components/icons";

// Full 236px sidebar shell -- used on Main/Portfolio/Quant per the design
// canvas (FULL-APP-PLAN.md section 1). Content redesign for the pages
// that use this happens in later phases (C, E, H); this phase only adds
// the shell itself, see phase-a.md.
export default function SidebarNav() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[236px] shrink-0 flex-col border-r border-app-border bg-app-bg px-4 py-6">
      <Link href="/stocks" className="mb-8 flex items-center gap-2 px-2 text-app-fg">
        <LogoMark className="text-app-accent" />
        <span className="font-serif text-base font-medium tracking-tight">VN Stock Sim</span>
      </Link>

      <nav className="flex flex-1 flex-col gap-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = pathname === item.href;

          if (item.kind !== "built") {
            return (
              <div
                key={item.href}
                title="Coming in a later phase"
                className="flex cursor-not-allowed items-center gap-3 rounded-lg px-3 py-2 text-sm text-neutral-600"
              >
                <Icon />
                <span className="flex-1">{item.label}</span>
                <span className="rounded-full border border-app-border px-2 py-0.5 text-[10px] uppercase tracking-wide text-neutral-500">
                  {item.kind === "will" ? "Will" : "Soon"}
                </span>
              </div>
            );
          }

          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                active
                  ? "bg-app-surface text-app-accent"
                  : "text-neutral-300 hover:bg-app-hover hover:text-app-fg"
              }`}
            >
              <Icon />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>

      <button
        type="button"
        title="Account menu — built in a later phase"
        className="flex cursor-not-allowed items-center gap-2 rounded-lg border border-app-border px-3 py-2 text-sm text-neutral-400"
      >
        <AvatarPlaceholder />
        <span>Account</span>
      </button>
    </aside>
  );
}
