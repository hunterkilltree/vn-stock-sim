// Starting-capital presets from design/screens/Signup.dc.html's capRaw.
// The USDT preset is shown but disabled until a crypto market exists
// (Phase I) -- see phase-g.md decision 6.
export type CapitalPreset = { amount: number; label: string; note: string; disabled?: boolean };

export const STOCK_CAPITAL_PRESETS: CapitalPreset[] = [
  { amount: 1_000_000_000, label: "1 tỷ ₫", note: "Mặc định" },
  { amount: 500_000_000, label: "500 triệu ₫", note: "Sát vốn thật" },
];

export const CRYPTO_CAPITAL_PRESET: CapitalPreset = {
  amount: 10_000,
  label: "10.000 USDT",
  note: "Crypto — sắp có",
  disabled: true,
};

export function isStockCapitalPreset(amount: number): boolean {
  return STOCK_CAPITAL_PRESETS.some((p) => p.amount === amount);
}
