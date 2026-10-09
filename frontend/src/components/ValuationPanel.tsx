import type { Valuation, ValuationMethod } from "@/lib/api";
import { formatThousandsVN, formatVN, signVN } from "@/lib/format";

// "Vùng giá mua": today's buy price per fair-value rule, snapped to the
// exchange's price grid, with the most conservative one highlighted
// (phase-valuation.md, stage V-1).
const METHOD: Record<ValuationMethod, { label: string; note: string }> = {
  graham_number: { label: "Graham Number", note: "√(22,5 × EPS × BVPS)" },
  graham_formula: { label: "Graham (1974)", note: "EPS × (8,5 + 2g) × 4,4 / Y" },
  lynch_fair_value: { label: "Peter Lynch", note: "P/E hợp lý = tăng trưởng (PEG = 1)" },
  rule1_sticker: { label: "Rule #1", note: "Giá mục tiêu 10 năm chiết khấu 15%" },
  weiss_yield: { label: "Weiss (cổ tức)", note: "Tỷ suất cổ tức gần đỉnh lịch sử" },
  pe_band: { label: "Dải P/E", note: "EPS × (P/E trung bình − 1σ)" },
};

export default function ValuationPanel({ valuation }: { valuation: Valuation }) {
  const v = valuation;
  return (
    <div className="flex flex-col gap-2 border-t border-app-hairline pt-3">
      <div className="flex items-center justify-between">
        <span className="text-[12px] font-semibold">Vùng giá mua</span>
        {v.buyPrice !== undefined && (
          <span
            className={`rounded-full px-[8px] py-[1px] text-[10px] font-medium ${
              v.inBuyZone ? "bg-price-up/15 text-price-up" : "bg-app-surface-3 text-app-text-muted"
            }`}
          >
            {v.inBuyZone ? "Đang trong vùng mua" : "Chưa tới vùng mua"}
          </span>
        )}
      </div>

      {v.buyPrice !== undefined ? (
        <div className="grid grid-cols-3 gap-2">
          {[
            { k: "Giá mua ≤", val: formatThousandsVN(v.buyPrice) },
            { k: "Giá trị hợp lý", val: v.fairValue !== undefined ? formatThousandsVN(v.fairValue) : "—" },
            { k: "So với giá hiện tại", val: v.upsidePct !== undefined ? `${signVN(v.upsidePct, 1)}%` : "—" },
          ].map((x) => (
            <div key={x.k} className="flex flex-col gap-[2px]">
              <span className="text-[9.5px] text-app-text-muted">{x.k}</span>
              <span className="font-plex-mono text-[12.5px] font-semibold">{x.val}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="m-0 text-[11px] text-app-text-muted">Chưa phương pháp nào áp dụng được cho mã này.</p>
      )}

      {!v.quality.passed && (
        <p className="m-0 rounded-[9px] border border-app-warn-border bg-app-warn-surface px-[10px] py-[7px] text-[11px] leading-[1.45] text-app-warn-text">
          Không qua bộ lọc chất lượng ({v.quality.reason}): giá rẻ theo công thức có thể là bẫy giá trị.
        </p>
      )}

      <ul className="m-0 flex list-none flex-col p-0">
        {v.values.map((m) => (
          <li key={m.method} className="flex items-start justify-between gap-3 border-b border-app-hairline py-[6px] last:border-b-0">
            <div className="flex min-w-0 flex-col">
              <span className={`text-[11.5px] ${m.method === v.buyMethod ? "font-semibold text-app-text" : "text-app-text-2"}`}>
                {METHOD[m.method].label}
              </span>
              <span className="text-[10px] text-app-text-faint">{METHOD[m.method].note}</span>
            </div>
            {m.applicable ? (
              <div className="flex shrink-0 flex-col items-end font-plex-mono text-[10.5px]">
                <span className="text-app-text">≤ {formatThousandsVN(m.buyPrice ?? 0)}</span>
                <span className="text-app-text-muted">
                  hợp lý {formatThousandsVN(m.fairValue ?? 0)}
                  {m.marginOfSafety > 0 && ` · AT ${formatVN(m.marginOfSafety * 100, 0)}%`}
                </span>
              </div>
            ) : (
              <span className="max-w-[45%] shrink-0 text-right text-[10px] leading-[1.35] text-app-text-faint">{m.reason}</span>
            )}
          </li>
        ))}
      </ul>
      <p className="m-0 text-[9.5px] leading-[1.4] text-app-text-faint">
        Giá mua = giá trị hợp lý × (1 − biên an toàn, AT), làm tròn xuống theo bước giá; chọn mức thấp nhất. Lợi suất TPCP 10 năm{" "}
        {formatVN(v.bondYieldPct, 2)}% ({v.bondYieldAsOf}).
      </p>
    </div>
  );
}
