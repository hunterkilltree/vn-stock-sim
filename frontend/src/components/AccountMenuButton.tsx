import Link from "next/link";
import AccountMenu, { type MenuPortfolio } from "@/components/AccountMenu";
import { AvatarPlaceholder } from "@/components/icons";
import { getPortfolioSummaryByID, type Portfolio } from "@/lib/api";
import { formatVN } from "@/lib/format";
import { getActivePortfolio, getSessionToken, getSessionUser } from "@/lib/session";

type Props = { placement: "below" | "right" };

// "Trần Đức" -> "TĐ", matching Account-Menu.dc.html's avatar.
function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "?";
  const first = words[0][0];
  const last = words.length > 1 ? words[words.length - 1][0] : words[0][1] ?? "";
  return (first + last).toUpperCase();
}

// NAV in the design's short form: "1.086,42 tr ₫" / "14.268,40 USDT".
function navLabel(p: Portfolio, equity: number): string {
  if (p.currency === "VND") return `${formatVN(equity / 1_000_000, 2)} tr ₫`;
  return `${formatVN(equity, 2)} ${p.currency}`;
}

// Server-side loader for the account popover (phase-g.md decision 9):
// renders the Client AccountMenu with real portfolios + NAVs, or a
// sign-in link for guests. Passed into Client shells (RailNav) as a
// prop, so the data fetch stays on the server.
export default async function AccountMenuButton({ placement }: Props) {
  const user = await getSessionUser();
  const token = user ? await getSessionToken() : null;

  if (!user || !token) {
    return (
      <Link
        href="/login"
        title="Đăng nhập"
        aria-label="Đăng nhập"
        className="flex h-11 w-11 items-center justify-center rounded-[11px] border border-app-border bg-app-surface-2 text-app-text-3"
      >
        <AvatarPlaceholder width={19} height={19} />
      </Link>
    );
  }

  let portfolios: MenuPortfolio[] = [];
  try {
    // Both markets are listed; each market keeps its own active
    // portfolio (phase-i.md decision 9), so up to two rows are marked.
    const [stock, crypto] = await Promise.all([getActivePortfolio(token, "stock"), getActivePortfolio(token, "crypto")]);
    const activeIDs = new Set([stock.active?.id, crypto.active?.id]);
    const list = stock.all;
    const summaries = await Promise.all(list.map((p) => getPortfolioSummaryByID(p.id, token)));
    portfolios = list.map((p, i) => {
      const equity = summaries[i].totalEquity;
      return {
        id: p.id,
        name: p.name,
        nav: navLabel(p, equity),
        pct: p.startingCapital > 0 ? ((equity - p.startingCapital) / p.startingCapital) * 100 : 0,
        active: activeIDs.has(p.id),
        market: p.market === "crypto" ? "crypto" : "stock",
      };
    });
  } catch {
    // Menu still opens (profile, settings, logout) with an empty list.
  }

  return (
    <AccountMenu
      placement={placement}
      user={{ name: user.displayName, email: user.email, initials: initialsOf(user.displayName) }}
      portfolios={portfolios}
    />
  );
}
