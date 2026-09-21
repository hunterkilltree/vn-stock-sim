import type { Insight } from "@/lib/api";

const directionStyles: Record<string, string> = {
  bullish: "bg-emerald-500/10 text-emerald-400",
  bearish: "bg-red-500/10 text-red-400",
  neutral: "bg-neutral-500/10 text-neutral-400",
};

// Follows luxalgo.com's "Quant" AI-assistant panel pattern (dark card,
// sparkle mark, short written read on the chart) but scoped down and
// labeled honestly: this is a deterministic, rule-based summary over
// price/trend/volume (see backend/internal/insight), not a real LLM
// call -- wiring one up needs an API key and has a real per-call cost,
// which is the user's call to make, not something to assume. The
// "rule-based preview" badge and Insight.source both make this explicit
// so a future real integration is a visible, deliberate upgrade, not a
// silent swap.
export default function AIInsightCard({ insight }: { insight: Insight }) {
  return (
    <div className="overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-950 p-6">
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-base">✨</span>
          <h2 className="text-sm font-medium text-neutral-100">AI Insight</h2>
        </div>
        <span className="rounded-full border border-neutral-700 px-2.5 py-0.5 text-xs text-neutral-400">
          rule-based preview
        </span>
      </div>

      <p className="mb-5 text-sm leading-relaxed text-neutral-300">{insight.summary}</p>

      <ul className="space-y-2">
        {insight.signals.map((signal) => (
          <li key={signal.label} className="flex items-center justify-between rounded-lg bg-neutral-900 px-3 py-2 text-sm">
            <span className="text-neutral-400">{signal.label}</span>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${directionStyles[signal.direction]}`}>
              {signal.detail}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
