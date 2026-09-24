// Shapes returned by the Go backend's /api/v1/quant endpoints
// (backend/internal/quant/types.go).

export type QuantProvider = "claude" | "openai" | "custom";

export type QuantCondition = { field: string; op: string; value: number; period: number };

export type QuantStrategy = {
  kind: "none" | "ema_crossover" | "rsi_reversion";
  name: string;
  symbol: string;
  fast: number;
  slow: number;
  rsiEntry: number;
  rsiExit: number;
  stopLossPercent: number;
  trendSma: number;
};

export type QuantPlan = {
  reply: string;
  screen: { exchange: string; conditions: QuantCondition[] };
  strategy: QuantStrategy;
};

export type QuantRow = {
  symbol: string;
  exchange: string;
  sector: string;
  price: number;
  changePercent: number;
  rsi14: number;
  roe: number;
  pe: number;
  pb: number;
  eps: number;
  dividendYield: number;
  marketCap: number;
  avgVolume20: number;
};

export type QuantScreen = {
  exchange: string;
  conditions: QuantCondition[];
  labels: string[];
  universe: number;
  matches: QuantRow[];
  sortedBy: "rsi14" | "symbol";
};

export type QuantUsage = { inputTokens: number; outputTokens: number };

export type QuantChatResponse = {
  reply: string;
  screen: QuantScreen | null;
  strategy: QuantStrategy | null;
  dropped: string[];
  plan: QuantPlan;
  usage: QuantUsage;
  latencyMs: number;
  model: string;
  temperatureApplied: boolean;
};

export type QuantTestResponse = {
  ok: boolean;
  latencyMs: number;
  usage: QuantUsage;
  model: string;
  sampleQuestion: string;
  understood: number;
  expected: number;
  labels: string[];
  temperatureApplied: boolean;
  models: string[];
  modelsError?: string;
};

export type QuantError = { code: string; message: string; providerStatus?: number };

export type BacktestResult = {
  id: string;
  symbol: string;
  ruleType: string;
  returnPercent?: number;
  winRate?: number;
  totalTrades?: number;
  maxDrawdownPercent: number;
  profitFactor: number;
  finalCapital?: number;
};
