// Client-side SMA / RSI(14, Wilder) / MACD(12,26,9) for crypto charts.
// The backend's indicator endpoints read the stock market service, so
// crypto pages compute the same indicators from the bars they fetched
// -- same formulas as backend/internal/market (sma.go, rsi.go, macd.go).
import type { Bar, IndicatorMultiPoint, IndicatorPoint } from "./api";

export function sma(bars: Bar[], period: number): IndicatorPoint[] {
  const out: IndicatorPoint[] = [];
  let sum = 0;
  bars.forEach((b, i) => {
    sum += b.close;
    if (i >= period) sum -= bars[i - period].close;
    if (i >= period - 1) out.push({ time: b.time, value: sum / period });
  });
  return out;
}

export function rsi(bars: Bar[], period = 14): IndicatorPoint[] {
  const out: IndicatorPoint[] = [];
  let gain = 0;
  let loss = 0;
  for (let i = 1; i < bars.length; i++) {
    const d = bars[i].close - bars[i - 1].close;
    const g = Math.max(d, 0);
    const l = Math.max(-d, 0);
    if (i < period) {
      gain += g;
      loss += l;
      continue;
    }
    if (i === period) {
      gain = (gain + g) / period;
      loss = (loss + l) / period;
    } else {
      gain = (gain * (period - 1) + g) / period;
      loss = (loss * (period - 1) + l) / period;
    }
    const v = loss === 0 ? (gain === 0 ? 50 : 100) : 100 - 100 / (1 + gain / loss);
    out.push({ time: bars[i].time, value: v });
  }
  return out;
}

function ema(values: number[], period: number): (number | null)[] {
  const out: (number | null)[] = values.map(() => null);
  if (values.length < period) return out;
  const k = 2 / (period + 1);
  let prev = values.slice(0, period).reduce((a, b) => a + b, 0) / period;
  out[period - 1] = prev;
  for (let i = period; i < values.length; i++) {
    prev = values[i] * k + prev * (1 - k);
    out[i] = prev;
  }
  return out;
}

export function macd(bars: Bar[], fast = 12, slow = 26, signal = 9): IndicatorMultiPoint[] {
  const closes = bars.map((b) => b.close);
  const f = ema(closes, fast);
  const s = ema(closes, slow);
  const line: { time: number; v: number }[] = [];
  bars.forEach((b, i) => {
    if (f[i] !== null && s[i] !== null) line.push({ time: b.time, v: (f[i] as number) - (s[i] as number) });
  });
  const sig = ema(line.map((l) => l.v), signal);
  const out: IndicatorMultiPoint[] = [];
  line.forEach((l, i) => {
    if (sig[i] !== null) {
      out.push({ time: l.time, values: { macd: l.v, signal: sig[i] as number, histogram: l.v - (sig[i] as number) } });
    }
  });
  return out;
}
