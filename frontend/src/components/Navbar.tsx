import Link from "next/link";
import Button from "@/components/Button";

// Sticky top nav, following luxalgo.com's pattern: logo left, nav links
// next to it, primary CTA on the far right, white background with a
// hairline bottom border.
export default function Navbar() {
  return (
    <header className="sticky top-0 z-10 border-b border-neutral-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-8 py-4">
        <div className="flex items-center gap-8">
          <Link href="/" className="text-base font-semibold tracking-tight text-neutral-900">
            VN Stock Sim
          </Link>
          <nav className="hidden items-center gap-6 text-sm text-neutral-600 sm:flex">
            <Link href="/stocks" className="hover:text-neutral-900">
              Stocks
            </Link>
            <Link href="/chart" className="hover:text-neutral-900">
              Chart
            </Link>
          </nav>
        </div>
        <Button href="/stocks">Browse Stocks</Button>
      </div>
    </header>
  );
}
