import type { ReplayResult } from "@/lib/api";
import { formatVN, signVN, tone } from "@/lib/format";

// Replay.dc.html's "Kết quả phiên" + "Điểm kỹ năng tạm tính" panels.
// Rendered continuously (not only after the session ends -- the design
// itself labels the skill panel "provisional"), backed by real KPIs
// (phase-f.md reuses Phase E's portfolio.Stats over the session's own
// dedicated portfolio) and a real, explicitly-labeled heuristic skill
// score (score.go) rather than the design's fixed sample numbers.
export default function ReplayResultsPanel({ result }: { result: ReplayResult }) {
  const stats: { k: string; v: string; color: string }[] = [
    { k: "NAV mô phỏng", v: `${formatVN(result.nav / 1_000_000, 2)} tr`, color: "var(--app-text)" },
    { k: "Lãi/lỗ phiên", v: `${signVN(result.pnlPercent, 2)}%`, color: tone(result.pnlPercent) },
    { k: "Số lệnh", v: `${result.totalTrades}`, color: "var(--app-text)" },
    { k: "Tỷ lệ thắng", v: `${result.wins} / ${result.wins + result.losses}`, color: tone(result.wins - result.losses) },
    { k: "Sụt giảm tối đa", v: result.maxDrawdownPercent > 0 ? `−${formatVN(result.maxDrawdownPercent, 2)}%` : "0%", color: "#FF5C5C" },
    { k: "Hệ số lợi nhuận", v: result.profitFactor > 0 ? formatVN(result.profitFactor, 2) : "—", color: "var(--app-text)" },
  ];

  const skills: { k: string; v: number }[] = [
    { k: "Điểm vào", v: result.skill.entryQuality },
    { k: "Điểm ra", v: result.skill.exitQuality },
    { k: "Kỷ luật cắt lỗ", v: result.skill.stopDiscipline },
    { k: "Quản trị vị thế", v: result.skill.positionSizing },
  ];

  const circ = 2 * Math.PI * 38;
  const dash = `${((circ * result.skill.overall) / 100).toFixed(1)} ${circ.toFixed(1)}`;

  return (
    <>
      <section className="flex flex-col gap-[14px] rounded-2xl border border-app-border bg-app-surface p-[18px]">
        <h2 className="m-0 text-[15px] font-semibold">Kết quả phiên</h2>
        <div className="grid grid-cols-2 gap-[14px_10px]">
          {stats.map((s) => (
            <div key={s.k} className="flex flex-col gap-[3px]">
              <span className="text-[10.5px] text-app-text-muted">{s.k}</span>
              <span className="font-plex-mono text-[16px] font-semibold" style={{ color: s.color }}>
                {s.v}
              </span>
            </div>
          ))}
        </div>
      </section>

      <section className="flex flex-1 flex-col gap-[14px] rounded-2xl border border-[#4A3521] bg-[#1C1813] p-[18px]">
        <h2 className="m-0 text-[15px] font-semibold">
          {result.skill.final ? "Điểm kỹ năng" : "Điểm kỹ năng tạm tính"}
        </h2>
        <div className="flex items-center gap-[18px]">
          <svg width="92" height="92" viewBox="0 0 92 92" fill="none" role="img" aria-label={`Điểm kỹ năng ${result.skill.final ? "" : "tạm tính "}${formatVN(result.skill.overall, 0)} trên 100`}>
            <circle cx="46" cy="46" r="38" stroke="#2C2C28" strokeWidth="9" />
            <circle cx="46" cy="46" r="38" stroke="#E08A3C" strokeWidth="9" strokeLinecap="round" strokeDasharray={dash} transform="rotate(-90 46 46)" />
            <text x="46" y="50" textAnchor="middle" fill="#F3F0E9" fontFamily="'IBM Plex Mono', monospace" fontSize="24" fontWeight="600">
              {formatVN(result.skill.overall, 0)}
            </text>
            <text x="46" y="65" textAnchor="middle" fill="#8A867E" fontFamily="'Be Vietnam Pro', sans-serif" fontSize="9.5">
              / 100
            </text>
          </svg>
          <div className="flex min-w-0 flex-1 flex-col gap-[9px]">
            {skills.map((s) => (
              <div key={s.k} className="flex flex-col gap-[4px]">
                <div className="flex justify-between text-[11.5px]">
                  <span className="text-app-text-3">{s.k}</span>
                  <span className="font-plex-mono text-app-text-2">{formatVN(s.v, 0)}</span>
                </div>
                <div className="h-[5px] rounded-[3px] bg-app-border">
                  <div className="h-[5px] rounded-[3px] bg-app-accent" style={{ width: `${Math.min(100, Math.max(0, s.v))}%` }} />
                </div>
              </div>
            ))}
          </div>
        </div>
        <p className="m-0 text-[12px] leading-[1.55] text-app-text-3">
          {result.skill.final
            ? "Điểm heuristic dựa trên lệnh thực trong phiên -- không phải đánh giá kỹ năng chính thức."
            : "Điểm tạm tính, chỉ dựa trên các nến đã hiện -- điểm cuối cùng chốt lại khi phiên kết thúc."}
        </p>
      </section>
    </>
  );
}
