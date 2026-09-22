// Single source of truth for the dark app shell's navigation, shared by
// SidebarNav (full, 236px) and RailNav (icon-only, 72px) so the two
// shells never drift out of sync -- see FULL-APP-PLAN.md Phase A and
// phase-a.md.
import type { ComponentType, SVGProps } from "react";
import {
  MarketIcon,
  DetailIcon,
  PortfolioIcon,
  ReplayIcon,
  QuantIcon,
  SettingsIcon,
} from "@/components/icons";

export type NavItem = {
  href: string;
  label: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  // "built": a real route exists today, renders as a normal link.
  // "soon": scheduled in a later phase (Phases D-H) but not built yet --
  //   renders disabled rather than link to a 404.
  // "will": true Phase K backlog (screener, standalone heatmap, strategy
  //   builder, standalone backtesting UI, standalone trade journal) --
  //   renders disabled with the canvas's own muted "WILL" pill.
  kind: "built" | "soon" | "will";
};

export const navItems: NavItem[] = [
  { href: "/stocks", label: "Market", icon: MarketIcon, kind: "built" },
  { href: "/chart", label: "Chart", icon: DetailIcon, kind: "built" },
  { href: "/portfolio", label: "Portfolio", icon: PortfolioIcon, kind: "soon" },
  { href: "/replay", label: "Replay", icon: ReplayIcon, kind: "soon" },
  { href: "/quant", label: "Quant", icon: QuantIcon, kind: "soon" },
  { href: "/settings", label: "Settings", icon: SettingsIcon, kind: "soon" },
];
