import { notFound } from "next/navigation";
import WillBadge from "@/components/WillBadge";
import { settingsSections } from "@/lib/settingsSections";

export default async function SettingsSectionPage({ params }: PageProps<"/settings/[section]">) {
  const { section } = await params;
  const s = settingsSections.find((x) => x.slug === section && !x.built);
  if (!s) notFound();

  return (
    <>
      <header className="flex items-center gap-3">
        <h2 className="m-0 font-display text-[25px] font-bold tracking-[-0.015em]">{s.label}</h2>
        <WillBadge />
      </header>
      <section className="flex max-w-[640px] flex-col gap-2 rounded-[14px] border border-app-border bg-app-surface p-[18px]">
        <p className="m-0 text-[13px] leading-[1.6] text-app-text-3">
          Mục này nằm trong danh sách dự kiến (WILL) của bản thiết kế — chưa được thiết kế chi tiết nên chưa có tuỳ chọn nào ở đây.
        </p>
      </section>
    </>
  );
}
