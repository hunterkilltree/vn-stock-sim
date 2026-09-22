// The design system's ".badge-will" component (design/tokens.css,
// design/DESIGN-SYSTEM.md section 4): marks a feature with no screen in
// the design set (see design/SCREENS.md's "Not designed yet" list).
// Grey, never amber -- amber is reserved for Replay/primary action/
// active market, and a planned feature must not out-shout a built one.
export default function WillBadge() {
  return (
    <span
      className="ml-auto rounded-[5px] border border-app-border-strong px-[6px] py-[1.5px] font-plex-mono text-[9px] font-semibold text-app-text-muted"
      style={{ letterSpacing: "0.12em" }}
    >
      WILL
    </span>
  );
}
