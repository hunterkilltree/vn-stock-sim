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
