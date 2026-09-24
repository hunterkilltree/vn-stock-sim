"use client";

import { useState, useTransition } from "react";
import { openCryptoWalletAction } from "@/lib/cryptoActions";

export default function OpenCryptoWalletButton() {
  const [pending, start] = useTransition();
  const [error, setError] = useState<string | null>(null);
  return (
    <>
      <button
        type="button"
        disabled={pending}
        onClick={() => start(async () => setError((await openCryptoWalletAction()).error))}
        className="flex h-11 items-center justify-center rounded-[11px] bg-app-accent text-[13.5px] font-semibold text-app-accent-ink disabled:opacity-60"
      >
        {pending ? "Đang mở ví…" : "Mở ví crypto 10.000 USDT"}
      </button>
      {error && <p className="m-0 text-xs text-price-down">{error}</p>}
    </>
  );
}
