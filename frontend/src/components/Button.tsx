import Link from "next/link";
import type { ReactNode } from "react";

type Variant = "primary" | "secondary";

const base =
  "inline-flex items-center gap-1.5 rounded-full px-6 py-2.5 text-sm font-medium transition-colors";
const variants: Record<Variant, string> = {
  // Solid black / white-on-black, matching luxalgo.com's primary CTA.
  primary: "bg-neutral-900 text-white hover:bg-neutral-700",
  // Off-white with a subtle border, matching their secondary CTA.
  secondary: "bg-neutral-50 text-neutral-900 border border-neutral-200 hover:bg-neutral-100",
};

type Props = {
  href: string;
  variant?: Variant;
  children: ReactNode;
  className?: string;
};

export default function Button({ href, variant = "primary", children, className = "" }: Props) {
  return (
    <Link href={href} className={`${base} ${variants[variant]} ${className}`}>
      {children}
    </Link>
  );
}
