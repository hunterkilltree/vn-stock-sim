// VN-locale number formatting + market-color helpers, ported directly
// from design/screens/Main.dc.html's renderVals() script block (num/
// sign/tone) so the formatting matches the design exactly, not a
// reinvented approximation. See phase-c.md.

export function formatVN(value: number, decimals: number): string {
  return value.toLocaleString("vi-VN", {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

// "+1,23%" / "-1,23%" / "1,23%" (no sign for exactly zero), matching the
// design's sign(): a literal minus sign U+2212, not a hyphen, for the
// negative case.
export function signVN(value: number, decimals: number): string {
  const prefix = value > 0 ? "+" : value < 0 ? "−" : "";
  return prefix + formatVN(Math.abs(value), decimals);
}

// up/down/reference hex, matching the design's tone() exactly -- these
// are the app-* CSS custom property values from globals.css, inlined
// here because this helper returns a value for an inline style attr
// (SVG stroke, dynamic text color), not a Tailwind class.
export function tone(value: number): string {
  if (value > 0) return "#35C77F";
  if (value < 0) return "#FF5C5C";
  return "#F0C243";
}

// Volume in the design's short form, e.g. "4,1 tr" (triệu/million) --
// design/DESIGN-SYSTEM.md section 8's "Short forms: tr / tỷ / nghìn tỷ".
// V1's mock volumes never reach tỷ (billion), so only the tr case is
// implemented; add tỷ/nghìn tỷ if a future data source needs them.
export function formatVolumeVN(volume: number): string {
  return `${formatVN(volume / 1_000_000, 1)} tr`;
}

// The Detail screen (design/screens/Detail.dc.html) quotes every price
// in thousands of VND (its own order-ticket label is "Gia (nghin d)"),
// e.g. FPT at "128,50" means 128,500 VND -- this app's backend returns
// raw VND everywhere (symbol.Detail.LastPrice etc.), so every Detail-
// screen price display divides by 1000 through this one helper rather
// than each component reimplementing the convention ad hoc.
export function formatThousandsVN(rawVnd: number, decimals = 2): string {
  return formatVN(rawVnd / 1000, decimals);
}

// Sector palette for the Portfolio page's allocation bars
// (Portfolio.dc.html's allocRaw), cycled by first-seen order since the
// backend returns sector names, not colors. "Tiền mặt" (cash) always
// gets the fixed muted gray the design uses for it, matching allocRaw's
// own ['Tiền mặt', 19.8, '#6F6C63'].
const SECTOR_PALETTE = ["#E08A3C", "#4FD3E8", "#7FA2FF", "#C08BFF", "#35C77F", "#F0C243"];

export function sectorColor(sector: string, index: number): string {
  if (sector === "Tiền mặt") return "#6F6C63";
  return SECTOR_PALETTE[index % SECTOR_PALETTE.length];
}

// Market cap short form, matching design/DESIGN-SYSTEM.md section 8's
// "Short forms: tr / ty / nghin ty" (million/billion/thousand-billion).
export function formatMarketCapVN(rawVnd: number): string {
  if (rawVnd >= 1_000_000_000_000) {
    return `${formatVN(rawVnd / 1_000_000_000_000, 1)} nghìn tỷ`;
  }
  return `${formatVN(rawVnd / 1_000_000_000, 1)} tỷ`;
}

// Crypto prices (USDT) span 64.820,50 (BTC) to 0,0000392 (VNDC): keep 2
// decimals above 1 and ~4 significant digits below (Crypto-Main's
// "0,00412", "0,4820").
export function formatCryptoPrice(v: number): string {
  if (!isFinite(v) || v === 0) return "0";
  const abs = Math.abs(v);
  if (abs >= 1) return formatVN(v, 2);
  const decimals = Math.min(10, Math.max(4, 3 - Math.floor(Math.log10(abs))));
  return v.toLocaleString("vi-VN", { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
}

// Compact amounts in the design's short forms: "1,28 ngh.tỷ USD",
// "28,4 tỷ USDT", "162 tr".
export function formatCompact(v: number, unit = ""): string {
  const suffix = unit ? ` ${unit}` : "";
  const abs = Math.abs(v);
  if (abs >= 1e12) return `${formatVN(v / 1e12, 2)} ngh.tỷ${suffix}`;
  if (abs >= 1e9) return `${formatVN(v / 1e9, 1)} tỷ${suffix}`;
  if (abs >= 1e6) return `${formatVN(v / 1e6, 1)} tr${suffix}`;
  return `${formatVN(v, 0)}${suffix}`;
}

// Crypto amounts: up to 8 decimals, trailing zeros trimmed ("0,025").
export function formatAmount(v: number, maxDecimals = 8): string {
  return v.toLocaleString("vi-VN", { minimumFractionDigits: 0, maximumFractionDigits: maxDecimals });
}
