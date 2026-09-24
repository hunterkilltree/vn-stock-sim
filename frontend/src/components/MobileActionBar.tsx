import Link from "next/link";

// Mobile-Detail.dc.html's bottom bar: Replay button + "Mua giấy" /
// "Bán giấy". Sticky to the bottom of the page on phones, hidden from lg
// up where the order ticket sits beside the chart (phase-j.md decision 6).
// Buy/Sell are links to the ticket below with that side preselected.
export default function MobileActionBar({ replayHref, buyHref, sellHref }: { replayHref: string; buyHref: string; sellHref: string }) {
  return (
    <div
      className="sticky bottom-0 z-30 -mx-[18px] mt-auto flex items-center gap-[10px] border-t border-app-border bg-app-chrome px-[18px] pt-3 lg:hidden"
      style={{ paddingBottom: "calc(14px + env(safe-area-inset-bottom))" }}
    >
      <Link
        href={replayHref}
        aria-label="Chạy Replay cho mã này"
        className="flex h-[52px] w-[52px] shrink-0 items-center justify-center rounded-[14px] border border-app-accent-border bg-app-accent-surface text-app-accent"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z" />
        </svg>
      </Link>
      <Link href={buyHref} className="flex h-[52px] flex-1 items-center justify-center rounded-[14px] bg-price-up text-[15px] font-semibold text-price-up-ink">
        Mua giấy
      </Link>
      <Link href={sellHref} className="flex h-[52px] flex-1 items-center justify-center rounded-[14px] bg-price-down text-[15px] font-semibold text-price-down-ink">
        Bán giấy
      </Link>
    </div>
  );
}
