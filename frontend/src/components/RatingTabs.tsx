"use client";

import { useState, type ReactNode } from "react";

// Tab row for the Khuyến nghị card (phase-holding-horizon.md "Where it
// shows up"). Panels are rendered on the server and passed in; this only
// toggles which one is visible.
export default function RatingTabs({ tabs }: { tabs: { key: string; label: string; content: ReactNode }[] }) {
  const [active, setActive] = useState(tabs[0]?.key);
  return (
    <div className="flex flex-col gap-3">
      <div role="tablist" aria-label="Tầm nắm giữ" className="grid gap-1 rounded-[10px] bg-app-surface-2 p-[3px]" style={{ gridTemplateColumns: `repeat(${tabs.length}, 1fr)` }}>
        {tabs.map((t) => (
          <button
            key={t.key}
            type="button"
            role="tab"
            id={`rating-tab-${t.key}`}
            aria-selected={active === t.key}
            aria-controls={`rating-panel-${t.key}`}
            onClick={() => setActive(t.key)}
            className={`h-8 rounded-[8px] text-[11.5px] font-medium ${
              active === t.key ? "bg-app-surface-3 text-app-text" : "text-app-text-muted hover:text-app-text-2"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>
      {tabs.map((t) => (
        <div key={t.key} role="tabpanel" id={`rating-panel-${t.key}`} aria-labelledby={`rating-tab-${t.key}`} hidden={active !== t.key}>
          {t.content}
        </div>
      ))}
    </div>
  );
}
