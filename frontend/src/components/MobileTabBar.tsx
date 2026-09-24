"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { Mode } from "@/lib/navItems";

type Tab = { label: string; href: string; d: string; active: (path: string) => boolean };

// Icon paths copied from the Mobile-*.dc.html tab() calls.
const MARKET_D = "M3 3h18v18H3zM9 3v18M15 3v18M3 9h18M3 15h18";
const CHART_D = "M4 20V11M9 20V4M14 20V14M19 20V8";
const WALLET_D = "M3 7h18v12H3zM3 7l3-4h12l3 4M16 13h2";
const QUANT_D = "M12 3l2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5z";

function tabsFor(mode: Mode): { left: Tab[]; right: Tab[]; replay: string } {
  if (mode === "crypto") {
    return {
      left: [
        { label: "Thị trường", href: "/crypto", d: MARKET_D, active: (p) => p === "/crypto" || p === "/heatmap" },
        {
          label: "Biểu đồ",
          href: "/crypto/BTCUSDT",
          d: CHART_D,
          active: (p) => p.startsWith("/crypto/") && p !== "/crypto/replay",
        },
      ],
      right: [
        // Crypto has no Portfolio page; the wallet card lives on the overview (phase-i.md decision 10).
        { label: "Ví", href: "/crypto#vi-crypto", d: WALLET_D, active: () => false },
        { label: "Quant", href: "/quant", d: QUANT_D, active: (p) => p.startsWith("/quant") },
      ],
      replay: "/crypto/replay",
    };
  }
  return {
    left: [
      { label: "Thị trường", href: "/stocks", d: MARKET_D, active: (p) => p === "/stocks" || p === "/heatmap" },
      { label: "Biểu đồ", href: "/stocks/VNM", d: CHART_D, active: (p) => p.startsWith("/stocks/") },
    ],
    right: [
      { label: "Danh mục", href: "/portfolio", d: WALLET_D, active: (p) => p.startsWith("/portfolio") },
      { label: "Quant", href: "/quant", d: QUANT_D, active: (p) => p.startsWith("/quant") },
    ],
    replay: "/replay",
  };
}

function TabLink({ tab, on }: { tab: Tab; on: boolean }) {
  return (
    <Link
      href={tab.href}
      aria-current={on ? "page" : undefined}
      className="flex w-16 flex-col items-center gap-1 rounded-xl py-[6px]"
      style={{ color: on ? "var(--app-text)" : "var(--app-text-muted)" }}
    >
      <svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d={tab.d} />
      </svg>
      <span className="text-[10px] font-medium">{tab.label}</span>
    </Link>
  );
}

// Phone bottom navigation from design/screens/Mobile-Market.dc.html: two
// tabs, the raised Replay button, two tabs (phase-j.md decision 2). Shown
// below `lg` only; the desktop sidebar/rail takes over from there. Its
// height is --tabbar-h, which globals.css reserves at the bottom of the
// page whenever the bar is present.
export default function MobileTabBar({ mode }: { mode: Mode }) {
  const pathname = usePathname();
  const { left, right, replay } = tabsFor(mode);
  return (
    <nav
      data-mobile-tabbar
      aria-label="Điều hướng"
      className="fixed inset-x-0 bottom-0 z-40 flex items-center justify-between border-t border-app-border bg-app-chrome px-3 pt-2 lg:hidden"
      style={{ height: "var(--tabbar-h)", paddingBottom: "calc(18px + env(safe-area-inset-bottom))" }}
    >
      {left.map((t) => (
        <TabLink key={t.label} tab={t} on={t.active(pathname)} />
      ))}
      <Link
        href={replay}
        aria-label="Chế độ Replay"
        className="mb-[10px] flex h-14 w-14 shrink-0 flex-col items-center justify-center gap-[1px] rounded-[18px] bg-app-accent text-app-accent-ink"
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
        </svg>
        <span className="text-[9px] font-bold">Replay</span>
      </Link>
      {right.map((t) => (
        <TabLink key={t.label} tab={t} on={t.active(pathname)} />
      ))}
    </nav>
  );
}
