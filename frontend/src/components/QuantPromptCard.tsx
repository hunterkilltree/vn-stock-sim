import Link from "next/link";

export const QUANT_PROMPT_SAMPLE = "Lọc cổ phiếu HOSE RSI < 30, ROE > 15%";

// "Hỏi Quant" quick-ask card (design/screens/Main.dc.html's right
// column) -- opens Quant with the sample question pre-filled.
export default function QuantPromptCard() {
  return (
    <Link
      href={`/quant?q=${encodeURIComponent(QUANT_PROMPT_SAMPLE)}`}
      className="flex items-center gap-3 rounded-2xl border border-app-border bg-app-surface p-[14px_16px] text-app-text"
    >
      <span className="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded-[10px] bg-app-surface-3">
        <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#E08A3C" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M12 3l2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5z" />
        </svg>
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-[1px]">
        <span className="text-[13px] font-semibold">Hỏi Quant</span>
        <span className="overflow-hidden text-ellipsis whitespace-nowrap text-[11.5px] text-app-text-muted">
          &ldquo;{QUANT_PROMPT_SAMPLE}&rdquo;
        </span>
      </div>
      <span className="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded-[10px] bg-app-surface-3 text-app-text">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M5 12h13M13 6l6 6-6 6" />
        </svg>
      </span>
    </Link>
  );
}
