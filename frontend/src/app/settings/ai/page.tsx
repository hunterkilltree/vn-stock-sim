export const metadata = { title: "Mô hình AI — Cài đặt — VN Stock Sim" };

// Header copy from design/screens/Settings-AI.dc.html. The provider/key/
// permissions form itself is Phase H (FULL-APP-PLAN.md section 10) --
// no inputs are rendered here until they can actually be saved.
export default function SettingsAIPage() {
  return (
    <>
      <header className="flex flex-col gap-[5px]">
        <h2 className="m-0 font-display text-[25px] font-bold tracking-[-0.015em]">Mô hình AI</h2>
        <span className="text-[12.5px] text-app-text-muted">
          Chọn mô hình chạy Trợ lý Quant và quyết định dữ liệu nào được gửi đi
        </span>
      </header>

      <section className="flex max-w-[640px] flex-col gap-3 rounded-[14px] border border-app-border bg-app-surface p-[18px]">
        <h3 className="m-0 text-[15px] font-semibold">Chưa kết nối mô hình nào</h3>
        <p className="m-0 text-[13px] leading-[1.6] text-app-text-3">
          Trợ lý Quant sẽ dùng khoá API của riêng bạn (Claude, OpenAI hoặc máy chủ tương thích OpenAI). Khoá chỉ lưu trên thiết bị
          này. Phần chọn nhà cung cấp, nhập khoá và kiểm tra kết nối sẽ có ở bản cập nhật tiếp theo — hiện VN Stock Sim chưa gửi
          dữ liệu nào tới mô hình AI.
        </p>
      </section>

      <div className="flex max-w-[640px] items-start gap-[9px] rounded-[10px] border border-app-warn-border bg-app-warn-surface p-[11px_12px]">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#F0C243" strokeWidth="1.8" strokeLinecap="round" className="mt-[1px] shrink-0" aria-hidden="true">
          <path d="M12 9v5M12 17h0M12 3l9 17H3L12 3z" />
        </svg>
        <p className="m-0 text-[11.5px] leading-[1.5] text-app-warn-text">
          Dù dùng mô hình nào, Quant cũng không đặt lệnh thật. Mọi quyền ở đây chỉ áp dụng cho tài khoản giấy.
        </p>
      </div>
    </>
  );
}
