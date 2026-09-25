"use client";

import Link from "next/link";
import { useOptimistic, useTransition } from "react";
import { setWatchedAction } from "@/lib/watchlistActions";

type Props = { symbol: string; watched: boolean; signedIn: boolean; className?: string };

const BOOKMARK_D = "M6 4h12v17l-6-4-6 4V4z"; // Mobile-Detail.dc.html's bookmark icon

// Mobile-Detail.dc.html's "Thêm vào danh sách theo dõi" button, used on
// both Detail screens and both layouts (phase-k.md decision 12). Guests
// get a link to sign in instead.
export default function WatchlistButton({ symbol, watched, signedIn, className = "" }: Props) {
  const [on, setOn] = useOptimistic(watched);
  const [pending, start] = useTransition();
  const cls = `flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border ${className}`;
  const label = on ? "Bỏ khỏi danh sách theo dõi" : "Thêm vào danh sách theo dõi";

  if (!signedIn) {
    return (
      <Link href="/login" aria-label={label} title="Đăng nhập để theo dõi mã" className={`${cls} border-app-border bg-app-surface text-app-text-3`}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d={BOOKMARK_D} />
        </svg>
      </Link>
    );
  }
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={on}
      title={label}
      disabled={pending}
      onClick={() =>
        start(async () => {
          setOn(!on);
          await setWatchedAction(symbol, !on);
        })
      }
      className={`${cls} ${on ? "border-app-accent-border bg-app-accent-surface text-app-accent" : "border-app-border bg-app-surface text-app-text-3"}`}
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill={on ? "currentColor" : "none"} stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d={BOOKMARK_D} />
      </svg>
    </button>
  );
}
