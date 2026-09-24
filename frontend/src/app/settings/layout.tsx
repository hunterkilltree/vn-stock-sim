import RailNav from "@/components/RailNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import SettingsSubnav from "@/components/SettingsSubnav";

// Settings shell from design/screens/Settings-AI.dc.html: icon rail +
// subnav column + section content. See phase-g.md decision 11.
export default function SettingsLayout({ children }: LayoutProps<"/settings">) {
  return (
    // Phones stack the section tabs above the content (phase-j.md decision 11).
    <div className="flex flex-1 flex-col bg-app-bg text-app-text lg:flex-row">
      <RailNav account={<AccountMenuButton placement="right" />} />
      <SettingsSubnav />
      <div className="flex min-w-0 flex-1 flex-col gap-[14px] px-[18px] pt-[18px] lg:gap-[18px] lg:p-[24px_28px]">{children}</div>
    </div>
  );
}
