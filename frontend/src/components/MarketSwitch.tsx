import Link from "next/link";
import type { Mode } from "@/lib/navItems";

// Compact Cổ phiếu | Crypto control for phone headers, where the
// sidebar's full-width switch is hidden (phase-j.md decision 3).
export default function MarketSwitch({ mode, stockHref = "/stocks", cryptoHref = "/crypto" }: { mode: Mode; stockHref?: string; cryptoHref?: string }) {
  return (
    <div role="group" aria-label="Chọn thị trường" className="flex gap-1 rounded-xl border border-app-border bg-app-surface p-1 lg:hidden">
      {(
        [
          { m: "stock", href: stockHref, label: "Cổ phiếu" },
          { m: "crypto", href: cryptoHref, label: "Crypto" },
        ] as const
      ).map((x) => {
        const on = x.m === mode;
        return (
          <Link
            key={x.m}
            href={x.href}
            aria-current={on ? "page" : undefined}
            className="flex h-9 flex-1 items-center justify-center rounded-[9px] px-4 text-[12.5px]"
            style={{
              background: on ? "var(--app-accent)" : "transparent",
              color: on ? "var(--app-accent-ink)" : "var(--app-text-3)",
              fontWeight: on ? 600 : 500,
            }}
          >
            {x.label}
          </Link>
        );
      })}
    </div>
  );
}
