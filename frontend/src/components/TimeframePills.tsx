import Link from "next/link";

export type TimeframeKey = "1d" | "1w" | "1m" | "3m" | "1y" | "5y";

export const TIMEFRAMES: { key: TimeframeKey; label: string }[] = [
  { key: "1d", label: "1 ngày" },
  { key: "1w", label: "1 tuần" },
  { key: "1m", label: "1 tháng" },
  { key: "3m", label: "3 tháng" },
  { key: "1y", label: "1 năm" },
  { key: "5y", label: "5 năm" },
];

// Real URL-driven navigation (searchParams), not client state -- matches
// this repo's Server-Component-first pattern (phase-d.md decision 6).
// Default active timeframe is "1m", matching the design's own default.
export default function TimeframePills({ symbol, active }: { symbol: string; active: TimeframeKey }) {
  return (
    <div className="flex gap-[6px]">
      {TIMEFRAMES.map((f) => {
        const on = f.key === active;
        return (
          <Link
            key={f.key}
            href={`/stocks/${symbol}?tf=${f.key}`}
            className="box-border flex h-[34px] shrink-0 items-center whitespace-nowrap rounded-[9px] border px-[13px] text-[12.5px]"
            style={{
              borderColor: on ? "var(--app-border-strong)" : "var(--app-border)",
              background: on ? "var(--app-border)" : "var(--app-surface)",
              color: on ? "var(--app-text)" : "var(--app-text-muted)",
              fontWeight: on ? 600 : 500,
            }}
          >
            {f.label}
          </Link>
        );
      })}
    </div>
  );
}
