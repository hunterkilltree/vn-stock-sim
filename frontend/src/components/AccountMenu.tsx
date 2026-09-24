"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, useTransition } from "react";
import WillBadge from "@/components/WillBadge";
import { createPortfolioAction, setActivePortfolioAction } from "@/lib/accountActions";
import { logoutAction } from "@/lib/authActions";
import { STOCK_CAPITAL_PRESETS } from "@/lib/capital";
import { signVN } from "@/lib/format";

export type MenuPortfolio = { id: string; name: string; nav: string; pct: number; active: boolean };

type Props = {
  placement: "below" | "right";
  user: { name: string; email: string; initials: string };
  portfolios: MenuPortfolio[];
};

const PROFILE_D = "M12 12a4 4 0 100-8 4 4 0 000 8zM5 20a7 7 0 0114 0";
const SETTINGS_D =
  "M12 15a3 3 0 100-6 3 3 0 000 6zM19.4 15a1.6 1.6 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.6 1.6 0 00-2.7 1.1V21a2 2 0 11-4 0v-.1A1.6 1.6 0 005.9 19.7l-.1.1a2 2 0 11-2.8-2.8l.1-.1A1.6 1.6 0 003 14.6a2 2 0 010-4h.1A1.6 1.6 0 004.3 8L4.2 8a2 2 0 112.8-2.8l.1.1a1.6 1.6 0 002.7-1.1V4a2 2 0 114 0v.1A1.6 1.6 0 0017 5.3l.1-.1A2 2 0 1119.9 8l-.1.1a1.6 1.6 0 001.1 2.7H21a2 2 0 010 4h-.1a1.6 1.6 0 00-1.5 1.2z";
const HELP_D = "M12 21a9 9 0 100-18 9 9 0 000 18zM9.5 9.5a2.5 2.5 0 114 2c-.9.7-1.5 1.2-1.5 2.5M12 17.5h0";

function LinkIcon({ d }: { d: string }) {
  return (
    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#8A867E" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={d} />
    </svg>
  );
}

// Avatar popover from design/screens/Account-Menu.dc.html -- profile
// header, the multi-portfolio switcher, create-portfolio, links, logout.
// See phase-g.md decisions 1, 8-10.
export default function AccountMenu({ placement, user, portfolios }: Props) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [capital, setCapital] = useState(STOCK_CAPITAL_PRESETS[0].amount);
  const [error, setError] = useState<string | null>(null);
  const [pending, startTransition] = useTransition();
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onDown(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  function toggle() {
    setOpen((o) => !o);
    setCreating(false);
    setError(null);
  }

  // Each row links to the Portfolio page, as in the design; choosing a
  // row also makes it the active portfolio for the whole app.
  function selectPortfolio(id: string) {
    setError(null);
    startTransition(async () => {
      const res = await setActivePortfolioAction(id);
      if (res.error) {
        setError(res.error);
        return;
      }
      setOpen(false);
      router.push("/portfolio");
    });
  }

  function submitCreate(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    startTransition(async () => {
      const res = await createPortfolioAction(name, capital);
      if (res.error) {
        setError(res.error);
        return;
      }
      setName("");
      setCreating(false);
      setOpen(false);
      router.push("/portfolio");
    });
  }

  const popoverPos = placement === "below" ? "right-0 top-[calc(100%+8px)]" : "bottom-0 left-[calc(100%+12px)]";

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={toggle}
        aria-label="Tài khoản của bạn"
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex h-11 w-11 items-center justify-center rounded-[11px] border border-app-border bg-app-surface-2 text-[13px] font-semibold text-app-text"
      >
        {user.initials}
      </button>

      {open && (
        <div
          role="menu"
          aria-label="Menu tài khoản"
          className={`absolute z-50 flex w-[360px] flex-col gap-[6px] rounded-2xl border border-app-border bg-app-surface p-2 text-app-text shadow-[0_18px_40px_rgba(0,0,0,0.55)] ${popoverPos}`}
        >
          <div className="flex items-center gap-3 border-b border-app-hairline p-[12px_12px_14px]">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-app-border text-sm font-semibold">{user.initials}</span>
            <div className="flex min-w-0 flex-grow flex-col gap-[2px]">
              <span className="text-sm font-semibold">{user.name}</span>
              <span className="truncate text-[11.5px] text-app-text-muted">{user.email}</span>
            </div>
            <span className="shrink-0 rounded-md border border-app-border px-2 py-[3px] text-[10.5px] text-app-text-3">Cá nhân</span>
          </div>

          <div className="flex flex-col gap-1 border-b border-app-hairline p-[4px_4px_8px]">
            <div className="flex items-center justify-between px-2 py-1">
              <span className="text-[10px] uppercase tracking-[0.1em] text-app-text-muted">Danh mục giấy</span>
              <span className="text-[10.5px] text-app-text-faint">{portfolios.length} danh mục</span>
            </div>

            {portfolios.map((p) => (
              <button
                key={p.id}
                type="button"
                role="menuitemradio"
                aria-checked={p.active}
                disabled={pending}
                onClick={() => selectPortfolio(p.id)}
                className="flex h-12 items-center gap-[11px] rounded-[10px] px-3 text-left disabled:opacity-60"
                style={{ background: p.active ? "var(--app-surface-2)" : "transparent" }}
              >
                <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: p.active ? "var(--app-accent)" : "var(--app-border-strong)" }} />
                <span className="flex min-w-0 flex-grow flex-col gap-[2px]">
                  <span
                    className="truncate text-[13.5px]"
                    style={{ fontWeight: p.active ? 600 : 500, color: p.active ? "var(--app-text)" : "var(--app-text-2)" }}
                  >
                    {p.name}
                  </span>
                  <span className="font-plex-mono text-[11px] text-app-text-muted">{p.nav}</span>
                </span>
                <span
                  className="shrink-0 font-plex-mono text-xs font-semibold"
                  style={{ color: p.pct >= 0 ? "var(--price-up)" : "var(--price-down)" }}
                >
                  {signVN(p.pct, 2)}%
                </span>
              </button>
            ))}

            {creating ? (
              <form onSubmit={submitCreate} className="flex flex-col gap-2 rounded-[10px] border border-app-border bg-app-surface-2 p-3">
                <label htmlFor="new-pf-name" className="text-[11.5px] text-app-text-3">Tên danh mục</label>
                <input
                  id="new-pf-name"
                  autoFocus
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Thử chiến lược RSI"
                  maxLength={60}
                  className="h-10 rounded-[9px] border border-app-border bg-app-surface px-3 text-[13px] text-app-text outline-none placeholder:text-app-text-faint"
                />
                <span className="text-[11.5px] text-app-text-3">Vốn ảo ban đầu</span>
                <div className="flex gap-2">
                  {STOCK_CAPITAL_PRESETS.map((c) => {
                    const selected = capital === c.amount;
                    return (
                      <button
                        key={c.amount}
                        type="button"
                        aria-pressed={selected}
                        onClick={() => setCapital(c.amount)}
                        className="flex h-10 flex-1 items-center justify-center rounded-[9px] border font-plex-mono text-[12px] font-semibold"
                        style={{
                          borderColor: selected ? "var(--app-accent)" : "var(--app-border)",
                          background: selected ? "var(--app-accent-surface)" : "var(--app-surface)",
                          color: selected ? "var(--app-text)" : "var(--app-text-3)",
                        }}
                      >
                        {c.label}
                      </button>
                    );
                  })}
                </div>
                <div className="flex gap-2 pt-1">
                  <button
                    type="button"
                    onClick={() => setCreating(false)}
                    className="h-9 flex-1 rounded-[9px] border border-app-border text-[12.5px] text-app-text-3"
                  >
                    Huỷ
                  </button>
                  <button
                    type="submit"
                    disabled={pending}
                    className="h-9 flex-1 rounded-[9px] bg-app-accent text-[12.5px] font-semibold text-app-accent-ink disabled:opacity-60"
                  >
                    {pending ? "Đang tạo…" : "Tạo danh mục"}
                  </button>
                </div>
              </form>
            ) : (
              <button
                type="button"
                onClick={() => {
                  setCreating(true);
                  setError(null);
                }}
                className="flex h-10 w-full items-center gap-[10px] rounded-[10px] px-3 text-[13px] font-medium text-app-accent"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                  <path d="M12 5v14M5 12h14" />
                </svg>
                <span>Tạo danh mục mới</span>
              </button>
            )}

            {error && <p className="m-0 px-3 text-[11.5px] text-price-down">{error}</p>}
          </div>

          <div className="flex flex-col gap-[2px] border-b border-app-hairline p-[2px_4px_6px]">
            <Link href="/settings/account" onClick={() => setOpen(false)} className="flex h-10 items-center gap-[11px] rounded-[10px] px-3 text-[13.5px] text-app-text-2">
              <LinkIcon d={PROFILE_D} />
              <span className="flex-grow">Hồ sơ &amp; bảo mật</span>
              <WillBadge />
            </Link>
            <Link href="/settings/ai" onClick={() => setOpen(false)} className="flex h-10 items-center gap-[11px] rounded-[10px] px-3 text-[13.5px] text-app-text-2">
              <LinkIcon d={SETTINGS_D} />
              <span className="flex-grow">Cài đặt</span>
            </Link>
            <div title="Chưa có thiết kế" className="flex h-10 cursor-not-allowed items-center gap-[11px] rounded-[10px] px-3 text-[13.5px] text-app-text-3">
              <LinkIcon d={HELP_D} />
              <span className="flex-grow">Trợ giúp</span>
              <WillBadge />
            </div>
          </div>

          <form action={logoutAction}>
            <button
              type="submit"
              className="mb-[2px] flex h-10 w-full items-center gap-[11px] rounded-[10px] px-3 text-[13.5px] font-medium text-price-down"
            >
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M15 17l5-5-5-5M20 12H9M12 20H6a2 2 0 01-2-2V6a2 2 0 012-2h6" />
              </svg>
              <span>Đăng xuất</span>
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
