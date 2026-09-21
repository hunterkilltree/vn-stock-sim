"use client";

import { useEffect, useRef } from "react";
import {
  createChart,
  CandlestickSeries,
  LineSeries,
  type IChartApi,
  type UTCTimestamp,
} from "lightweight-charts";
import type { Bar, IndicatorPoint } from "@/lib/api";

type Props = {
  bars: Bar[];
  sma20?: IndicatorPoint[];
  theme?: "light" | "dark";
  heightClassName?: string;
};

// Renders bars fetched server-side (see stocks/[symbol]/page.tsx) with
// TradingView's lightweight-charts library -- deliberately not the
// TradingView Charting Library itself (see RESUME.md: that one needs
// TradingView's gated access approval). Data is passed in as props rather
// than fetched here because this runs in the browser: in the Docker demo,
// the browser cannot resolve the "backend" hostname (that only exists on
// the Compose-internal network), so client-side fetches to the API would
// fail there even though server-side fetches work fine.
export default function StockChart({ bars, sma20, theme = "light", heightClassName }: Props) {
  const container = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);

  useEffect(() => {
    const el = container.current;
    if (!el || chartRef.current) return;

    const dark = theme === "dark";
    const chart = createChart(el, {
      autoSize: true,
      layout: {
        textColor: dark ? "#a3a3a3" : "#374151",
        background: { color: "transparent" },
      },
      grid: {
        vertLines: { color: dark ? "#262626" : "#f3f4f6" },
        horzLines: { color: dark ? "#262626" : "#f3f4f6" },
      },
      timeScale: { timeVisible: false, borderColor: dark ? "#404040" : "#e5e7eb" },
      rightPriceScale: { borderColor: dark ? "#404040" : "#e5e7eb" },
    });
    chartRef.current = chart;

    const candles = chart.addSeries(CandlestickSeries, {
      upColor: "#16a34a",
      downColor: "#dc2626",
      borderVisible: false,
      wickUpColor: "#16a34a",
      wickDownColor: "#dc2626",
    });
    candles.setData(
      bars.map((b) => ({
        time: b.time as UTCTimestamp,
        open: b.open,
        high: b.high,
        low: b.low,
        close: b.close,
      })),
    );

    if (sma20 && sma20.length > 0) {
      const sma = chart.addSeries(LineSeries, {
        color: "#2563eb",
        lineWidth: 2,
      });
      sma.setData(sma20.map((p) => ({ time: p.time as UTCTimestamp, value: p.value })));
    }

    chart.timeScale().fitContent();

    return () => {
      chart.remove();
      chartRef.current = null;
    };
  }, [bars, sma20, theme]);

  return <div ref={container} className={heightClassName ?? "h-[400px] w-full lg:h-[480px]"} />;
}
