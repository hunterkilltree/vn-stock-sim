"use client";

// Browser-only storage for Phase H (phase-h.md): the AI settings including
// the user's API keys, the monthly token counter, the last connection
// test, and Quant conversations. The keys never go to the VN Stock Sim
// database -- they only leave this browser inside each request to
// /api/quant/*, which forwards them to the provider.
import { useSyncExternalStore } from "react";
import type { QuantChatResponse, QuantProvider, BacktestResult } from "./quantTypes";

export type QuantSettings = {
  provider: QuantProvider;
  apiKeys: Record<QuantProvider, string>;
  models: Record<QuantProvider, string>;
  baseUrl: string;
  timeoutSeconds: number;
  temperature: number;
  autoBacktest: boolean;
  scope: { prices: boolean; indicators: boolean; watchlist: boolean; positions: boolean };
};

export const DEFAULT_SETTINGS: QuantSettings = {
  provider: "claude",
  apiKeys: { claude: "", openai: "", custom: "" },
  models: { claude: "claude-opus-5", openai: "", custom: "" },
  baseUrl: "",
  timeoutSeconds: 60,
  temperature: 0.2,
  autoBacktest: true,
  scope: { prices: true, indicators: true, watchlist: true, positions: false },
};

export type LastTest = {
  at: string;
  ok: boolean;
  provider: QuantProvider;
  model: string;
  models: string[];
  temperatureApplied: boolean;
};

export type UsageMonth = { month: string; tokens: number; questions: number; strategies: number };

export type Turn =
  | { role: "user"; text: string }
  | { role: "assistant"; response: QuantChatResponse; backtest?: BacktestResult | null; backtestError?: string | null }
  | { role: "error"; text: string };

export type Conversation = { id: string; title: string; updatedAt: string; turns: Turn[] };

// Tiny localStorage-backed store usable with useSyncExternalStore. Every
// read/write is guarded: storage can be unavailable (private mode,
// blocked site data) and the app must still render.
function createStore<T>(key: string, fallback: T, normalize: (raw: unknown) => T = (v) => v as T) {
  const listeners = new Set<() => void>();
  let cachedRaw: string | null | undefined;
  let cachedValue: T = fallback;

  function read(): T {
    let raw: string | null = null;
    try {
      raw = window.localStorage.getItem(key);
    } catch {
      raw = null;
    }
    if (raw === cachedRaw) return cachedValue;
    cachedRaw = raw;
    try {
      cachedValue = raw ? normalize(JSON.parse(raw)) : fallback;
    } catch {
      cachedValue = fallback;
    }
    return cachedValue;
  }

  function write(value: T) {
    try {
      window.localStorage.setItem(key, JSON.stringify(value));
    } catch {
      // Storage full or blocked: keep the in-memory value for this tab.
      cachedRaw = undefined;
      cachedValue = value;
    }
    listeners.forEach((l) => l());
  }

  function subscribe(cb: () => void) {
    listeners.add(cb);
    const onStorage = (e: StorageEvent) => {
      if (e.key === key) cb();
    };
    window.addEventListener("storage", onStorage);
    return () => {
      listeners.delete(cb);
      window.removeEventListener("storage", onStorage);
    };
  }

  function useValue(): T {
    return useSyncExternalStore(subscribe, read, () => fallback);
  }

  return { read, write, useValue };
}

function normalizeSettings(raw: unknown): QuantSettings {
  const r = (raw ?? {}) as Partial<QuantSettings>;
  return {
    ...DEFAULT_SETTINGS,
    ...r,
    apiKeys: { ...DEFAULT_SETTINGS.apiKeys, ...(r.apiKeys ?? {}) },
    models: { ...DEFAULT_SETTINGS.models, ...(r.models ?? {}) },
    scope: { ...DEFAULT_SETTINGS.scope, ...(r.scope ?? {}) },
  };
}

export const settingsStore = createStore<QuantSettings>("vss.quant.settings.v1", DEFAULT_SETTINGS, normalizeSettings);
export const lastTestStore = createStore<LastTest | null>("vss.quant.lastTest.v1", null);
export const usageStore = createStore<UsageMonth | null>("vss.quant.usage.v1", null);
export const historyStore = createStore<Conversation[]>("vss.quant.history.v1", []);
export const activeConversationStore = createStore<string | null>("vss.quant.active.v1", null);

export function currentMonth(): string {
  return new Date().toISOString().slice(0, 7);
}

export function recordUsage(tokens: number, strategy: boolean) {
  const month = currentMonth();
  const prev = usageStore.read();
  const base = prev && prev.month === month ? prev : { month, tokens: 0, questions: 0, strategies: 0 };
  usageStore.write({
    month,
    tokens: base.tokens + tokens,
    questions: base.questions + 1,
    strategies: base.strategies + (strategy ? 1 : 0),
  });
}

// A connection is usable once it has what the backend requires: a key
// for Claude/OpenAI, a base URL for a self-hosted endpoint, and a model.
export function isConfigured(s: QuantSettings): boolean {
  const hasModel = s.models[s.provider].trim() !== "";
  if (s.provider === "custom") return s.baseUrl.trim() !== "" && hasModel;
  return s.apiKeys[s.provider].trim() !== "" && hasModel;
}

// The connection fields every /api/quant request carries.
export function connectionOf(s: QuantSettings) {
  return {
    provider: s.provider,
    apiKey: s.apiKeys[s.provider],
    baseUrl: s.provider === "custom" ? s.baseUrl : "",
    model: s.models[s.provider],
    timeoutSeconds: s.timeoutSeconds,
    temperature: s.temperature,
  };
}

export const MAX_CONVERSATIONS = 10;

export function saveConversation(conv: Conversation) {
  const rest = historyStore.read().filter((c) => c.id !== conv.id);
  historyStore.write([conv, ...rest].slice(0, MAX_CONVERSATIONS));
}
