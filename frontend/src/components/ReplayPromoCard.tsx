// Replay Mode promo card (design/screens/Main.dc.html's right column).
// The design links this CTA straight to Replay.dc.html; this app's
// /replay route doesn't exist until Phase F, so the button renders
// disabled (would otherwise 404) rather than a live link, same
// "designed, not built yet" treatment as SidebarNav's Replay nav item.
export default function ReplayPromoCard() {
  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-app-accent-border bg-app-accent-surface p-[18px_20px]">
      <span className="text-[11px] uppercase tracking-[0.09em] text-app-accent">Chế độ Replay</span>
      <p className="m-0 font-display text-[18px] font-semibold leading-[1.35]" style={{ textWrap: "pretty" }}>
        &ldquo;Giao dịch quá khứ trước khi giao dịch tương lai.&rdquo;
      </p>
      <p className="m-0 text-[12.5px] leading-[1.5] text-app-text-3">
        Chạy lại phiên 2021 từng nến một, đặt lệnh mà không thấy tương lai, rồi nhận điểm.
      </p>
      <button
        type="button"
        disabled
        title="Sắp ra mắt"
        className="flex h-11 cursor-not-allowed items-center justify-center rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink opacity-60"
      >
        Bắt đầu phiên Replay
      </button>
    </section>
  );
}
