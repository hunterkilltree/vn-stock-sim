"use client";

import Link from "next/link";
import { useActionState, useState } from "react";
import { registerAction, type AuthFormState } from "@/lib/authActions";
import { CRYPTO_CAPITAL_PRESET, STOCK_CAPITAL_PRESETS } from "@/lib/capital";

const initialState: AuthFormState = { error: null };

const POINTS = [
  "Chọn vốn ảo ban đầu — 1 tỷ ₫ cho cổ phiếu hoặc 10.000 USDT cho crypto.",
  "Danh mục, sổ giao dịch và điểm Replay gắn với tài khoản của bạn.",
  "Không thu phí, không tiền thật, không kết nối tài khoản chứng khoán.",
];

const MARKETS: { value: "stock" | "crypto" | "both"; label: string }[] = [
  { value: "stock", label: "Cổ phiếu" },
  { value: "crypto", label: "Crypto" },
  { value: "both", label: "Cả hai" },
];

// Decorative candle strip, same seeded generator as Signup.dc.html's
// renderVals() so it's stable across renders.
function buildCandles() {
  let seed = 51244;
  const r = () => {
    seed = (seed * 1103515245 + 12345) % 2147483648;
    return seed / 2147483648;
  };
  const n = 48;
  const step = 596 / n;
  const bw = step * 0.55;
  const data: { o: number; h: number; l: number; c: number }[] = [];
  let px = 100;
  for (let i = 0; i < n; i++) {
    const o = px;
    const c = o * (1 + ((r() - 0.44) * 2.4) / 100);
    const h = Math.max(o, c) * (1 + r() * 0.008);
    const l = Math.min(o, c) * (1 - r() * 0.008);
    data.push({ o, h, l, c });
    px = c;
  }
  const hi = Math.max(...data.map((d) => d.h));
  const lo = Math.min(...data.map((d) => d.l));
  const py = (v: number) => 8 + ((hi - v) / (hi - lo)) * 100;
  return data.map((d, i) => {
    const cx = i * step + step / 2;
    const top = py(Math.max(d.o, d.c));
    const bot = py(Math.min(d.o, d.c));
    return {
      col: d.c >= d.o ? "#35C77F" : "#FF5C5C",
      wick: `M${cx.toFixed(1)} ${py(d.h).toFixed(1)}V${py(d.l).toFixed(1)}`,
      x: cx - bw / 2,
      y: top,
      w: bw,
      h: Math.max(1.4, bot - top),
    };
  });
}
const CANDLES = buildCandles();

function choiceStyle(selected: boolean) {
  return {
    borderColor: selected ? "var(--app-accent)" : "var(--app-border)",
    background: selected ? "var(--app-accent-surface)" : "var(--app-surface)",
    color: selected ? "var(--app-text)" : "var(--app-text-3)",
  };
}

// Rebuilt against design/screens/Signup.dc.html (phase-g.md decisions
// 5-7, 13): the chosen capital opens the user's main portfolio, and the
// market interest is stored on the account.
export default function RegisterPage() {
  const [state, formAction, pending] = useActionState(registerAction, initialState);
  const [password, setPassword] = useState("");
  const [capital, setCapital] = useState(STOCK_CAPITAL_PRESETS[0].amount);
  const [market, setMarket] = useState<"stock" | "crypto" | "both">("both");

  const rules = [
    { label: "Từ 8 ký tự", ok: password.length >= 8 },
    { label: "Có chữ số", ok: /\d/.test(password) },
    { label: "Có chữ hoa", ok: /[A-ZÀ-Ỹ]/.test(password) },
  ];

  const inputCls =
    "h-12 rounded-xl border border-app-border bg-app-surface px-[14px] text-sm text-app-text outline-none placeholder:text-app-text-faint focus:border-app-border-strong";

  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <div className="hidden w-[740px] shrink-0 flex-col justify-between border-r border-app-border bg-app-chrome p-[64px_72px] lg:flex">
        <Link href="/stocks" className="flex items-center gap-3 text-app-text">
          <span className="flex h-[38px] w-[38px] shrink-0 items-center justify-center rounded-[11px] bg-app-accent">
            <svg width="21" height="21" viewBox="0 0 24 24" aria-hidden="true">
              <rect x="4" y="9" width="6" height="6" fill="#131210" />
              <rect x="14" y="9" width="6" height="8" fill="#131210" />
              <path d="M7 4v5M7 15v5M17 3v6M17 17v4" stroke="#131210" strokeWidth="2.2" strokeLinecap="round" />
            </svg>
          </span>
          <span className="flex flex-col gap-[2px]">
            <span className="font-display text-xl font-bold tracking-[-0.01em]">VN Stock Sim</span>
            <span className="text-[10.5px] uppercase tracking-[0.1em] text-app-text-muted">Mô phỏng · HOSE</span>
          </span>
        </Link>

        <div className="flex flex-col gap-[26px]">
          <h1 className="m-0 max-w-[520px] font-display text-[40px] font-bold leading-[1.2] tracking-[-0.02em]" style={{ textWrap: "pretty" }}>
            Một tài khoản, một danh mục giấy của riêng bạn.
          </h1>
          <p className="m-0 max-w-[470px] text-[14.5px] leading-[1.65] text-app-text-3" style={{ textWrap: "pretty" }}>
            Bạn chọn vốn ảo khi mở tài khoản. Từ đó mọi lệnh, mọi phiên Replay và mọi ghi chú đều thuộc về riêng bạn.
          </p>
          <div className="flex flex-col gap-[13px]">
            {POINTS.map((p) => (
              <div key={p} className="flex items-start gap-[11px]">
                <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#35C77F" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mt-[2px] shrink-0" aria-hidden="true">
                  <path d="M4 12l5 5L20 7" />
                </svg>
                <span className="text-[13.5px] leading-[1.5] text-app-text-2">{p}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-[18px]">
          <svg viewBox="0 0 596 120" width="596" height="120" fill="none" aria-hidden="true">
            <line x1="0" y1="119" x2="596" y2="119" stroke="#23231F" strokeWidth="1" />
            {CANDLES.map((c, i) => (
              <g key={i}>
                <path d={c.wick} stroke={c.col} strokeWidth="1.2" opacity="0.5" />
                <rect x={c.x.toFixed(1)} y={c.y.toFixed(1)} width={c.w.toFixed(1)} height={c.h.toFixed(1)} fill={c.col} opacity="0.5" rx="0.7" />
              </g>
            ))}
          </svg>
          <span className="text-[11.5px] text-app-text-faint">
            Không thu phí, không cần CMND/CCCD, không kết nối tài khoản chứng khoán thật.
          </span>
        </div>
      </div>

      <div className="flex min-w-0 flex-grow items-center justify-center p-[40px_48px]">
        <form action={formAction} className="flex w-[420px] max-w-full flex-col gap-[18px]">
          <div className="flex flex-col gap-[6px]">
            <h2 className="m-0 font-display text-[30px] font-bold tracking-[-0.015em]">Tạo tài khoản</h2>
            <p className="m-0 text-[13px] leading-[1.5] text-app-text-muted">Miễn phí. Đây là tài khoản mô phỏng — không cần thông tin thật.</p>
          </div>

          <div className="flex flex-col gap-[6px]">
            <label htmlFor="displayName" className="text-[12.5px] text-app-text-3">Tên hiển thị</label>
            <input id="displayName" name="displayName" type="text" required autoComplete="name" placeholder="Trần Đức" className={inputCls} />
          </div>

          <div className="flex flex-col gap-[6px]">
            <label htmlFor="email" className="text-[12.5px] text-app-text-3">Email</label>
            <input id="email" name="email" type="email" required autoComplete="email" placeholder="ban@email.com" className={inputCls} />
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="password" className="text-[12.5px] text-app-text-3">Mật khẩu</label>
            <input
              id="password"
              name="password"
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputCls}
            />
            <div className="flex gap-[14px]" aria-live="polite">
              {rules.map((r) => (
                <span key={r.label} className="flex items-center gap-[6px] text-[11.5px]" style={{ color: r.ok ? "var(--price-up)" : "var(--app-text-muted)" }}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d={r.ok ? "M4 12l5 5L20 7" : "M12 5v14M5 12h14"} />
                  </svg>
                  <span>{r.label}</span>
                </span>
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <span className="text-[12.5px] text-app-text-3">Vốn ảo ban đầu</span>
            <input type="hidden" name="startingCapital" value={capital} />
            <div className="flex gap-2">
              {STOCK_CAPITAL_PRESETS.map((c) => (
                <button
                  key={c.amount}
                  type="button"
                  aria-pressed={capital === c.amount}
                  onClick={() => setCapital(c.amount)}
                  className="flex h-14 flex-1 flex-col items-center justify-center gap-[3px] rounded-[11px] border px-[10px]"
                  style={choiceStyle(capital === c.amount)}
                >
                  <span className="font-plex-mono text-[13px] font-semibold">{c.label}</span>
                  <span className="text-[10.5px] text-app-text-muted">{c.note}</span>
                </button>
              ))}
              <button
                type="button"
                aria-pressed={capital === CRYPTO_CAPITAL_PRESET.amount}
                onClick={() => setCapital(CRYPTO_CAPITAL_PRESET.amount)}
                className="flex h-14 flex-1 flex-col items-center justify-center gap-[3px] rounded-[11px] border px-[10px]"
                style={choiceStyle(capital === CRYPTO_CAPITAL_PRESET.amount)}
              >
                <span className="font-plex-mono text-[13px] font-semibold">{CRYPTO_CAPITAL_PRESET.label}</span>
                <span className="text-[10.5px] text-app-text-muted">{CRYPTO_CAPITAL_PRESET.note}</span>
              </button>
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <span className="text-[12.5px] text-app-text-3">Thị trường bạn quan tâm</span>
            <input type="hidden" name="marketInterest" value={market} />
            <div className="flex gap-2">
              {MARKETS.map((m) => (
                <button
                  key={m.value}
                  type="button"
                  aria-pressed={market === m.value}
                  onClick={() => setMarket(m.value)}
                  className="h-11 flex-1 rounded-[11px] border text-[13px]"
                  style={{ ...choiceStyle(market === m.value), fontWeight: market === m.value ? 600 : 500 }}
                >
                  {m.label}
                </button>
              ))}
            </div>
          </div>

          <label className="flex cursor-pointer items-start gap-[10px]">
            <input type="checkbox" name="acknowledge" required className="mt-[2px] h-4 w-4 shrink-0 accent-[#E08A3C]" />
            <span className="text-[12.5px] leading-[1.5] text-app-text-3">
              Tôi hiểu đây là công cụ mô phỏng, không phải dịch vụ đầu tư và không có tiền thật.
            </span>
          </label>

          {state.error && (
            <p role="alert" className="m-0 rounded-[10px] border border-price-down/40 bg-price-down/10 px-3 py-2 text-[13px] text-price-down">
              {state.error}
            </p>
          )}

          <button
            type="submit"
            disabled={pending}
            className="h-[52px] rounded-xl bg-app-accent text-[15px] font-semibold text-app-accent-ink disabled:opacity-60"
          >
            {pending ? "Đang tạo tài khoản…" : "Tạo tài khoản"}
          </button>

          <div className="flex justify-center gap-[7px] border-t border-app-hairline pt-1 text-[13.5px] text-app-text-muted">
            <span>Đã có tài khoản?</span>
            <Link href="/login" className="font-semibold text-app-accent hover:text-app-accent-hover">
              Đăng nhập
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
