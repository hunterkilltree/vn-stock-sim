"use client";

// Last-resort fallback when the root layout itself fails; must render its
// own <html>/<body> and cannot depend on app styles or fonts.
export default function GlobalError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="vi">
      <body style={{ fontFamily: "system-ui, sans-serif", textAlign: "center", padding: "4rem 1rem" }}>
        <h1>VN Stock Sim</h1>
        <p>Dịch vụ đang tạm thời gián đoạn. Vui lòng thử lại sau ít phút.</p>
        <p>The service is temporarily unavailable. Please try again shortly.</p>
        <button onClick={() => reset()} style={{ padding: "0.5rem 1.25rem" }}>
          Thử lại / Retry
        </button>
      </body>
    </html>
  );
}
