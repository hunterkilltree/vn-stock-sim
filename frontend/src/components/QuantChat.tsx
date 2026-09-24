"use client";

import Link from "next/link";
import { useRef, useState } from "react";
import QuantAssistantCard, { SparkIcon } from "@/components/QuantAssistantCard";
import QuantResultsPanel from "@/components/QuantResultsPanel";
import { runBacktestAction } from "@/lib/quantActions";
import {
  activeConversationStore,
  connectionOf,
  historyStore,
  isConfigured,
  recordUsage,
  saveConversation,
  settingsStore,
  type Conversation,
  type Turn,
} from "@/lib/quantSettings";
import type { QuantChatResponse, QuantError, QuantScreen, QuantStrategy } from "@/lib/quantTypes";

const SUGGESTIONS = ["Giải thích MACD bằng ví dụ", "So sánh FPT với VNM", "Mã nào đang nằm trên SMA 50?"];
const EXCHANGES = ["ALL", "HOSE", "HNX", "UPCOM"] as const;
const YEARS = [1, 3, 5];
const PROVIDER_NAMES = { claude: "Claude", openai: "OpenAI", custom: "Máy chủ riêng" } as const;

type AssistantTurn = Extract<Turn, { role: "assistant" }>;

function newId(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Math.random()).slice(2);
}

// Builds the chat history the backend accepts: user text, and each
// earlier assistant answer as its validated JSON plan so the model can
// refine it (phase-h.md). At most 20 messages, starting with a user turn.
type ChatMessage = { role: "user" | "assistant"; content: string };

function toMessages(turns: Turn[]): ChatMessage[] {
  const msgs = turns.flatMap((t): ChatMessage[] =>
    t.role === "user"
      ? [{ role: "user" as const, content: t.text }]
      : t.role === "assistant"
        ? [{ role: "assistant" as const, content: JSON.stringify(t.response.plan) }]
        : [],
  );
  const tail = msgs.slice(-20);
  while (tail.length && tail[0].role !== "user") tail.shift();
  return tail;
}

function latest<T>(turns: Turn[], pick: (t: AssistantTurn) => T | null): T | null {
  for (let i = turns.length - 1; i >= 0; i--) {
    const t = turns[i];
    if (t.role === "assistant") {
      const v = pick(t);
      if (v) return v;
    }
  }
  return null;
}

// The symbol a strategy is backtested on: the one the user named, else
// the first stock the latest screen found, else FPT.
function backtestSymbolFor(turns: Turn[], index: number, strategy: QuantStrategy): string {
  if (strategy.symbol) return strategy.symbol;
  for (let i = index; i >= 0; i--) {
    const t = turns[i];
    if (t.role === "assistant" && t.response.screen?.matches.length) return t.response.screen.matches[0].symbol;
  }
  return "FPT";
}

export default function QuantChat({ initialQuestion }: { initialQuestion: string }) {
  const settings = settingsStore.useValue();
  const history = historyStore.useValue();
  const activeId = activeConversationStore.useValue();
  const conversation = history.find((c) => c.id === activeId) ?? null;
  const turns = conversation?.turns ?? [];

  const [input, setInput] = useState(initialQuestion);
  const [pending, setPending] = useState(false);
  const [exchange, setExchange] = useState<(typeof EXCHANGES)[number]>("ALL");
  const [years, setYears] = useState(5);
  const [backtesting, setBacktesting] = useState<number | null>(null);
  const threadEnd = useRef<HTMLDivElement>(null);

  const configured = isConfigured(settings);
  const screen: QuantScreen | null = latest(turns, (t) => t.response.screen);
  const strategy: QuantStrategy | null = latest(turns, (t) => t.response.strategy);

  function persist(conv: Conversation) {
    saveConversation(conv);
    activeConversationStore.write(conv.id);
    requestAnimationFrame(() => threadEnd.current?.scrollIntoView({ behavior: "smooth", block: "end" }));
  }

  async function runBacktest(conv: Conversation, index: number) {
    const t = conv.turns[index];
    if (t.role !== "assistant" || !t.response.strategy) return;
    setBacktesting(index);
    const symbol = backtestSymbolFor(conv.turns, index, t.response.strategy);
    const res = await runBacktestAction(symbol, t.response.strategy, years);
    setBacktesting(null);
    const fresh = historyStore.read().find((c) => c.id === conv.id) ?? conv;
    const updated = fresh.turns.map((x, i) => (i === index && x.role === "assistant" ? { ...x, backtest: res.result, backtestError: res.error } : x));
    persist({ ...fresh, turns: updated, updatedAt: new Date().toISOString() });
  }

  async function send(text: string) {
    const question = text.trim();
    if (!question || pending || !configured) return;
    setInput("");
    setPending(true);

    const base: Conversation = conversation ?? {
      id: newId(),
      title: question.slice(0, 60),
      updatedAt: new Date().toISOString(),
      turns: [],
    };
    const withQuestion: Conversation = { ...base, turns: [...base.turns, { role: "user", text: question }], updatedAt: new Date().toISOString() };
    persist(withQuestion);

    let next: Conversation;
    try {
      const res = await fetch("/api/quant/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...connectionOf(settings),
          messages: toMessages(withQuestion.turns),
          scope: settings.scope,
          defaultExchange: exchange,
        }),
      });
      const payload = await res.json().catch(() => null);
      if (!res.ok) {
        const err = payload as QuantError | null;
        const status = err?.providerStatus ? ` (${err.providerStatus})` : "";
        next = { ...withQuestion, turns: [...withQuestion.turns, { role: "error", text: `${err?.message ?? "Quant không trả lời được."}${status}` }] };
      } else {
        const response = payload as QuantChatResponse;
        recordUsage(response.usage.inputTokens + response.usage.outputTokens, !!response.strategy);
        next = { ...withQuestion, turns: [...withQuestion.turns, { role: "assistant", response }] };
      }
    } catch {
      next = { ...withQuestion, turns: [...withQuestion.turns, { role: "error", text: "Không thể kết nối tới máy chủ." }] };
    }
    next.updatedAt = new Date().toISOString();
    persist(next);
    setPending(false);

    const lastIndex = next.turns.length - 1;
    const last = next.turns[lastIndex];
    if (settings.autoBacktest && last.role === "assistant" && last.response.strategy) {
      await runBacktest(next, lastIndex);
    }
  }

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-[18px] p-[24px_28px]">
      <header className="flex items-center justify-between gap-6">
        <div className="flex items-center gap-[14px]">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-app-accent-border bg-app-accent-surface">
            <SparkIcon size={20} />
          </span>
          <div className="flex flex-col gap-[3px]">
            <h1 className="m-0 font-display text-[25px] font-bold tracking-[-0.015em]">Trợ lý Quant</h1>
            <span className="text-[12.5px] text-app-text-muted">Lọc cổ phiếu và dựng chiến lược bằng câu tiếng Việt · HOSE, HNX, UPCOM</span>
          </div>
        </div>
        <div className="flex items-center gap-[10px]">
          <Link
            href="/settings/ai"
            className="flex h-[34px] items-center gap-[7px] rounded-[9px] border border-app-border bg-app-surface px-3 text-[11.5px] text-app-text-3"
          >
            <span className="h-[7px] w-[7px] rounded-full" style={{ background: configured ? "var(--price-up)" : "var(--app-text-faint)" }} />
            <span>{configured ? `${PROVIDER_NAMES[settings.provider]} · ${settings.models[settings.provider]}` : "Chưa kết nối mô hình"}</span>
          </Link>
          <button
            type="button"
            onClick={() => activeConversationStore.write(null)}
            className="flex h-10 items-center gap-2 rounded-[10px] border border-app-border bg-app-surface-2 px-[14px] text-[13px] font-medium text-app-text"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" aria-hidden="true">
              <path d="M12 5v14M5 12h14" />
            </svg>
            <span>Hội thoại mới</span>
          </button>
        </div>
      </header>

      <div className="flex min-h-0 flex-1 gap-5">
        <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-3">
          <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto pr-1" aria-live="polite">
            {!configured && (
              <div className="flex flex-col gap-2 rounded-[14px] border border-app-accent-border bg-app-accent-surface p-4">
                <span className="text-[14px] font-semibold">Kết nối một mô hình AI để bắt đầu</span>
                <p className="m-0 text-[12.5px] leading-[1.5] text-app-text-3">
                  Quant dùng khoá API của riêng bạn (Claude, OpenAI hoặc máy chủ tương thích OpenAI). VN Stock Sim không trả phí thay bạn và
                  không lưu khoá trên máy chủ.
                </p>
                <Link href="/settings/ai" className="flex h-10 w-fit items-center rounded-[10px] bg-app-accent px-4 text-[13px] font-semibold text-app-accent-ink">
                  Mở Cài đặt › Mô hình AI
                </Link>
              </div>
            )}

            {turns.length === 0 && configured && (
              <p className="m-0 max-w-[560px] text-[13.5px] leading-[1.6] text-app-text-3">
                Hỏi bằng tiếng Việt: điều kiện lọc, ý tưởng chiến lược, giải thích chỉ báo. Quant chỉ đọc yêu cầu thành điều kiện; kết quả
                lọc và kiểm thử do VN Stock Sim tự tính trên dữ liệu của ứng dụng.
              </p>
            )}

            {turns.map((t, i) =>
              t.role === "user" ? (
                <div key={i} className="max-w-[80%] self-end rounded-[14px] rounded-br-[4px] border border-app-border bg-app-surface-2 p-[12px_14px]">
                  <p className="m-0 whitespace-pre-wrap text-[13.5px] leading-[1.55]">{t.text}</p>
                </div>
              ) : t.role === "assistant" ? (
                <QuantAssistantCard
                  key={i}
                  response={t.response}
                  backtest={t.backtest}
                  backtestError={t.backtestError}
                  backtestSymbol={t.response.strategy ? backtestSymbolFor(turns, i, t.response.strategy) : null}
                  years={years}
                  runningBacktest={backtesting === i}
                  onRunBacktest={() => conversation && runBacktest(conversation, i)}
                />
              ) : (
                <div key={i} role="alert" className="flex gap-3 rounded-[12px] border border-price-down/40 bg-price-down/10 p-[12px_14px] text-[13px] text-price-down">
                  {t.text}
                </div>
              ),
            )}

            {pending && (
              <div className="flex items-center gap-3 text-[13px] text-app-text-muted">
                <span className="flex h-[30px] w-[30px] items-center justify-center rounded-[9px] bg-app-surface-3">
                  <SparkIcon />
                </span>
                Quant đang đọc yêu cầu…
              </div>
            )}
            <div ref={threadEnd} />
          </div>

          <div className="flex flex-wrap gap-2">
            {SUGGESTIONS.map((s) => (
              <button
                key={s}
                type="button"
                disabled={!configured || pending}
                onClick={() => send(s)}
                className="h-8 rounded-full border border-app-border bg-app-surface px-3 text-[12px] text-app-text-3 disabled:opacity-50"
              >
                {s}
              </button>
            ))}
          </div>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              send(input);
            }}
            className="flex flex-col gap-2 rounded-[14px] border border-app-border bg-app-surface p-[12px_14px]"
          >
            <label htmlFor="ask" className="sr-only">Câu hỏi cho trợ lý Quant</label>
            <textarea
              id="ask"
              rows={2}
              value={input}
              maxLength={4000}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  send(input);
                }
              }}
              placeholder="Hỏi bằng tiếng Việt: điều kiện lọc, ý tưởng chiến lược, giải thích chỉ báo…"
              className="resize-none border-0 bg-transparent text-[13.5px] leading-[1.5] text-app-text outline-none placeholder:text-app-text-faint"
            />
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setExchange(EXCHANGES[(EXCHANGES.indexOf(exchange) + 1) % EXCHANGES.length])}
                className="h-8 rounded-lg border border-app-border px-[10px] text-[12px] text-app-text-3"
              >
                Sàn: {exchange === "ALL" ? "Tất cả" : exchange}
              </button>
              <button
                type="button"
                onClick={() => setYears(YEARS[(YEARS.indexOf(years) + 1) % YEARS.length])}
                className="h-8 rounded-lg border border-app-border px-[10px] text-[12px] text-app-text-3"
              >
                Khoảng: {years} năm
              </button>
              <span className="flex h-8 items-center rounded-lg border border-app-hairline px-[10px] text-[12px] text-app-text-faint">Nến: ngày</span>
              <div className="flex-1" />
              <span className="text-[11.5px] text-app-text-faint">Enter để gửi</span>
              <button
                type="submit"
                aria-label="Gửi câu hỏi"
                disabled={!configured || pending || !input.trim()}
                className="flex h-11 w-11 items-center justify-center rounded-xl bg-app-accent text-app-accent-ink disabled:opacity-50"
              >
                <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M5 12h13M13 6l6 6-6 6" />
                </svg>
              </button>
            </div>
          </form>
        </div>

        <QuantResultsPanel key={conversation?.id ?? "none"} screen={screen} strategy={strategy} years={years} />
      </div>
    </div>
  );
}
