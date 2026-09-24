"use client";

import { useState } from "react";
import WillBadge from "@/components/WillBadge";
import { formatVN } from "@/lib/format";
import {
  DEFAULT_SETTINGS,
  connectionOf,
  currentMonth,
  lastTestStore,
  settingsStore,
  usageStore,
  type QuantSettings,
} from "@/lib/quantSettings";
import type { QuantError, QuantProvider, QuantTestResponse } from "@/lib/quantTypes";

// Claude tiers from Settings-AI.dc.html's model select, mapped to current
// model IDs (phase-h.md decision 5). OpenAI and self-hosted models come
// only from the key's own model list.
const CLAUDE_TIERS = [
  { id: "claude-opus-5", label: "Bản mạnh nhất — suy luận sâu (khuyến nghị)" },
  { id: "claude-sonnet-5", label: "Bản cân bằng — nhanh và rẻ hơn" },
  { id: "claude-haiku-4-5", label: "Bản nhẹ — chỉ lọc và tóm tắt" },
];

const PROVIDERS: { id: QuantProvider | "cloud"; name: string; note: string }[] = [
  { id: "cloud", name: "Quant Cloud", note: "Máy chủ của VN Stock Sim — chưa mở: cần quyết định của nền tảng về chi phí" },
  { id: "claude", name: "Claude (khoá của bạn)", note: "Bạn trả phí trực tiếp cho nhà cung cấp" },
  { id: "openai", name: "OpenAI (khoá của bạn)", note: "Bạn trả phí trực tiếp cho nhà cung cấp" },
  { id: "custom", name: "Máy chủ riêng", note: "Endpoint tương thích OpenAI do bạn tự vận hành" },
];

const SCOPES: { key: keyof QuantSettings["scope"]; label: string; note: string }[] = [
  { key: "prices", label: "Giá và khối lượng (OHLCV)", note: "Giá, % thay đổi và KLTB20 của các mã trong ứng dụng" },
  { key: "indicators", label: "Chỉ báo kỹ thuật", note: "RSI(14) và vị trí giá so với SMA 20/50/200" },
  { key: "watchlist", label: "Danh sách theo dõi của bạn", note: "Để Quant ưu tiên mã bạn quan tâm" },
  { key: "positions", label: "Số dư và vị thế giấy", note: "Chỉ bật nếu muốn Quant gợi ý theo danh mục" },
];

function tempLabel(t: number): string {
  const mood = t <= 0.3 ? "chặt chẽ" : t >= 0.7 ? "sáng tạo" : "cân bằng";
  return `${formatVN(t, 1)} · ${mood}`;
}

function countChanges(a: QuantSettings, b: QuantSettings): number {
  let n = 0;
  if (a.provider !== b.provider) n++;
  for (const p of ["claude", "openai", "custom"] as const) {
    if (a.apiKeys[p] !== b.apiKeys[p]) n++;
    if (a.models[p] !== b.models[p]) n++;
  }
  if (a.baseUrl !== b.baseUrl) n++;
  if (a.timeoutSeconds !== b.timeoutSeconds) n++;
  if (a.temperature !== b.temperature) n++;
  if (a.autoBacktest !== b.autoBacktest) n++;
  for (const k of Object.keys(a.scope) as (keyof QuantSettings["scope"])[]) {
    if (a.scope[k] !== b.scope[k]) n++;
  }
  return n;
}

function Toggle({ on, disabled, onChange, label }: { on: boolean; disabled?: boolean; onChange?: (v: boolean) => void; label: string }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange?.(!on)}
      className="relative h-6 w-[42px] shrink-0 rounded-xl disabled:cursor-not-allowed disabled:opacity-50"
      style={{ background: on ? "var(--app-accent)" : "var(--app-border)" }}
    >
      <span
        className="absolute top-[3px] h-[18px] w-[18px] rounded-full"
        style={{ left: on ? 21 : 3, background: on ? "var(--app-accent-ink)" : "var(--app-text-muted)" }}
      />
    </button>
  );
}

const card = "flex flex-col rounded-[14px] border border-app-border bg-app-surface p-[18px]";
const inputCls =
  "h-[46px] min-w-0 rounded-[11px] border border-app-border bg-app-surface-2 px-[14px] text-[13px] text-app-text outline-none placeholder:text-app-text-faint focus:border-app-border-strong";

// Settings-AI.dc.html (phase-h.md). The parent remounts this with a new
// `key` whenever the saved settings change, so the draft always starts
// from what is actually stored.
function Form({ saved }: { saved: QuantSettings }) {
  const [draft, setDraft] = useState<QuantSettings>(saved);
  const [showKey, setShowKey] = useState(false);
  const [testing, setTesting] = useState(false);
  const [test, setTest] = useState<QuantTestResponse | null>(null);
  const [testError, setTestError] = useState<string | null>(null);
  const lastTest = lastTestStore.useValue();
  const usage = usageStore.useValue();

  const changes = countChanges(draft, saved);
  const p = draft.provider;
  const listedModels = lastTest && lastTest.ok && lastTest.provider === p ? lastTest.models : [];
  const canTest = p === "custom" ? draft.baseUrl.trim() !== "" && draft.models.custom.trim() !== "" : draft.apiKeys[p].trim() !== "" && draft.models[p].trim() !== "";

  function update(patch: Partial<QuantSettings>) {
    setDraft((d) => ({ ...d, ...patch }));
  }

  async function runTest() {
    setTesting(true);
    setTestError(null);
    setTest(null);
    try {
      const res = await fetch("/api/quant/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(connectionOf(draft)),
      });
      const payload = await res.json().catch(() => null);
      if (!res.ok) {
        const err = payload as QuantError | null;
        const status = err?.providerStatus ? `${err.providerStatus}: ` : "";
        setTestError(`${status}${err?.message ?? "Kiểm tra thất bại."}`);
        lastTestStore.write({ at: new Date().toISOString(), ok: false, provider: p, model: draft.models[p], models: [], temperatureApplied: false });
        return;
      }
      const t = payload as QuantTestResponse;
      setTest(t);
      lastTestStore.write({
        at: new Date().toISOString(),
        ok: true,
        provider: p,
        model: draft.models[p],
        models: t.models,
        temperatureApplied: t.temperatureApplied,
      });
    } catch {
      setTestError("Không thể kết nối tới máy chủ.");
    } finally {
      setTesting(false);
    }
  }

  const statusChip = lastTest
    ? {
        color: lastTest.ok ? "var(--price-up)" : "var(--price-down)",
        text: `${lastTest.ok ? "Đã kết nối" : "Kết nối lỗi"} · kiểm tra ${new Date(lastTest.at).toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit" })}`,
      }
    : { color: "var(--app-text-faint)", text: "Chưa kiểm tra kết nối" };

  const monthUsage = usage && usage.month === currentMonth() ? usage : null;
  const monthNumber = Number(currentMonth().slice(5));

  return (
    <>
      <header className="flex items-end justify-between gap-6">
        <div className="flex flex-col gap-[5px]">
          <h2 className="m-0 font-display text-[25px] font-bold tracking-[-0.015em]">Mô hình AI</h2>
          <span className="text-[12.5px] text-app-text-muted">Chọn mô hình chạy Trợ lý Quant và quyết định dữ liệu nào được gửi đi</span>
        </div>
        <span className="flex h-[34px] items-center gap-[7px] rounded-[9px] border border-app-border bg-app-surface px-3 text-[11.5px] text-app-text-3">
          <span className="h-[7px] w-[7px] rounded-full" style={{ background: statusChip.color }} />
          <span>{statusChip.text}</span>
        </span>
      </header>

      <div className="flex min-h-0 flex-grow gap-5">
        <div className="flex min-w-0 flex-grow flex-col gap-4">
          <section className={`${card} gap-[13px]`}>
            <div className="flex flex-col gap-[3px]">
              <h3 className="m-0 text-[15px] font-semibold">Nhà cung cấp mô hình</h3>
              <span className="text-[11.5px] text-app-text-muted">Cắm khoá API của riêng bạn — mỗi câu hỏi được tính phí trực tiếp vào khoá đó</span>
            </div>
            <div role="radiogroup" aria-label="Nhà cung cấp mô hình" className="grid grid-cols-2 gap-[10px]">
              {PROVIDERS.map((prov) => {
                const disabled = prov.id === "cloud";
                const checked = prov.id === p;
                return (
                  <label
                    key={prov.id}
                    className="flex items-start gap-[11px] rounded-xl border p-[13px_14px]"
                    style={{
                      borderColor: checked ? "var(--app-accent)" : "var(--app-border)",
                      background: checked ? "var(--app-accent-surface)" : "#1A1A17",
                      cursor: disabled ? "not-allowed" : "pointer",
                      opacity: disabled ? 0.55 : 1,
                    }}
                  >
                    <input
                      type="radio"
                      name="provider"
                      checked={checked}
                      disabled={disabled}
                      onChange={() => !disabled && update({ provider: prov.id as QuantProvider })}
                      className="mt-[2px] h-[17px] w-[17px] shrink-0 accent-[#E08A3C]"
                    />
                    <span className="flex flex-col gap-[3px]">
                      <span className="text-[13px] font-semibold">{prov.name}</span>
                      <span className="text-[11.5px] leading-[1.4] text-app-text-muted">{prov.note}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          </section>

          <section className={`${card} gap-[14px]`}>
            <h3 className="m-0 text-[15px] font-semibold">Kết nối</h3>

            {p === "custom" && (
              <div className="flex flex-col gap-[7px]">
                <label htmlFor="baseurl" className="text-xs text-app-text-3">Địa chỉ endpoint (tương thích OpenAI)</label>
                <input
                  id="baseurl"
                  type="url"
                  placeholder="https://llm.example.com/v1"
                  value={draft.baseUrl}
                  onChange={(e) => update({ baseUrl: e.target.value })}
                  className={`${inputCls} font-plex-mono`}
                />
              </div>
            )}

            <div className="flex flex-col gap-[7px]">
              <label htmlFor="apikey" className="text-xs text-app-text-3">
                Khoá API{p === "custom" && " (nếu máy chủ của bạn yêu cầu)"}
              </label>
              <div className="flex gap-[10px]">
                <input
                  id="apikey"
                  type={showKey ? "text" : "password"}
                  autoComplete="off"
                  spellCheck={false}
                  placeholder={p === "claude" ? "sk-ant-…" : p === "openai" ? "sk-…" : "tuỳ chọn"}
                  value={draft.apiKeys[p]}
                  onChange={(e) => update({ apiKeys: { ...draft.apiKeys, [p]: e.target.value } })}
                  className={`${inputCls} flex-grow font-plex-mono`}
                />
                <button
                  type="button"
                  onClick={() => setShowKey((v) => !v)}
                  aria-label={showKey ? "Ẩn khoá" : "Hiện khoá"}
                  className="h-[46px] rounded-[11px] border border-app-border px-3 text-xs text-app-text-3"
                >
                  {showKey ? "Ẩn" : "Hiện"}
                </button>
                <button
                  type="button"
                  onClick={runTest}
                  disabled={!canTest || testing}
                  className="h-[46px] rounded-[11px] border border-app-border-strong bg-app-border px-[15px] text-[13px] font-semibold text-app-text disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {testing ? "Đang kiểm tra…" : "Kiểm tra"}
                </button>
              </div>
              <span className="flex items-start gap-[7px] text-[11.5px] leading-[1.5] text-app-text-muted">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#8A867E" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" className="mt-[2px] shrink-0" aria-hidden="true">
                  <path d="M6 11V8a6 6 0 1112 0v3M5 11h14v9H5z" />
                </svg>
                <span>
                  Khoá chỉ được lưu trong trình duyệt này. Mỗi câu hỏi, khoá được gửi kèm qua máy chủ VN Stock Sim tới nhà cung cấp và không
                  bao giờ được lưu hay ghi log ở máy chủ.
                </span>
              </span>
            </div>

            <div className="grid grid-cols-[1.4fr_1fr] gap-3">
              <div className="flex flex-col gap-[7px]">
                <label htmlFor="model" className="text-xs text-app-text-3">Mô hình</label>
                {p === "claude" ? (
                  <select
                    id="model"
                    value={draft.models.claude}
                    onChange={(e) => update({ models: { ...draft.models, claude: e.target.value } })}
                    className={inputCls}
                  >
                    <optgroup label="Khuyến nghị">
                      {CLAUDE_TIERS.map((t) => (
                        <option key={t.id} value={t.id}>
                          {t.label}
                        </option>
                      ))}
                    </optgroup>
                    {listedModels.filter((m) => !CLAUDE_TIERS.some((t) => t.id === m)).length > 0 && (
                      <optgroup label="Tất cả mô hình từ khoá của bạn">
                        {listedModels
                          .filter((m) => !CLAUDE_TIERS.some((t) => t.id === m))
                          .map((m) => (
                            <option key={m} value={m}>
                              {m}
                            </option>
                          ))}
                      </optgroup>
                    )}
                    {!CLAUDE_TIERS.some((t) => t.id === draft.models.claude) && !listedModels.includes(draft.models.claude) && (
                      <option value={draft.models.claude}>{draft.models.claude}</option>
                    )}
                  </select>
                ) : (
                  <>
                    <input
                      id="model"
                      list="model-options"
                      placeholder={listedModels.length ? "Chọn từ danh sách" : "Nhập tên mô hình"}
                      value={draft.models[p]}
                      onChange={(e) => update({ models: { ...draft.models, [p]: e.target.value } })}
                      className={`${inputCls} font-plex-mono`}
                    />
                    <datalist id="model-options">
                      {listedModels.map((m) => (
                        <option key={m} value={m} />
                      ))}
                    </datalist>
                  </>
                )}
              </div>
              <div className="flex flex-col gap-[7px]">
                <label htmlFor="timeout" className="text-xs text-app-text-3">Thời gian chờ</label>
                <select
                  id="timeout"
                  value={draft.timeoutSeconds}
                  onChange={(e) => update({ timeoutSeconds: Number(e.target.value) })}
                  className={inputCls}
                >
                  {[30, 60, 120].map((s) => (
                    <option key={s} value={s}>
                      {s} giây
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <span className="text-[11.5px] text-app-text-muted">
              {listedModels.length
                ? `Đã tải ${listedModels.length} mô hình từ khoá của bạn ở lần kiểm tra gần nhất.`
                : "Danh sách mô hình lấy trực tiếp từ khoá của bạn sau khi kiểm tra kết nối."}
            </span>
          </section>

          <section className={`${card} flex-grow gap-[15px]`}>
            <h3 className="m-0 text-[15px] font-semibold">Cách Quant trả lời</h3>
            <div className="flex flex-col gap-[9px]">
              <div className="flex items-baseline justify-between">
                <label htmlFor="temp" className="text-[12.5px] text-app-text-2">Mức sáng tạo</label>
                <span className="font-plex-mono text-xs text-app-accent">{tempLabel(draft.temperature)}</span>
              </div>
              <input
                id="temp"
                type="range"
                min={0}
                max={100}
                value={Math.round(draft.temperature * 100)}
                onChange={(e) => update({ temperature: Number(e.target.value) / 100 })}
                className="h-[6px] w-full cursor-pointer accent-[#E08A3C]"
              />
              <span className="text-[11.5px] text-app-text-muted">
                Mức thấp cho quy tắc giao dịch lặp lại được; mức cao hợp với việc gợi ý ý tưởng.
                {lastTest?.ok && lastTest.provider === p && !lastTest.temperatureApplied &&
                  " Mô hình đang chọn không hỗ trợ tham số này nên sẽ bỏ qua nó."}
              </span>
            </div>

            <div className="flex flex-col gap-[10px] border-t border-app-hairline pt-[14px]">
              <div className="flex items-center gap-[13px]">
                <Toggle on={draft.autoBacktest} onChange={(v) => update({ autoBacktest: v })} label="Tự động kiểm thử quy tắc Quant sinh ra" />
                <span className="flex min-w-0 flex-grow flex-col gap-[2px]">
                  <span className="text-[13px] font-medium">Tự động kiểm thử quy tắc Quant sinh ra</span>
                  <span className="text-[11.5px] leading-[1.4] text-app-text-muted">Chạy backtest ngay sau khi có chiến lược mới</span>
                </span>
              </div>
              <div className="flex items-center gap-[13px]">
                <Toggle on={false} disabled label="Cho phép Quant đặt lệnh trên tài khoản giấy" />
                <span className="flex min-w-0 flex-grow flex-col gap-[2px]">
                  <span className="flex items-center gap-2 text-[13px] font-medium">
                    Cho phép Quant đặt lệnh trên tài khoản giấy <WillBadge />
                  </span>
                  <span className="text-[11.5px] leading-[1.4] text-app-text-muted">Quant tự khớp lệnh mô phỏng mà không hỏi lại — chưa hỗ trợ</span>
                </span>
              </div>
            </div>
          </section>
        </div>

        <div className="flex w-[348px] shrink-0 flex-col gap-4">
          <section className={`${card} gap-[13px]`}>
            <div className="flex flex-col gap-[3px]">
              <h3 className="m-0 text-[15px] font-semibold">Dữ liệu gửi cho mô hình</h3>
              <span className="text-[11.5px] text-app-text-muted">Câu hỏi luôn được gửi; chỉ những mục bật mới được gửi kèm</span>
            </div>
            <div className="flex flex-col gap-[11px]">
              {SCOPES.map((s) => (
                <label key={s.key} className="flex cursor-pointer items-start gap-[11px]">
                  <input
                    type="checkbox"
                    checked={draft.scope[s.key]}
                    onChange={(e) => update({ scope: { ...draft.scope, [s.key]: e.target.checked } })}
                    className="mt-[1px] h-[17px] w-[17px] shrink-0 accent-[#E08A3C]"
                  />
                  <span className="flex flex-col gap-[2px]">
                    <span className="text-[12.5px] font-medium" style={{ color: draft.scope[s.key] ? "var(--app-text)" : "var(--app-text-3)" }}>
                      {s.label}
                    </span>
                    <span className="text-[11px] leading-[1.4] text-app-text-muted">{s.note}</span>
                  </span>
                </label>
              ))}
              <label className="flex cursor-not-allowed items-start gap-[11px] opacity-60">
                <input type="checkbox" disabled checked={false} className="mt-[1px] h-[17px] w-[17px] shrink-0" />
                <span className="flex flex-col gap-[2px]">
                  <span className="text-[12.5px] font-medium text-app-text-3">Sổ giao dịch và ghi chú riêng</span>
                  <span className="text-[11px] leading-[1.4] text-app-text-muted">Ứng dụng chưa có ghi chú riêng cho từng lệnh</span>
                </span>
              </label>
            </div>
          </section>

          <section className={`${card} gap-3`}>
            <div className="flex items-baseline justify-between">
              <h3 className="m-0 text-[15px] font-semibold">Mức dùng tháng {monthNumber}</h3>
              <span className="font-plex-mono text-xs text-app-text-3">{formatVN(monthUsage?.tokens ?? 0, 0)} token</span>
            </div>
            <div className="flex justify-between text-[11.5px] text-app-text-muted">
              <span>
                {monthUsage?.questions ?? 0} câu hỏi · {monthUsage?.strategies ?? 0} chiến lược
              </span>
              <span>Đếm trên trình duyệt này</span>
            </div>
          </section>

          <section className={`${card} flex-grow gap-3`}>
            <h3 className="m-0 text-[15px] font-semibold">Thử trước khi lưu</h3>
            <div className="flex flex-col gap-2 rounded-[11px] border border-app-hairline bg-[#1A1A17] p-[12px_13px]">
              <span className="text-[12.5px] leading-[1.5] text-app-text-2">“Lọc HOSE có RSI dưới 35 và ROE trên 15%.”</span>
              <div className="flex items-start gap-[9px] border-t border-app-hairline pt-[9px]" aria-live="polite">
                {test ? (
                  <>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#35C77F" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mt-[2px] shrink-0" aria-hidden="true">
                      <path d="M4 12l5 5L20 7" />
                    </svg>
                    <span className="text-[11.5px] text-app-text-3">
                      Hiểu đúng {test.understood}/{test.expected} điều kiện · {formatVN(test.latencyMs / 1000, 1)} giây ·{" "}
                      {formatVN(test.usage.inputTokens + test.usage.outputTokens, 0)} token
                      {test.labels.length > 0 && <span className="block text-app-text-muted">{test.labels.join(" · ")}</span>}
                    </span>
                  </>
                ) : testError ? (
                  <span className="text-[11.5px] text-price-down">{testError}</span>
                ) : (
                  <span className="text-[11.5px] text-app-text-muted">{testing ? "Đang gửi câu hỏi mẫu…" : "Chưa chạy thử với cấu hình này."}</span>
                )}
              </div>
            </div>
            <button
              type="button"
              onClick={runTest}
              disabled={!canTest || testing}
              className="h-11 rounded-[11px] border border-app-border-strong bg-app-border text-[13px] font-semibold text-app-text disabled:cursor-not-allowed disabled:opacity-50"
            >
              {test || testError ? "Chạy thử lại" : "Chạy thử"}
            </button>
            <div className="flex-grow" />
            <div className="flex items-start gap-[9px] rounded-[10px] border border-app-warn-border bg-app-warn-surface p-[11px_12px]">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#F0C243" strokeWidth="1.8" strokeLinecap="round" className="mt-[1px] shrink-0" aria-hidden="true">
                <path d="M12 9v5M12 17h0M12 3l9 17H3L12 3z" />
              </svg>
              <p className="m-0 text-[11.5px] leading-[1.5] text-app-warn-text">
                Dù dùng mô hình nào, Quant cũng không đặt lệnh thật. Mọi quyền ở đây chỉ áp dụng cho tài khoản giấy.
              </p>
            </div>
          </section>
        </div>
      </div>

      <footer className="flex shrink-0 items-center justify-between gap-5 border-t border-app-hairline pt-4">
        <span className="flex items-center gap-2 text-[12.5px]" style={{ color: changes ? "var(--price-ref)" : "var(--app-text-muted)" }}>
          <span className="h-[7px] w-[7px] rounded-full" style={{ background: changes ? "var(--price-ref)" : "var(--app-border-strong)" }} />
          <span>{changes ? `Có ${changes} thay đổi chưa lưu` : "Không có thay đổi"}</span>
        </span>
        <div className="flex gap-[10px]">
          <button
            type="button"
            disabled={!changes}
            onClick={() => setDraft(saved)}
            className="h-[46px] rounded-[11px] border border-app-border bg-app-surface px-[18px] text-[13.5px] font-medium text-app-text-3 disabled:opacity-50"
          >
            Hoàn tác
          </button>
          <button
            type="button"
            disabled={!changes}
            onClick={() => settingsStore.write(draft)}
            className="h-[46px] rounded-[11px] bg-app-accent px-[22px] text-[13.5px] font-semibold text-app-accent-ink disabled:opacity-50"
          >
            Lưu thay đổi
          </button>
        </div>
      </footer>
    </>
  );
}

export default function AISettingsForm() {
  const saved = settingsStore.useValue();
  // Remount the form when the stored settings change (after hydration and
  // after every save) so the draft never goes stale.
  return <Form key={saved === DEFAULT_SETTINGS ? "defaults" : JSON.stringify(saved)} saved={saved} />;
}
