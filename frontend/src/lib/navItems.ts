// Single source of truth for the dark app shell's sidebar navigation --
// the exact 10-item list + icon path data from
// design/screens/Main.dc.html's renderVals() `navRaw` array (copied
// verbatim, not redrawn by eye, per phase-c.md decision 2). "Settings"
// is deliberately NOT in this list -- the design renders it as a
// separate link below the loop (see SidebarNav.tsx).
export type NavItem = {
  href: string;
  label: string;
  // SVG path "d" attribute, viewBox 0 0 24 24, stroke-based (matches
  // every nav icon in the design).
  d: string;
  // "built": a real route exists today, renders as a normal link.
  // "soon": a real, planned screen (design/screens/*.dc.html exists or
  //   is scheduled) not built yet -- renders disabled, no badge, matches
  //   the source's `planned: false` entries whose target page isn't
  //   live yet.
  // "will": the source's `planned: true` entries -- true backlog with
  //   no screen in the design set at all (design/SCREENS.md's "Not
  //   designed yet" list) -- gets the real WillBadge.
  kind: "built" | "soon" | "will";
};

export const navItems: NavItem[] = [
  {
    href: "/stocks",
    label: "Tổng quan",
    d: "M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z",
    kind: "built",
  },
  {
    href: "/stocks/VNM",
    label: "Biểu đồ & chỉ báo",
    d: "M4 20V11M9 20V4M14 20V14M19 20V8",
    kind: "built",
  },
  {
    href: "/screener",
    label: "Bộ lọc cổ phiếu",
    d: "M4 6h16M7 12h10M10 18h4",
    kind: "will",
  },
  {
    href: "/heatmap",
    label: "Bản đồ nhiệt",
    d: "M3 3h18v18H3zM9 3v18M15 3v18M3 9h18M3 15h18",
    kind: "built",
  },
  {
    href: "/strategy-builder",
    label: "Xây chiến lược",
    d: "M6 4h5v5H6zM13 15h5v5h-5zM8.5 9v4a2 2 0 002 2h2.5",
    kind: "will",
  },
  {
    href: "/backtest",
    label: "Kiểm thử lịch sử",
    d: "M12 21a9 9 0 100-18 9 9 0 000 18zM12 7.5V12l3 2",
    kind: "will",
  },
  {
    href: "/portfolio",
    label: "Giao dịch giấy",
    d: "M3 7h18v12H3zM3 7l3-4h12l3 4M16 13h2",
    kind: "built",
  },
  {
    href: "/replay",
    label: "Chế độ Replay",
    d: "M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z",
    kind: "built",
  },
  {
    href: "/journal",
    label: "Sổ giao dịch",
    d: "M5 4h9a3 3 0 013 3v13H8a3 3 0 01-3-3V4zM5 17h11",
    kind: "will",
  },
  {
    href: "/quant",
    label: "Trợ lý Quant",
    d: "M12 3l2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5z",
    kind: "built",
  },
];

export type Mode = "stock" | "crypto";

// Crypto mode's list, from design/screens/Crypto-Main.dc.html's navRaw
// (same icons; its WILL flags kept, except the heatmap, built in Phase I).
// Crypto paper trading has no screen of its own -- the wallet lives on
// the Crypto overview (phase-i.md decision 10).
export const cryptoNavItems: NavItem[] = [
  { href: "/crypto", label: "Tổng quan", d: "M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z", kind: "built" },
  { href: "/crypto/BTCUSDT", label: "Biểu đồ & chỉ báo", d: "M4 20V11M9 20V4M14 20V14M19 20V8", kind: "built" },
  { href: "/crypto/screener", label: "Bộ lọc coin", d: "M4 6h16M7 12h10M10 18h4", kind: "will" },
  { href: "/heatmap?market=crypto", label: "Bản đồ nhiệt", d: "M3 3h18v18H3zM9 3v18M15 3v18M3 9h18M3 15h18", kind: "built" },
  { href: "/crypto/strategy-builder", label: "Xây chiến lược", d: "M6 4h5v5H6zM13 15h5v5h-5zM8.5 9v4a2 2 0 002 2h2.5", kind: "will" },
  { href: "/crypto/backtest", label: "Kiểm thử lịch sử", d: "M12 21a9 9 0 100-18 9 9 0 000 18zM12 7.5V12l3 2", kind: "will" },
  { href: "/crypto/paper", label: "Giao dịch giấy", d: "M3 7h18v12H3zM3 7l3-4h12l3 4M16 13h2", kind: "will" },
  { href: "/crypto/replay", label: "Chế độ Replay", d: "M11 6L4 12l7 6V6zM20 6l-7 6 7 6V6z", kind: "built" },
  { href: "/crypto/quant", label: "Trợ lý Quant", d: "M12 3l2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5z", kind: "will" },
];

export function navItemsFor(mode: Mode): NavItem[] {
  return mode === "crypto" ? cryptoNavItems : navItems;
}

// An item is active when the path matches its href's path (query ignored),
// or, for the chart item, any detail page of that market.
export function isNavActive(item: NavItem, pathname: string, mode: Mode): boolean {
  const path = item.href.split("?")[0];
  if (item.label.startsWith("Biểu đồ")) {
    return mode === "crypto"
      ? pathname.startsWith("/crypto/") && pathname !== "/crypto/replay"
      : pathname.startsWith("/stocks/");
  }
  return pathname === path;
}

// Settings -- rendered separately below the nav loop, always inactive
// style, per Main.dc.html's markup (not part of navRaw).
export const settingsItem = {
  href: "/settings/ai",
  label: "Cài đặt",
  d: "M12 15a3 3 0 100-6 3 3 0 000 6zM19.4 15a1.6 1.6 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.6 1.6 0 00-2.7 1.1V21a2 2 0 11-4 0v-.1A1.6 1.6 0 005.9 19.7l-.1.1a2 2 0 11-2.8-2.8l.1-.1A1.6 1.6 0 003 14.6a2 2 0 010-4h.1A1.6 1.6 0 004.3 8L4.2 8a2 2 0 112.8-2.8l.1.1a1.6 1.6 0 002.7-1.1V4a2 2 0 114 0v.1A1.6 1.6 0 0017 5.3l.1-.1A2 2 0 1119.9 8l-.1.1a1.6 1.6 0 001.1 2.7H21a2 2 0 010 4h-.1a1.6 1.6 0 00-1.5 1.2z",
};
