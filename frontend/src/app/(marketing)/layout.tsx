import type { ReactNode } from "react";
import Navbar from "@/components/Navbar";

// Route-group layout for the light, luxalgo-style marketing/auth surface
// (/, /login; /register moved out in Phase G to follow Signup.dc.html's
// full-screen dark layout) -- these pages are NOT part of the design
// canvas's 20 screens, they predate it, so they keep their own nav
// instead of picking up SidebarNav/RailNav. A layout (a Server Component)
// is required here rather than each page importing <Navbar/> directly,
// because /login and /register are Client Components ("use client") and
// cannot import Navbar (a Server Component that reads cookies() via
// session.ts) themselves -- that broke the build with "You're importing
// a module that depends on next/headers ... in the Pages Router" (it was
// actually the client/server boundary, not the Pages Router) the first
// time this phase tried it. See phase-a.md / RESUME.md.
export default function MarketingLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <Navbar />
      {children}
    </>
  );
}
