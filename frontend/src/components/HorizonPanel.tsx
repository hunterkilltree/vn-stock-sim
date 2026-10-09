import Link from "next/link";
import type { HoldStats, Horizon, HorizonVerdict } from "@/lib/api";
import { formatThousandsVN, formatVN, signVN } from "@/lib/format";

// One holding-horizon tab of the Khuyến nghị card: verdict, why (every
// factor with the rule it teaches), risk, and what history says
// (phase-holding-horizon.md).
const VERDICT: Record<HorizonVerdict, { label: string; color: string }> = {
  suitable: { label: "Phù hợp nắm giữ", color: "#35C77F" },
  watch: { label: "Cần theo dõi", color: "#E3B341" },
  avoid: { label: "Chưa phù hợp", color: "#FF5C5C" },
};

// Backend formats with ".", the app shows Vietnamese "," decimals.
const vn = (s: string) => s.replaceAll(".", ",");

function pct(v: number) {
  return `${signVN(v, 1)}%`;
}

function History({ stats }: { stats: HoldStats[] }) {
  if (stats.length === 0) return null;
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[10.5px] uppercase tracking-[0.04em] text-app-text-faint">Lịch sử nói gì</span>
      {stats.map((s) => (
        <div key={s.label} className="flex flex-col gap-[5px] rounded-[11px] border border-app-hairline bg-app-surface-2 p-[9px_11px]">
          <span className="text-[11.5px] text-app-text-2">
            Mua vào một ngày bất kỳ, giữ <b>{s.label}</b>:
          </span>
          <div className="grid grid-cols-4 gap-2 font-plex-mono text-[11px]">
            {[
              { k: "Có lãi", v: `${formatVN(s.winRate, 0)}%`, c: "" },
              { k: "Trung vị", v: pct(s.median), c: s.median >= 0 ? "text-price-up" : "text-price-down" },
              { k: "Xấu 10%", v: pct(s.p10), c: "text-price-down" },
              { k: "Tốt 10%", v: pct(s.p90), c: "text-price-up" },
            ].map((x) => (
              <div key={x.k} className="flex flex-col">
                <span className="font-sans text-[9.5px] text-app-text-muted">{x.k}</span>
                <span className={`font-semibold ${x.c}`}>{x.v}</span>
              </div>
            ))}
          </div>
          {s.index && (
            <span className="text-[10px] text-app-text-muted">
              VN-Index cùng kỳ: có lãi {formatVN(s.index.winRate, 0)}% số lần, trung vị {pct(s.index.median)}
            </span>
          )}
        </div>
      ))}
      <span className="text-[9.5px] leading-[1.4] text-app-text-faint">
        {formatVN(stats[0].samples, 0)} điểm mua chồng lấn nhau, không phải các lần độc lập. Lợi nhuận quá khứ không đảm bảo tương lai.
      </span>
    </div>
  );
}

export default function HorizonPanel({ horizon, symbol }: { horizon: Horizon; symbol: string }) {
  const v = VERDICT[horizon.verdict];
  const available = horizon.factors.filter((f) => f.available);
  const missing = horizon.factors.filter((f) => !f.available);
  const r = horizon.risk;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-baseline justify-between gap-3">
        <span className="font-display text-[22px] font-bold" style={{ color: v.color }}>
          {v.label}
        </span>
        <span className="font-plex-mono text-[11.5px] text-app-text-muted">{formatVN(horizon.score, 0)}/100</span>
      </div>
      <div className="relative h-[6px] rounded-full bg-app-hairline" role="img" aria-label={`Điểm ${horizon.score} trên 100`}>
        <div className="absolute inset-y-0 left-0 rounded-full" style={{ width: `${horizon.score}%`, background: v.color }} />
        {/* Band edges: 45 (Cần theo dõi) and 65 (Phù hợp). */}
        <div className="absolute inset-y-[-2px] left-[45%] w-px bg-app-text-faint" />
        <div className="absolute inset-y-[-2px] left-[65%] w-px bg-app-text-faint" />
      </div>

      {horizon.knockout && (
        <p className="m-0 rounded-[9px] border border-app-warn-border bg-app-warn-surface px-[10px] py-[7px] text-[11px] leading-[1.45] text-app-warn-text">
          {horizon.knockout}
        </p>
      )}

      <details className="group">
        <summary className="cursor-pointer list-none text-[11.5px] font-medium text-app-accent">
          <span className="group-open:hidden">Vì sao? Xem {available.length} yếu tố</span>
          <span className="hidden group-open:inline">Ẩn yếu tố</span>
        </summary>
        <ul className="m-0 mt-2 flex list-none flex-col p-0">
          {available.map((f) => (
            <li key={f.key} className="flex flex-col gap-[3px] border-b border-app-hairline py-[7px] last:border-b-0">
              <div className="flex items-baseline justify-between gap-3">
                <span className="text-[12px] text-app-text-2">{f.label}</span>
                <span className="shrink-0 font-plex-mono text-[10.5px] text-app-text-muted" title="Điểm yếu tố / tỷ trọng trong tổng điểm">
                  {formatVN(f.score, 0)}/100 · {formatVN(f.weight * 100, 0)}%
                </span>
              </div>
              <span className="font-plex-mono text-[10.5px] text-app-text">{vn(f.value)}</span>
              <span className="text-[10.5px] leading-[1.4] text-app-text-muted">{f.why}</span>
            </li>
          ))}
        </ul>
        {missing.length > 0 && (
          <p className="m-0 mt-1 text-[10px] leading-[1.45] text-app-text-faint">
            Chưa có dữ liệu, không tính vào điểm: {missing.map((f) => f.label).join(", ")}.
          </p>
        )}
      </details>

      <div className="grid grid-cols-3 gap-2">
        {[
          { k: "Biến động", v: `${formatVN(r.volatilityPct, 0)}%/năm` },
          { k: `Giảm sâu nhất ${r.drawdownWindow}`, v: pct(r.maxDrawdownPct) },
          ...(r.stopLoss ? [{ k: "Cắt lỗ tham khảo", v: `${formatThousandsVN(r.stopLoss)} (${pct(r.stopPct ?? 0)})` }] : []),
        ].map((x) => (
          <div key={x.k} className="flex min-w-0 flex-col gap-[2px]">
            <span className="text-[9.5px] leading-[1.3] text-app-text-muted">{x.k}</span>
            <span className="font-plex-mono text-[11.5px] font-semibold">{x.v}</span>
          </div>
        ))}
      </div>

      <History stats={horizon.history} />

      <Link href={`/replay?symbol=${symbol}`} className="text-[11.5px] font-medium text-app-accent">
        Thử nắm giữ trong Replay →
      </Link>
    </div>
  );
}
