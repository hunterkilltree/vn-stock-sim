"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/navItems";
import { LogoMark, AvatarPlaceholder } from "@/components/icons";

function RailIcon({ d }: { d: string }) {
  return (
    <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={d} />
    </svg>
  );
}

// Collapsed 72px icon-only rail -- used on Detail/Replay/Settings/
// Quant-Chart per design/DESIGN-SYSTEM.md section 4 ("Icon rail item":
// 44x44, 11px radius, 19px icon). Icon data now shares navItems.ts's
// exact `d` path list (see phase-c.md) -- full rail-specific spec
// fidelity (its own icon set/ordering, if it differs from the sidebar's)
// is checked against design/screens/Detail.dc.html in Phase D.
export default function RailNav() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[72px] shrink-0 flex-col items-center gap-6 bg-app-chrome py-6">
      <Link href="/stocks" className="text-app-accent" title="VN Stock Sim">
        <LogoMark />
      </Link>

      <nav className="flex flex-1 flex-col items-center gap-1">
        {navItems.map((item) => {
          const active = pathname === item.href;

          if (item.kind !== "built") {
            return (
              <div
                key={item.href}
                title={`${item.label} — designed, not built yet`}
                className="flex h-11 w-11 cursor-not-allowed items-center justify-center rounded-[11px] text-app-text-muted"
              >
                <RailIcon d={item.d} />
              </div>
            );
          }

          return (
            <Link
              key={item.href}
              href={item.href}
              title={item.label}
              aria-label={item.label}
              className="flex h-11 w-11 items-center justify-center rounded-[11px] transition-colors"
              style={{
                color: active ? "var(--app-text)" : "var(--app-text-3)",
                background: active ? "var(--app-surface-3)" : "transparent",
              }}
            >
              <RailIcon d={item.d} />
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
