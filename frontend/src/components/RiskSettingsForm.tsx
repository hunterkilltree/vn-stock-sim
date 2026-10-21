"use client";

import { useActionState, useState } from "react";
import { saveRiskSettingsAction, type RiskSettingsState } from "@/lib/riskActions";
import type { RiskMode, RiskSettings } from "@/lib/api";

const MODES: { id: RiskMode; name: string; note: string }[] = [
  { id: "coach", name: "Huấn luyện", note: "Cảnh báo và ghi lại vi phạm, lệnh vẫn được đặt" },
  { id: "strict", name: "Nghiêm ngặt", note: "Chặn lệnh mua thiếu điểm cắt lỗ hoặc rủi ro vượt mức tối đa" },
];

const FIELDS: { name: keyof Omit<RiskSettings, "mode">; label: string; note: string; step: string; min: number; max: number }[] = [
  { name: "riskPercent", label: "Rủi ro mỗi lệnh (% vốn)", note: "Khối lượng gợi ý = vốn × % ÷ (giá mua − cắt lỗ), làm tròn xuống lô 100", step: "0.1", min: 0.1, max: 10 },
  { name: "maxRiskPercent", label: "Rủi ro tối đa (% vốn)", note: "Quy tắc 2% của Elder: không lệnh nào được mất quá mức này", step: "0.1", min: 0.1, max: 10 },
  { name: "minRewardRisk", label: "Tỷ lệ lời/lỗ tối thiểu", note: "(Mục tiêu − giá mua) ÷ (giá mua − cắt lỗ); thấp hơn chỉ cảnh báo", step: "0.1", min: 0, max: 10 },
];

export default function RiskSettingsForm({ portfolioId, portfolioName, initial }: { portfolioId: string; portfolioName: string; initial: RiskSettings }) {
  const [state, formAction, pending] = useActionState<RiskSettingsState, FormData>(saveRiskSettingsAction, { error: null, saved: false });
  const [mode, setMode] = useState<RiskMode>(initial.mode);

  return (
    <form action={formAction} className="flex max-w-[640px] flex-col gap-[14px]">
      <input type="hidden" name="portfolioId" value={portfolioId} />
      <input type="hidden" name="mode" value={mode} />

      <section className="flex flex-col gap-3 rounded-[14px] border border-app-border bg-app-surface p-[18px]">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <h3 className="m-0 text-[15px] font-semibold">Chế độ áp dụng</h3>
          <span className="text-[11.5px] text-app-text-muted">Danh mục: {portfolioName}</span>
        </div>
        <div role="radiogroup" aria-label="Chế độ áp dụng" className="grid gap-2 sm:grid-cols-2">
          {MODES.map((m) => (
            <button
              key={m.id}
              type="button"
              role="radio"
              aria-checked={mode === m.id}
              onClick={() => setMode(m.id)}
              className="flex flex-col gap-1 rounded-[12px] border p-[12px_14px] text-left"
              style={
                mode === m.id
                  ? { borderColor: "var(--app-accent)", background: "var(--app-surface-2)" }
                  : { borderColor: "var(--app-border)", background: "var(--app-surface-2)" }
              }
            >
              <span className="text-[13.5px] font-semibold text-app-text">{m.name}</span>
              <span className="text-[11.5px] leading-[1.45] text-app-text-muted">{m.note}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-4 rounded-[14px] border border-app-border bg-app-surface p-[18px]">
        <h3 className="m-0 text-[15px] font-semibold">Ngưỡng rủi ro</h3>
        {FIELDS.map((f) => (
          <div key={f.name} className="flex flex-col gap-[6px]">
            <label htmlFor={f.name} className="text-[12.5px] text-app-text-2">
              {f.label}
            </label>
            <input
              id={f.name}
              name={f.name}
              type="number"
              inputMode="decimal"
              step={f.step}
              min={f.min}
              max={f.max}
              required
              defaultValue={initial[f.name]}
              className="box-border h-11 max-w-[200px] rounded-[10px] border border-app-border bg-app-surface-2 px-3 font-plex-mono text-sm text-app-text outline-none"
            />
            <span className="text-[11.5px] text-app-text-muted">{f.note}</span>
          </div>
        ))}
        <p className="m-0 text-[11px] leading-[1.5] text-app-text-muted">
          Đây là nguyên tắc quản trị rủi ro phổ biến, không phải cam kết lợi nhuận. Lệnh bán không bị kiểm tra.
        </p>
      </section>

      {state.error && <p className="m-0 rounded-xl border border-app-warn-border bg-app-warn-surface p-3 text-[12.5px] text-app-warn-text">{state.error}</p>}
      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={pending}
          className="h-11 rounded-[11px] bg-app-accent px-6 text-[13.5px] font-semibold text-app-accent-ink disabled:opacity-60"
        >
          {pending ? "Đang lưu..." : "Lưu"}
        </button>
        {state.saved && !pending && <span className="text-[12.5px] text-price-up">Đã lưu</span>}
      </div>
    </form>
  );
}
