package insight

// Signal is one rule-based observation feeding into the summary -- kept
// structured (not just prose) so the frontend can render it as a labeled
// list, and so a future real model's output could populate the same
// shape without a frontend change.
type Signal struct {
	Label     string `json:"label"`
	Detail    string `json:"detail"`
	Direction string `json:"direction"` // "bullish" | "bearish" | "neutral"
}

// Insight is deliberately not called "AIInsight" in the API: Source makes
// explicit that this is rule-based today, not an LLM call (see RESUME.md
// -- that needs an API key and real per-call cost, a decision left to the
// user, not assumed here). A future real integration would keep this same
// shape and just set Source to "llm".
type Insight struct {
	Symbol      string   `json:"symbol"`
	Summary     string   `json:"summary"`
	Signals     []Signal `json:"signals"`
	Source      string   `json:"source"`
	GeneratedAt string   `json:"generatedAt"`
}
