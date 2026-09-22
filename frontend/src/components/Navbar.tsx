import Link from "next/link";
import Button from "@/components/Button";
import { getSessionUser } from "@/lib/session";
import { logoutAction } from "@/lib/authActions";

// Sticky top nav, following luxalgo.com's pattern: logo left, nav links
// next to it, primary CTA on the far right, white background with a
// hairline bottom border. Async because it reads the session (GET
// /auth/me via the httpOnly cookie) to decide between "Log In/Sign Up"
// and "Log Out" -- see session.ts/authActions.ts.
export default async function Navbar() {
  const user = await getSessionUser();

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

        {user ? (
          <div className="flex items-center gap-4">
            <span className="hidden text-sm text-neutral-600 sm:inline">{user.displayName}</span>
            <form action={logoutAction}>
              <button
                type="submit"
                className="rounded-full border border-neutral-200 px-5 py-2 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50"
              >
                Log Out
              </button>
            </form>
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <Link href="/login" className="text-sm font-medium text-neutral-600 hover:text-neutral-900">
              Log In
            </Link>
            <Button href="/register">Sign Up</Button>
          </div>
        )}
      </div>
    </header>
  );
}
