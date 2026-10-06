"use client";

import Link from "next/link";

// Route-level fallback: any Server/Client Component that throws (e.g. the
// backend is down and a page's data fetch fails) renders this instead of
// a blank 500 page.
export default function Error({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
      <h1 className="text-2xl font-semibold">Không thể tải dữ liệu</h1>
      <p className="max-w-md text-neutral-600">
        Máy chủ dữ liệu đang tạm thời không phản hồi. Vui lòng thử lại sau ít phút.
        <br />
        The data service is temporarily unavailable. Please try again shortly.
      </p>
      <div className="flex gap-3">
        <button onClick={() => reset()} className="rounded-full bg-neutral-900 px-5 py-2 text-white">
          Thử lại / Retry
        </button>
        <Link href="/" className="rounded-full border border-neutral-300 px-5 py-2">
          Trang chủ / Home
        </Link>
      </div>
    </main>
  );
}
