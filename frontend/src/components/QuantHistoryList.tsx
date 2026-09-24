"use client";

import { activeConversationStore, historyStore } from "@/lib/quantSettings";

// Quant.dc.html's "Hội thoại gần đây" block under the sidebar nav --
// conversations are kept in this browser only (phase-h.md decision 13).
export default function QuantHistoryList() {
  const history = historyStore.useValue();
  const activeId = activeConversationStore.useValue();

  return (
    <div className="flex flex-col gap-[2px]">
      <span className="px-3 pb-1 text-[10px] uppercase tracking-[0.1em] text-app-text-muted">Hội thoại gần đây</span>
      {history.length === 0 && <span className="px-3 text-[12px] text-app-text-faint">Chưa có hội thoại nào.</span>}
      {history.slice(0, 6).map((c) => (
        <button
          key={c.id}
          type="button"
          onClick={() => activeConversationStore.write(c.id)}
          className="truncate rounded-[9px] px-3 py-[9px] text-left text-[12.5px]"
          style={{
            color: c.id === activeId ? "var(--app-text)" : "var(--app-text-muted)",
            background: c.id === activeId ? "var(--app-surface-2)" : "transparent",
          }}
        >
          {c.title}
        </button>
      ))}
    </div>
  );
}
