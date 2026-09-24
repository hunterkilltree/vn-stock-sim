"use client";

import { useState } from "react";
import type { Allocation, EquityPoint, Order, Position, PortfolioStats } from "@/lib/api";
import KpiRow from "./KpiRow";
import EquityCurveChart from "./EquityCurveChart";
import SectorAllocationCard from "./SectorAllocationCard";
import HoldingsTable from "./HoldingsTable";
import PendingOrdersCard from "./PendingOrdersCard";
import JournalCard from "./JournalCard";

type Props = {
  stats: PortfolioStats;
  equityHistory: EquityPoint[];
  positions: Position[];
  allocation: Allocation[];
  pendingOrders: Order[];
  startingCapital: number;
};

const TABS = ["Tổng quan", "Vị thế", "Lệnh chờ", "Sổ giao dịch"] as const;

// Portfolio.dc.html's 4-tab layout (phase-e.md item 4): data is fetched
// once server-side (page.tsx) and passed down as props, tab-switching
// happens client-side with no re-fetch/reload, matching the design's
// instant tab switching.
export default function PortfolioTabs({ stats, equityHistory, positions, allocation, pendingOrders, startingCapital }: Props) {
  const [tab, setTab] = useState<(typeof TABS)[number]>("Tổng quan");

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[18px]">
      <div className="-mx-[18px] flex gap-[6px] overflow-x-auto px-[18px] lg:mx-0 lg:px-0">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className="flex h-10 shrink-0 items-center gap-[6px] whitespace-nowrap rounded-[9px] border px-[14px] text-[13px] lg:h-9"
            style={
              tab === t
                ? { borderColor: "var(--app-border-strong)", background: "var(--app-border)", color: "var(--app-text)", fontWeight: 600 }
                : { borderColor: "transparent", background: "transparent", color: "var(--app-text-muted)", fontWeight: 500 }
            }
          >
            {t}
            {t === "Lệnh chờ" && pendingOrders.length > 0 && (
              <span className="rounded bg-app-surface-3 px-[5px] py-[1px] font-plex-mono text-[10px] text-app-text-2">
                {pendingOrders.length}
              </span>
            )}
          </button>
        ))}
      </div>

      <KpiRow stats={stats} />

      {tab === "Tổng quan" && (
        <div className="flex min-h-0 flex-1 flex-col gap-[18px] lg:flex-row">
          <div className="flex min-w-0 flex-1 flex-col gap-[18px]">
            <EquityCurveChart points={equityHistory} startingCapital={startingCapital} />
            <HoldingsTable positions={positions} totalEquity={stats.totalEquity} />
          </div>
          <div className="flex w-full flex-col gap-[18px] lg:w-[352px] lg:shrink-0">
            <SectorAllocationCard allocation={allocation} />
            <PendingOrdersCard orders={pendingOrders} />
          </div>
        </div>
      )}

      {tab === "Vị thế" && <HoldingsTable positions={positions} totalEquity={stats.totalEquity} />}

      {tab === "Lệnh chờ" && <PendingOrdersCard orders={pendingOrders} />}

      {tab === "Sổ giao dịch" && (
        <div className="flex min-h-0 flex-1 flex-col gap-[18px] lg:flex-row">
          <div className="min-w-0 flex-1">
            <JournalCard stats={stats} />
          </div>
          <div className="w-full lg:w-[352px] lg:shrink-0">
            <SectorAllocationCard allocation={allocation} />
          </div>
        </div>
      )}
    </div>
  );
}
