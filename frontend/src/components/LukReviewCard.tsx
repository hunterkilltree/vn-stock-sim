import type { LukCheck, LukReview } from "@/lib/api";
import { formatThousandsVN, formatVN } from "@/lib/format";
import LukAIReview from "@/components/LukAIReview";

// "Quy tắc Martin Luk": should I buy today by his swing-trading rules?
// The checklist is deterministic (backend/internal/luk,
// phase-luk-review.md); the Gemini commentary is optional and on demand.
const VERDICT: Record<LukReview["verdict"], { label: string; color: string; hint: string }> = {
  buy: { label: "Đủ điều kiện mua", color: "#35C77F", hint: "Mọi quy tắc cốt lõi đều đạt." },
  watch: { label: "Theo dõi", color: "#E3B341", hint: "Cổ phiếu đáng chú ý, nhưng hôm nay chưa phải điểm vào." },
  avoid: { label: "Không mua", color: "#FF5C5C", hint: "Vi phạm quy tắc cốt lõi." },
};

const STATUS: Record<LukCheck["status"], { icon: string; className: string; label: string }> = {
  pass: { icon: "✓", className: "text-price-up", label: "Đạt" },
  warn: { icon: "!", className: "text-[#E3B341]", label: "Cần chú ý" },
  fail: { icon: "✕", className: "text-price-down", label: "Không đạt" },
};

const GROUPS: { key: LukCheck["group"]; title: string }[] = [
  { key: "setup", title: "Chọn cổ phiếu" },
  { key: "entry", title: "Điểm vào" },
  { key: "risk", title: "Rủi ro & thị trường" },
];

function CheckRow({ c }: { c: LukCheck }) {
  const s = STATUS[c.status];
  return (
    <li className="flex items-start gap-2 border-b border-app-hairline py-[7px] last:border-b-0">
      <span className={`w-3 shrink-0 text-center text-[12px] font-bold ${s.className}`} aria-label={s.label}>
        {s.icon}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-[1px]">
        <span className="text-[12px] text-app-text-2">
          {c.label}
          {c.core && <span className="ml-1 text-[9.5px] text-app-text-faint">· cốt lõi</span>}
        </span>
        <span className="font-plex-mono text-[10.5px] text-app-text-muted">{c.value.replaceAll(".", ",")}</span>
        <span className="text-[10.5px] text-app-text-muted">{c.detail}</span>
      </div>
    </li>
  );
}

export default function LukReviewCard({ review }: { review: LukReview }) {
  const v = VERDICT[review.verdict];
  // A stop and target for a stock the rules reject would read as a trade idea.
  const plan = review.verdict === "avoid" ? null : review.plan;
  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[16px_18px]" aria-label="Quy tắc Martin Luk">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[13px] font-semibold">Quy tắc Martin Luk</h2>
        <span className="rounded-full border border-app-border px-[8px] py-[1px] text-[10px] text-app-text-muted">swing · pullback EMA</span>
      </div>

      <div className="flex items-baseline justify-between gap-3">
        <span className="font-display text-[22px] font-bold" style={{ color: v.color }}>
          {v.label}
        </span>
        <span className="font-plex-mono text-[11.5px] text-app-text-muted">
          {review.passed}/{review.total} đạt
        </span>
      </div>
      <p className="m-0 text-[11px] text-app-text-muted">{v.hint}</p>

      {plan && (
        <div className="grid grid-cols-3 gap-2">
          {[
            { k: "Dừng lỗ", v: `${formatThousandsVN(plan.stop)} (−${formatVN(plan.stopPct, 1)}%)` },
            { k: `Mục tiêu ${plan.rewardRisk}R`, v: formatThousandsVN(plan.target) },
            { k: `Tỷ trọng (rủi ro ${formatVN(plan.riskPct, 1)}%)`, v: `${formatVN(plan.positionPct, 1)}% vốn` },
          ].map((x) => (
            <div key={x.k} className="flex min-w-0 flex-col gap-[3px] rounded-[11px] border border-app-hairline bg-app-surface-2 p-[8px_9px]">
              <span className="text-[9.5px] text-app-text-muted">{x.k}</span>
              <span className="font-plex-mono text-[11.5px] font-semibold">{x.v}</span>
            </div>
          ))}
        </div>
      )}
      {plan && (
        <p className="m-0 text-[10.5px] text-app-text-muted">
          Thoát: chốt dần ~10% khi tăng mạnh, phần còn lại giữ đến khi đóng cửa dưới EMA 9 ({formatThousandsVN(plan.trailEma9)}).
        </p>
      )}

      <details className="group">
        <summary className="cursor-pointer list-none text-[11.5px] font-medium text-app-accent">
          <span className="group-open:hidden">Xem {review.checks.length} quy tắc</span>
          <span className="hidden group-open:inline">Ẩn quy tắc</span>
        </summary>
        <div className="mt-2 flex flex-col gap-3">
          {GROUPS.map((g) => {
            const rows = review.checks.filter((c) => c.group === g.key);
            if (rows.length === 0) return null;
            return (
              <div key={g.key} className="flex flex-col gap-1">
                <span className="text-[10.5px] uppercase tracking-[0.04em] text-app-text-faint">{g.title}</span>
                <ul className="m-0 flex list-none flex-col p-0">
                  {rows.map((c) => (
                    <CheckRow key={c.key} c={c} />
                  ))}
                </ul>
              </div>
            );
          })}
          <ul className="m-0 flex list-disc flex-col gap-1 pl-4 text-[10.5px] leading-[1.45] text-app-text-muted">
            {review.notes.map((n) => (
              <li key={n}>{n}</li>
            ))}
          </ul>
        </div>
      </details>

      {review.aiAvailable && <LukAIReview symbol={review.symbol} />}

      <p className="m-0 text-[10px] leading-[1.45] text-app-text-faint">
        Dựa trên các quy tắc Martin Luk chia sẻ công khai (rủi ro 0,5%/lệnh, dừng lỗ 1,5–5%, tối đa 35%/vị thế), điều chỉnh cho
        thị trường Việt Nam. Chỉ để luyện tập mô phỏng, không phải khuyến nghị đầu tư.
      </p>
    </section>
  );
}
