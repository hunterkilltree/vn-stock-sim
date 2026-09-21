"use client";

import { useEffect, useRef } from "react";

type Props = {
  symbol?: string;
  theme?: "light" | "dark";
};

// Embeds TradingView's public "Advanced Chart" widget
// (https://www.tradingview.com/widget/advanced-chart/) by injecting the
// script tag TradingView's own docs specify. This is the free, no-approval
// embed product, NOT the self-hosted Charting Library described in
// charting-library-integration.md -- that one requires TradingView's
// GitHub-gated access approval (see RESUME.md). This widget is step 1 of
// the plan: prove a TradingView chart renders in the app at all, before
// building the licensed integration with a custom Datafeed against our
// own /api/v1/market/bars.
//
// To change symbol/theme at runtime, remount this component with a
// `key={symbol}` prop from the parent rather than relying on this effect
// to swap the widget in place -- TradingView's script reads
// document.currentScript.parentElement once, synchronously-ish, so
// tearing the DOM down and rebuilding it under the same effect (e.g. on a
// prop change, or React Strict Mode's dev-only double-invoke) can detach
// the script node before it finishes and throw inside TradingView's code.
export default function TradingViewWidget({ symbol = "BITSTAMP:BTCUSD", theme = "light" }: Props) {
  const container = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = container.current;
    // Idempotent: if this container already has the widget div (e.g.
    // React Strict Mode's double effect invocation in dev), do nothing --
    // rebuilding here would detach TradingView's in-flight script.
    if (!el || el.childElementCount > 0) return;

    const widgetDiv = document.createElement("div");
    widgetDiv.className = "tradingview-widget-container__widget";
    widgetDiv.style.height = "100%";
    widgetDiv.style.width = "100%";
    el.appendChild(widgetDiv);

    const script = document.createElement("script");
    script.src = "https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js";
    script.type = "text/javascript";
    script.async = true;
    // TradingView's widget script reads its config from its own text
    // content as JSON -- this is the documented mechanism, not eval'd by
    // us; the script itself is TradingView's, loaded from their CDN.
    script.text = JSON.stringify({
      autosize: true,
      symbol,
      interval: "D",
      timezone: "Etc/UTC",
      theme,
      style: "1",
      locale: "en",
      allow_symbol_change: true,
      calendar: false,
      support_host: "https://www.tradingview.com",
    });
    el.appendChild(script);
  }, [symbol, theme]);

  return (
    <div
      className="tradingview-widget-container"
      ref={container}
      style={{ height: "600px", width: "100%" }}
    />
  );
}
