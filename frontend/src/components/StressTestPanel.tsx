import type { StressScenario, StressTest } from "@/lib/api";
import { formatCompact, formatVN, signVN } from "@/lib/format";

// "Sức chịu đựng": what today's holdings would lose if a past VN-Index
// fall repeated, or under a hypothetical shock (phase-market-risk.md
// slice 3). A rough estimate, labelled as one.

// The loss we suggest capping a worst case at, as % of equity.
const TOLERABLE_LOSS_PCT = 20;

function dateVN(unix?: number) {
  return unix ? new Date(unix * 1000).toLocaleDateString("vi-VN", { month: "2-digit", year: "numeric" }) : "";
}

function ScenarioRow({ s, worstPct }: { s: StressScenario; worstPct: number }) {
  const width = worstPct > 0 && s.lossPct > 0 ? Math.max(2, (s.lossPct / worstPct) * 100) : 0;
  const floorDays = s.holdings.reduce((n, h) => n + h.floorDays, 0);
  return (
    <li className="flex flex-col gap-[6px] border-b border-app-hairline py-[10px] last:border-b-0">
      <div className="flex items-baseline justify-between gap-3">
        <div className="flex min-w-0 flex-col">
          <span className="text-[12.5px] text-app-text">{s.label}</span>
          <span className="text-[10.5px] text-app-text-muted">
            {s.kind === "historical" ? `${dateVN(s.from)} → ${dateVN(s.to)} · ` : "Giả định · "}
            VN-Index {signVN(s.indexChangePct, 1)}%
          </span>
        </div>
        <div className="flex shrink-0 flex-col items-end">
          <span className={`font-plex-mono text-[13px] font-semibold ${s.lossVnd > 0 ? "text-price-down" : "text-price-up"}`}>
            {s.lossVnd > 0 ? "−" : "+"}
            {formatCompact(Math.abs(s.lossVnd), "₫")}
          </span>
          <span className="font-plex-mono text-[10.5px] text-app-text-muted">{signVN(-s.lossPct, 1)}% tài khoản</span>
        </div>
      </div>
      <div className="h-[5px] rounded-full bg-app-hairline">
        <div className="h-full rounded-full bg-price-down" style={{ width: `${width}%` }} />
      </div>
      <details className="group">
        <summary className="cursor-pointer list-none text-[10.5px] text-app-text-muted">
          <span className="group-open:hidden">Chi tiết từng mã</span>
          <span className="hidden group-open:inline">Ẩn</span>
          {floorDays > 0 && <span className="text-price-down"> · {floorDays} phiên giảm sàn</span>}
        </summary>
        <table className="mt-1 w-full text-[11px]">
          <tbody>
            {s.holdings.map((h) => (
              <tr key={h.symbol} className="border-t border-app-hairline">
                <td className="py-[4px] font-semibold">{h.symbol}</td>
                <td className={`py-[4px] text-right font-plex-mono ${h.changePct < 0 ? "text-price-down" : "text-price-up"}`}>
                  {signVN(h.changePct, 1)}%
                </td>
                <td className="py-[4px] text-right font-plex-mono text-app-text-2">{formatCompact(-h.lossVnd, "₫")}</td>
                <td className="py-[4px] pl-2 text-right text-[10px] text-app-text-muted">
                  {h.estimated ? "ước tính qua beta" : h.floorDays > 0 ? `${h.floorDays} sàn, chuỗi dài nhất ${h.longestFloors}` : "giá thực tế"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {s.note && <p className="m-0 mt-1 text-[10px] text-app-text-faint">{s.note}</p>}
      </details>
    </li>
  );
}

export default function StressTestPanel({ result }: { result: StressTest }) {
  if (result.stockValue <= 0) {
    return (
      <section className="rounded-2xl border border-app-border bg-app-surface p-[18px_20px] text-[13px] text-app-text-3">
        Danh mục chưa có cổ phiếu, nên một đợt giảm của thị trường không ảnh hưởng tới tiền mặt của bạn.
      </section>
    );
  }
  const worst = result.scenarios.reduce<StressScenario | null>((w, s) => (!w || s.lossVnd > w.lossVnd ? s : w), null);
  const worstPct = worst ? worst.lossPct : 0;
  // Stock share of equity that keeps the worst case at TOLERABLE_LOSS_PCT.
  const stockLossFrac = worst && result.stockValue > 0 ? worst.lossVnd / result.stockValue : 0;
  const safeStockPct = stockLossFrac > 0 ? Math.min(100, TOLERABLE_LOSS_PCT / stockLossFrac) : 100;
  const stockPct = (result.stockValue / result.equity) * 100;
  const historical = result.scenarios.filter((s) => s.kind === "historical");
  const hypothetical = result.scenarios.filter((s) => s.kind === "hypothetical");

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[18px] lg:flex-row">
      <section className="flex min-w-0 flex-1 flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px]">
        <div className="flex flex-col gap-1">
          <h2 className="m-0 text-[14px] font-semibold">Kiểm tra sức chịu đựng</h2>
          <span className="text-[12px] text-app-text-muted">Nếu thị trường lặp lại một đợt giảm, danh mục hiện tại sẽ mất bao nhiêu?</span>
        </div>

        {worst && worst.lossVnd > 0 && (
          <p className="m-0 rounded-[10px] bg-app-surface-2 px-[12px] py-[10px] text-[12px] leading-[1.5] text-app-text-2">
            Kịch bản xấu nhất (<b>{worst.label}</b>): mất khoảng <b className="text-price-down">{formatCompact(worst.lossVnd, "₫")}</b>, tức{" "}
            {formatVN(worst.lossPct, 1)}% tài khoản. Cổ phiếu đang chiếm {formatVN(stockPct, 0)}% tài khoản; để mức lỗ này không quá{" "}
            {TOLERABLE_LOSS_PCT}% tài khoản, tỷ trọng cổ phiếu nên khoảng <b>{formatVN(safeStockPct, 0)}%</b> trở xuống.
          </p>
        )}

        {historical.length > 0 && (
          <div className="flex flex-col">
            <span className="text-[10.5px] uppercase tracking-[0.04em] text-app-text-faint">Các đợt giảm thực tế trong dữ liệu (từ 20%)</span>
            <ul className="m-0 flex list-none flex-col p-0">
              {historical.map((s) => (
                <ScenarioRow key={s.key} s={s} worstPct={worstPct} />
              ))}
            </ul>
          </div>
        )}
        <div className="flex flex-col">
          <span className="text-[10.5px] uppercase tracking-[0.04em] text-app-text-faint">Kịch bản giả định</span>
          <ul className="m-0 flex list-none flex-col p-0">
            {hypothetical.map((s) => (
              <ScenarioRow key={s.key} s={s} worstPct={worstPct} />
            ))}
          </ul>
        </div>
      </section>

      <aside className="flex w-full flex-col gap-3 rounded-2xl border border-app-border bg-app-surface p-[18px_20px] lg:w-[352px] lg:shrink-0">
        <h3 className="m-0 text-[13px] font-semibold">Độ nhạy với thị trường (beta)</h3>
        <table className="w-full text-[11.5px]">
          <thead>
            <tr className="text-[10px] text-app-text-muted">
              <th className="pb-1 text-left font-normal">Mã</th>
              <th className="pb-1 text-right font-normal">Beta đo được</th>
              <th className="pb-1 text-right font-normal">Dùng khi sốc</th>
            </tr>
          </thead>
          <tbody>
            {result.betas.map((b) => (
              <tr key={b.symbol} className="border-t border-app-hairline">
                <td className="py-[5px] font-semibold">{b.symbol}</td>
                <td className="py-[5px] text-right font-plex-mono">{b.samples > 0 ? formatVN(b.beta, 2) : "—"}</td>
                <td className="py-[5px] text-right font-plex-mono">{formatVN(b.crisisBeta, 2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <ul className="m-0 flex list-disc flex-col gap-[6px] pl-4 text-[10.5px] leading-[1.45] text-app-text-muted">
          <li>Beta 1 nghĩa là cổ phiếu thường biến động như VN-Index; beta 1,5 là mạnh hơn 50%.</li>
          <li>Khi sốc, beta được nâng lên tối thiểu 1: trong khủng hoảng hầu hết cổ phiếu giảm cùng thị trường (như 2007–2009).</li>
          <li>Phiên giảm sàn: lệnh cắt lỗ có thể không khớp vì không có người mua. Cổ phiếu mua mới phải chờ T+2 mới bán được.</li>
          <li>Đây là ước tính thô dựa trên dữ liệu quá khứ, không phải dự báo.</li>
        </ul>
      </aside>
    </div>
  );
}
