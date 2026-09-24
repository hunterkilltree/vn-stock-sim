// Starting-capital presets from design/screens/Signup.dc.html's capRaw.
// The USDT preset opens a crypto wallet (Phase I, phase-i.md decision 9).
export type CapitalPreset = { amount: number; label: string; note: string; disabled?: boolean };

export const STOCK_CAPITAL_PRESETS: CapitalPreset[] = [
  { amount: 1_000_000_000, label: "1 tỷ ₫", note: "Mặc định" },
  { amount: 500_000_000, label: "500 triệu ₫", note: "Sát vốn thật" },
];

export const CRYPTO_CAPITAL_PRESET: CapitalPreset = {
  amount: 10_000,
  label: "10.000 USDT",
  note: "Cho crypto",
};

export function isStockCapitalPreset(amount: number): boolean {
  return STOCK_CAPITAL_PRESETS.some((p) => p.amount === amount);
}
