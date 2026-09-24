import Link from "next/link";

// Replay Mode promo card (design/screens/Main.dc.html's right column).
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
      <Link
        href="/replay"
        className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink"
      >
        Bắt đầu phiên Replay
      </Link>
    </section>
  );
}
