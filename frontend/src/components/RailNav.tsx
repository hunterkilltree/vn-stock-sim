"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { isNavActive, navItemsFor, settingsItem, type Mode } from "@/lib/navItems";
import { LogoMark } from "@/components/icons";

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
// `account` is the server-rendered <AccountMenuButton placement="right"/>
// -- a Server Component can only reach this Client shell as a prop
// (phase-g.md decision 9).
export default function RailNav({ account, mode = "stock" }: { account: ReactNode; mode?: Mode }) {
  const pathname = usePathname();
  const items = navItemsFor(mode);
  const settingsActive = pathname.startsWith("/settings");

  return (
    <aside className="flex w-[72px] shrink-0 flex-col items-center gap-6 bg-app-chrome py-6">
      <Link href={mode === "crypto" ? "/crypto" : "/stocks"} className="text-app-accent" title="VN Stock Sim">
        <LogoMark />
      </Link>

      {/* Crypto-Detail.dc.html's CP / CRY switch (phase-i.md decision 13). */}
      <div role="group" aria-label="Chọn thị trường" className="-mt-2 flex flex-col gap-[3px]">
        {([
          { m: "stock", href: "/stocks", label: "CP", aria: "Thị trường cổ phiếu" },
          { m: "crypto", href: "/crypto", label: "CRY", aria: "Thị trường crypto" },
        ] as const).map((x) => {
          const on = x.m === mode;
          return (
            <Link
              key={x.m}
              href={x.href}
              aria-label={x.aria}
              aria-current={on ? "page" : undefined}
              className="flex h-[30px] w-11 items-center justify-center rounded-[9px] text-[10px]"
              style={{
                background: on ? "var(--app-accent)" : "transparent",
                color: on ? "var(--app-accent-ink)" : "var(--app-text-muted)",
                fontWeight: on ? 700 : 600,
              }}
            >
              {x.label}
            </Link>
          );
        })}
      </div>

      <nav className="flex flex-1 flex-col items-center gap-1">
        {items.map((item) => {
          const active = isNavActive(item, pathname, mode);

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

      <div className="flex flex-col items-center gap-2">
        <Link
          href={settingsItem.href}
          title={settingsItem.label}
          aria-label={settingsItem.label}
          aria-current={settingsActive ? "page" : undefined}
          className="flex h-11 w-11 items-center justify-center rounded-[11px]"
          style={{
            color: settingsActive ? "var(--app-text)" : "var(--app-text-3)",
            background: settingsActive ? "var(--app-surface-3)" : "transparent",
          }}
        >
          <RailIcon d={settingsItem.d} />
        </Link>
        {account}
      </div>
    </aside>
  );
}
