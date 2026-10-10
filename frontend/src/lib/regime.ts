import type { MarketRegime, RegimeLevel } from "@/lib/api";
import { formatVN } from "@/lib/format";
import type { Exposure } from "@/lib/exposure";

// Display names and colours for the market regime levels, shared by the
// regime card, the Portfolio banner and the order ticket.
export const REGIME_LEVEL: Record<RegimeLevel, { label: string; color: string }> = {
  normal: { label: "Bình thường", color: "#35C77F" },
  caution: { label: "Thận trọng", color: "#E3B341" },
  high_risk: { label: "Rủi ro cao", color: "#FF5C5C" },
};

// "Nên làm" for a level, built from the backend's limits so the numbers
// live in one place (backend/internal/regime LimitsFor).
export function regimeAdvice(r: MarketRegime): string {
  const { exposureCapPct: cap, riskPerTradePct: risk } = r.limits;
  const riskText = `${formatVN(risk, risk < 1 ? 2 : 0)}% vốn`;
  switch (r.level) {
    case "high_risk":
      return `Ưu tiên bảo toàn vốn: cổ phiếu tối đa ${formatVN(cap, 0)}% tài khoản, rủi ro mỗi lệnh ${riskText}, chỉ giữ mã còn trên SMA 200.`;
    case "caution":
      return `Cổ phiếu tối đa ${formatVN(cap, 0)}% tài khoản, rủi ro mỗi lệnh ${riskText}, hạn chế mua mới, không dùng margin, nâng điểm cắt lỗ cho vị thế đang lãi.`;
    default:
      return `Giữ kỷ luật: mọi lệnh đều có điểm cắt lỗ, rủi ro tối đa ${riskText} mỗi lệnh.`;
  }
}

export function exposureFor(r: MarketRegime, stockValue: number, equity: number): Exposure {
  return { stockValue, equity, capPct: r.limits.exposureCapPct, levelLabel: REGIME_LEVEL[r.level].label, color: REGIME_LEVEL[r.level].color };
}
