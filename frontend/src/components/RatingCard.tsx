import type { Rating, RatingKey, RatingSignal, RatingSummary } from "@/lib/api";
import { formatVN } from "@/lib/format";

// "Khuyến nghị": the rule-based Strong Buy .. Strong Sell rating from
// backend/internal/rating (phase-rating.md). A five-step scale with the
// active step lit, the technical/fundamental split, and every rule's vote
// so the user can see why -- not a black box, and labeled as not advice.
const STEPS: { key: RatingKey; label: string; color: string }[] = [
  { key: "strong_sell", label: "Bán mạnh", color: "#FF5C5C" },
  { key: "sell", label: "Bán", color: "#FF8A7A" },
  { key: "neutral", label: "Trung lập", color: "#E3B341" },
  { key: "buy", label: "Mua", color: "#7FD8A6" },
  { key: "strong_buy", label: "Mua mạnh", color: "#35C77F" },
];

const VERDICT: Record<RatingSignal["verdict"], { label: string; className: string }> = {
  buy: { label: "Mua", className: "text-price-up" },
  neutral: { label: "Trung lập", className: "text-app-text-muted" },
  sell: { label: "Bán", className: "text-price-down" },
};

function step(key: RatingKey) {
  return STEPS.find((s) => s.key === key) ?? STEPS[2];
}

function GroupRow({ title, summary }: { title: string; summary: RatingSummary }) {
  const s = step(summary.rating);
  return (
    <div className="flex flex-col gap-[3px] rounded-[11px] border border-app-hairline bg-app-surface-2 p-[9px_11px]">
      <span className="text-[10.5px] text-app-text-muted">{title}</span>
      <span className="text-[13px] font-semibold" style={{ color: s.color }}>
        {s.label}
      </span>
      <span className="whitespace-nowrap text-[10.5px] text-app-text-muted">
        <span className="text-price-up">{summary.buy} mua</span> · {summary.neutral} TL ·{" "}
        <span className="text-price-down">{summary.sell} bán</span>
      </span>
    </div>
  );
}

function SignalList({ title, signals }: { title: string; signals: RatingSignal[] }) {
  if (signals.length === 0) return null;
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[10.5px] uppercase tracking-[0.04em] text-app-text-faint">{title}</span>
      <ul className="m-0 flex list-none flex-col p-0">
        {signals.map((sig) => (
          <li key={sig.label} className="flex items-start justify-between gap-3 border-b border-app-hairline py-[7px] last:border-b-0">
            <div className="flex min-w-0 flex-col gap-[1px]">
              <span className="text-[12px] text-app-text-2">{sig.label}</span>
              <span className="text-[10.5px] text-app-text-muted">{sig.detail}</span>
            </div>
            <div className="flex shrink-0 flex-col items-end gap-[1px]">
              <span className={`text-[11.5px] font-semibold ${VERDICT[sig.verdict].className}`}>{VERDICT[sig.verdict].label}</span>
              {/* Backend formats with ".", the app shows Vietnamese "," decimals. */}
              <span className="font-plex-mono text-[10.5px] text-app-text-muted">{sig.value.replaceAll(".", ",")}</span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function RatingCard({ rating }: { rating: Rating }) {
  const active = step(rating.overall.rating);
  const technical = rating.signals.filter((s) => s.group === "technical");
  const fundamental = rating.signals.filter((s) => s.group === "fundamental");

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[16px_18px]" aria-label="Khuyến nghị">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[13px] font-semibold">Khuyến nghị</h2>
        <span className="rounded-full border border-app-border px-[8px] py-[1px] text-[10px] text-app-text-muted">theo quy tắc</span>
      </div>

      <div className="flex items-baseline justify-between gap-3">
        <span className="font-display text-[24px] font-bold" style={{ color: active.color }}>
          {active.label}
        </span>
        <span className="font-plex-mono text-[11.5px] text-app-text-muted" title="Điểm trung bình các tín hiệu, từ -1 (bán) đến +1 (mua)">
          Điểm {rating.overall.score > 0 ? "+" : ""}
          {formatVN(rating.overall.score, 2)}
        </span>
      </div>

      <div className="flex flex-col gap-[5px]" role="img" aria-label={`Khuyến nghị: ${active.label}`}>
        <div className="grid grid-cols-5 gap-[3px]">
          {STEPS.map((s) => (
            <div
              key={s.key}
              className="h-[6px] rounded-full"
              style={{ background: s.color, opacity: s.key === active.key ? 1 : 0.18 }}
            />
          ))}
        </div>
        <div className="grid grid-cols-5 gap-[3px] text-center text-[9.5px]">
          {STEPS.map((s) => (
            <span
              key={s.key}
              className={s.key === active.key ? "font-semibold" : "text-app-text-faint"}
              style={s.key === active.key ? { color: s.color } : undefined}
            >
              {s.label}
            </span>
          ))}
        </div>
      </div>

      <div className={`grid gap-2 ${rating.fundamental ? "grid-cols-2" : "grid-cols-1"}`}>
        <GroupRow title="Kỹ thuật" summary={rating.technical} />
        {rating.fundamental && <GroupRow title="Cơ bản" summary={rating.fundamental} />}
      </div>

      <details className="group">
        <summary className="cursor-pointer list-none text-[11.5px] font-medium text-app-accent">
          <span className="group-open:hidden">Xem {rating.signals.length} tín hiệu</span>
          <span className="hidden group-open:inline">Ẩn tín hiệu</span>
        </summary>
        <div className="mt-2 flex flex-col gap-3">
          <SignalList title="Kỹ thuật" signals={technical} />
          <SignalList title="Cơ bản" signals={fundamental} />
        </div>
      </details>

      <p className="m-0 text-[10px] leading-[1.45] text-app-text-faint">
        Tổng hợp từ các quy tắc phổ biến (đường trung bình, RSI, MACD, động lượng, khối lượng, P/E, P/B, ROE, cổ tức). Chỉ
        mang tính tham khảo cho mô phỏng, không phải khuyến nghị đầu tư.
      </p>
    </section>
  );
}
