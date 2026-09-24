import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono, Lora, Be_Vietnam_Pro, IBM_Plex_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

// Dark app-shell font stack from the design canvas (see phase-a.md):
// Lora for headings, Be Vietnam Pro for UI text, IBM Plex Mono for every
// number. Additive -- the marketing/auth pages keep using Geist via
// --font-sans, unaffected by these.
const lora = Lora({
  variable: "--font-lora",
  subsets: ["latin", "vietnamese"],
});

const beVietnamPro = Be_Vietnam_Pro({
  variable: "--font-be-vietnam",
  weight: ["400", "500", "600"],
  subsets: ["latin", "vietnamese"],
});

const plexMono = IBM_Plex_Mono({
  variable: "--font-plex-mono-raw",
  weight: ["400", "500"],
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "VN Stock Sim",
  description: "Trade the past before you trade the future — simulate Vietnamese stock trading.",
};

// viewport-fit=cover lets the phone shell pad itself with
// env(safe-area-inset-*) instead of leaving bars under the home
// indicator (phase-j.md decision 15).
export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  themeColor: "#0f0f0e",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} ${lora.variable} ${beVietnamPro.variable} ${plexMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-white text-neutral-900">
        <div className="flex flex-1 flex-col">{children}</div>
      </body>
    </html>
  );
}
