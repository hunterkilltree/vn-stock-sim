import { formatThousandsVN } from "@/lib/format";

type Props = { ceiling: number; reference: number; floor: number };

// Tran/Tham chieu/San chips from design/screens/Detail.dc.html's header.
export default function PriceBandChips({ ceiling, reference, floor }: Props) {
  const bands: { label: string; value: number; color: string }[] = [
    { label: "Trần", value: ceiling, color: "#C08BFF" },
    { label: "Tham chiếu", value: reference, color: "#F0C243" },
    { label: "Sàn", value: floor, color: "#4FD3E8" },
  ];
  return (
    <div className="flex gap-2">
      {bands.map((b) => (
        <div key={b.label} className="box-border flex flex-col gap-[2px] rounded-[9px] border border-app-border bg-app-surface px-[11px] py-[6px]">
          <span className="text-[9.5px] uppercase tracking-[0.08em] text-app-text-muted">{b.label}</span>
          <span className="font-plex-mono text-[12.5px] font-semibold" style={{ color: b.color }}>
            {formatThousandsVN(b.value)}
          </span>
        </div>
      ))}
    </div>
  );
}
