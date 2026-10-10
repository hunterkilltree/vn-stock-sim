import Link from "next/link";
import type { MarketRegime, RegimeLevel, RegimeSignal } from "@/lib/api";

// "Nhiệt kế thị trường": the market risk regime (phase-market-risk.md
// feature 1). It measures conditions and says what to do about risk; it
// never claims to predict a crash. The guidance per level is the
// suggested tightening from feature 2 -- advice here, enforced only once
// regime-aware risk limits land.
const LEVEL: Record<RegimeLevel, { label: string; color: string; advice: string }> = {
  normal: {
    label: "Bình thường",
    color: "#35C77F",
    advice: "Giữ kỷ luật: mọi lệnh đều có điểm cắt lỗ, rủi ro tối đa 1% vốn mỗi lệnh.",
  },
  caution: {
    label: "Thận trọng",
    color: "#E3B341",
    advice: "Giảm rủi ro mỗi lệnh còn một nửa, hạn chế mua mới, không dùng margin, nâng điểm cắt lỗ cho vị thế đang lãi.",
  },
  high_risk: {
    label: "Rủi ro cao",
    color: "#FF5C5C",
    advice: "Ưu tiên bảo toàn vốn: tỷ trọng cổ phiếu tối đa khoảng 40%, rủi ro mỗi lệnh 0,25% vốn, chỉ giữ mã còn trên SMA 200.",
  },
};

const ZONE: Record<RegimeSignal["zone"], { label: string; color: string }> = {
  ok: { label: "Ổn", color: "#35C77F" },
  caution: { label: "Cảnh báo", color: "#E3B341" },
  risk: { label: "Rủi ro", color: "#FF5C5C" },
  nodata: { label: "Chưa có", color: "var(--app-text-faint)" },
};

// Backend formats with ".", the app shows Vietnamese "," decimals.
const vn = (s: string) => s.replaceAll(".", ",");

function asOfDate(unix: number) {
  return new Date(unix * 1000).toLocaleDateString("vi-VN", { timeZone: "Asia/Ho_Chi_Minh" });
}

// compact: a one-line banner for the Portfolio page, linking to the full
// card on /stocks.
export default function MarketRegimeCard({ regime, compact = false }: { regime: MarketRegime; compact?: boolean }) {
  const lv = LEVEL[regime.level];
  const counted = regime.signals.filter((s) => s.zone !== "nodata");
  const missing = regime.signals.filter((s) => s.zone === "nodata");

  if (compact) {
    return (
      <Link
        href="/stocks#nhiet-ke"
        className="flex items-center gap-3 rounded-[12px] border border-app-border bg-app-surface px-[14px] py-[10px]"
        aria-label={`Nhiệt kế thị trường: ${lv.label}`}
      >
        <span className="h-[10px] w-[10px] shrink-0 rounded-full" style={{ background: lv.color }} />
        <span className="shrink-0 text-[12px] font-semibold" style={{ color: lv.color }}>
          Thị trường: {lv.label}
        </span>
        <span className="min-w-0 flex-1 truncate text-[11.5px] text-app-text-muted">{lv.advice}</span>
        <span className="shrink-0 text-[11.5px] font-medium text-app-accent">Chi tiết</span>
      </Link>
    );
  }

  return (
    <section id="nhiet-ke" className="flex scroll-mt-4 flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[16px_18px]" aria-label="Nhiệt kế thị trường">
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[13px] font-semibold">Nhiệt kế thị trường</h2>
        <span className="text-[10.5px] text-app-text-muted">{asOfDate(regime.asOf)}</span>
      </div>

      <div className="flex items-baseline justify-between gap-3">
        <span className="font-display text-[22px] font-bold" style={{ color: lv.color }}>
          {lv.label}
        </span>
        <span className="shrink-0 whitespace-nowrap text-[11px] text-app-text-muted" title="Cảnh báo = 1 điểm, rủi ro = 2 điểm">
          Điểm rủi ro <span className="font-plex-mono">{regime.score}/{regime.maxScore}</span>
        </span>
      </div>

      <div className="grid grid-cols-3 gap-[3px]" role="img" aria-label={`Mức: ${lv.label}`}>
        {(["normal", "caution", "high_risk"] as RegimeLevel[]).map((k) => (
          <div key={k} className="h-[6px] rounded-full" style={{ background: LEVEL[k].color, opacity: k === regime.level ? 1 : 0.18 }} />
        ))}
      </div>

      {regime.knockout && (
        <p className="m-0 rounded-[9px] border border-app-warn-border bg-app-warn-surface px-[10px] py-[7px] text-[11px] leading-[1.45] text-app-warn-text">
          {regime.knockout}
        </p>
      )}

      <p className="m-0 rounded-[10px] bg-app-surface-2 px-[11px] py-[9px] text-[11.5px] leading-[1.5] text-app-text-2">
        <b className="text-app-text">Nên làm: </b>
        {lv.advice}
      </p>

      <ul className="m-0 flex list-none flex-col p-0">
        {counted.map((s) => (
          <li key={s.key} className="flex flex-col gap-[2px] border-b border-app-hairline py-[7px] last:border-b-0">
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-[12px] text-app-text-2">{s.label}</span>
              <span className="shrink-0 text-[11px] font-semibold" style={{ color: ZONE[s.zone].color }}>
                {ZONE[s.zone].label}
              </span>
            </div>
            <span className="font-plex-mono text-[10.5px] text-app-text">{vn(s.value)}</span>
            <details className="group">
              <summary className="cursor-pointer list-none text-[10.5px] text-app-text-muted">
                <span className="group-open:hidden">Vì sao quan trọng?</span>
                <span className="hidden group-open:inline">Ẩn</span>
              </summary>
              <span className="mt-1 block text-[10.5px] leading-[1.45] text-app-text-muted">
                {s.why} <span className="text-app-text-faint">Quy tắc: {s.rule}.</span>
              </span>
            </details>
          </li>
        ))}
      </ul>

      {missing.length > 0 && (
        <p className="m-0 text-[10px] leading-[1.45] text-app-text-faint">
          Chưa có dữ liệu, không tính điểm: {missing.map((s) => s.label).join(", ")}.
          {regime.universe > 0 && ` Độ rộng tính trên ${regime.universe} mã đang có trong ứng dụng.`}
        </p>
      )}
      <p className="m-0 text-[10px] leading-[1.45] text-app-text-faint">
        Đo lường điều kiện hiện tại, không dự báo sụp đổ. Không phải khuyến nghị đầu tư.
      </p>
    </section>
  );
}
