import RailNav from "@/components/RailNav";
import AccountMenuButton from "@/components/AccountMenuButton";
import SettingsSubnav from "@/components/SettingsSubnav";

// Settings shell from design/screens/Settings-AI.dc.html: icon rail +
// subnav column + section content. See phase-g.md decision 11.
export default function SettingsLayout({ children }: LayoutProps<"/settings">) {
  return (
    <div className="flex flex-1 bg-app-bg text-app-text">
      <RailNav account={<AccountMenuButton placement="right" />} />
      <SettingsSubnav />
      <div className="flex min-w-0 flex-1 flex-col gap-[18px] p-[24px_28px]">{children}</div>
    </div>
  );
}
