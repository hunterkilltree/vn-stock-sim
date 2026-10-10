// Stock exposure (stocks' market value as % of equity) before and after
// an order, checked against the market regime's cap (phase-market-risk.md
// slice 2). Advice only: the ticket warns, nothing is blocked.

export type Exposure = {
  stockValue: number; // market value of stock positions, VND
  equity: number; // cash + positions, VND
  capPct: number; // regime cap, % of equity
  levelLabel: string;
  color: string;
};

const LOT_SIZE = 100;

export function exposurePct(stockValue: number, equity: number): number {
  return equity > 0 ? (stockValue / equity) * 100 : 0;
}

// The fee leaves the account, so equity shrinks by it either way.
export function exposureAfter(e: Exposure, side: "buy" | "sell", orderValue: number, fee: number): number {
  const stock = side === "buy" ? e.stockValue + orderValue : Math.max(0, e.stockValue - orderValue);
  return exposurePct(stock, e.equity - fee);
}

// Largest buy (whole lots) that keeps exposure at or under the cap; 0 if
// the portfolio is already over it.
export function maxSharesUnderCap(e: Exposure, priceVnd: number, feeRate: number): number {
  if (priceVnd <= 0) return 0;
  const room = (e.capPct / 100) * e.equity - e.stockValue;
  if (room <= 0) return 0;
  return Math.floor(room / (priceVnd * (1 + feeRate)) / LOT_SIZE) * LOT_SIZE;
}
