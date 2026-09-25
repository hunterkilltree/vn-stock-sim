# Quant "Máy chủ riêng" reference server — a Python bridge to Gemini

phase-h.md built Trợ lý Quant against three provider slots: Claude,
OpenAI, and `custom` ("Máy chủ riêng") — any server that speaks the
OpenAI Chat Completions wire format at a user-supplied base URL. Nobody
had actually run something behind that third slot; it was verified only
against a throwaway local stub in phase-h.md's browser run. This phase
gives it a real, always-available example: a small Python service that
speaks that same OpenAI-compatible surface and forwards to Google's
Gemini API, so "Máy chủ riêng" in the docker-compose demo is a genuine,
working self-hosted model instead of an unexercised option.

## Why this doesn't touch Go or Next.js code at all

`internal/quant/openai_client.go`'s `openAIClient` already implements the
generic `ModelClient` port for `custom` exactly as it does for `openai` —
only the base URL differs (`validateBaseURL` + the SSRF guard in
`netguard.go` instead of a fixed OpenAI URL). The service just needs
something reachable at that URL implementing `GET /models` and
`POST /chat/completions` in the OpenAI shape. That's the same
dependency-inversion shape the VCI market-data adapter used
(phase-vci-market-data.md): the port was already correct, so a new
implementation slots in behind it with zero changes to the code that
calls it. This bridge is deliberately generic (it doesn't know anything
about Quant's `Plan` JSON schema) so it can be swapped later for any
other self-hosted OpenAI-compatible server (Ollama, vLLM, ...) by
changing only the base URL, not this decision.

## Decisions

1. **Language/shape**: Python + FastAPI + uvicorn, one `main.py`, in
   `services/quant-gemini-bridge/` (a new top-level `services/` dir,
   parallel to `backend/`/`frontend/` — this is real runtime code, not
   the `tools/excalidraw-diagram/` skill).
2. **Wire translation**:
   - `GET /v1/models` → Gemini's `GET /v1beta/models`, filtered to models
     whose `supportedGenerationMethods` includes `generateContent`, IDs
     stripped of the `models/` prefix, returned as OpenAI's
     `{"data": [{"id": "..."}]}` — matches what `ListModels` in
     `openai_client.go` parses, and what "Kiểm tra" lists in the Settings
     UI (phase-h.md decision 5).
   - `POST /v1/chat/completions` → the request's `system` message becomes
     Gemini's `systemInstruction`; `user`/`assistant` messages become
     `contents` with role `user`/`model`; `temperature` (when present)
     becomes `generationConfig.temperature`. The response's first
     candidate's text becomes `choices[0].message.content`; a missing
     candidate (safety block) becomes `choices[0].message.refusal` so the
     Go client's existing `ErrRefused` path fires; `usageMetadata` maps to
     `usage.prompt_tokens`/`completion_tokens`.
   - No `response_format`/schema enforcement — the `custom` provider never
     gets one in `openai_client.go` (phase-h.md decision 1: "support
     varies by server"), so this bridge is prompted exactly like the
     Go service already prompts every self-hosted server: plain text, and
     `parsePlan` in `service.go` extracts the outermost `{...}` itself.
   - Non-2xx from Gemini is forwarded as the same HTTP status with an
     OpenAI-style `{"error": {"message": "..."}}` body, so
     `oaErrorMessage`/`ProviderError` on the Go side show Gemini's real
     error text, the same way an invalid Claude key already surfaces
     Anthropic's own message (phase-h.md's verification notes).
3. **The Gemini key lives only in the bridge**, as the `GEMINI_API_KEY`
   environment variable, read once at startup and never logged. It is
   not the same thing as the per-request key phase-h.md's ground rules
   describe (that rule is about a key the *user* types into the browser
   for a commercial provider and that must pass through unstored) — this
   is the operator's own server-side credential for a self-hosted
   backend, the same relationship as someone running their own Ollama
   with its own config. The Settings UI's "Khoá API" field is left blank
   for this provider; the bridge does not require a caller-supplied key.
4. **Reachability inside the demo stack**: a new `gemini-bridge` service
   in `docker-compose.yml`, built from `services/quant-gemini-bridge/Dockerfile`,
   on the compose network as `http://gemini-bridge:8090/v1` — that is
   the literal string to type into Settings → Mô hình AI → Máy chủ riêng
   → "Địa chỉ endpoint" (the Go backend, not the browser, calls this
   URL). Its address is a compose-network private IP, so
   `QUANT_ALLOW_PRIVATE_ENDPOINTS=true` is added to the `backend`
   service's environment in `docker-compose.yml`, same caveat RUNNING.md
   already documents for a local Ollama: fine for this demo, not for a
   shared/public deployment. Also exposed on the host as `8090:8090` so
   it can be curled directly while testing.
5. **Secret hygiene**: the real key is never committed. It goes in a new
   root `.env` (now in `.gitignore`, alongside a new `.env.example` that
   documents the variable name with no value), which Docker Compose reads
   automatically for `${GEMINI_API_KEY}` substitution.

## Not doing

- No structured-output request to Gemini (`responseSchema`) — see
  decision 2; keeping the bridge schema-agnostic is what keeps it a
  generic, swappable "Máy chủ riêng" example rather than a Quant-specific
  special case.
- No changes to `internal/quant/*` or the frontend: this phase is purely
  additive infrastructure behind an already-correct port.

## Verification (done)

- `docker compose build gemini-bridge` clean; container starts and
  `GET :8090/v1/models` returned 40+ real, live Gemini model IDs (not a
  fixture) — including that the first model tried, `gemini-2.5-flash`,
  came back with Gemini's own real deprecation error ("no longer
  available to new users... use models/gemini-3.8-flash"), forwarded
  through this bridge with the same message and status, confirming the
  key is live and error passthrough works, not just the happy path.
- Direct curl of `POST :8090/v1/chat/completions` with `gemini-flash-latest`
  and a trivial prompt returned real Gemini-generated text
  (`"content":"pong"`) in the exact OpenAI shape.
- Through the whole stack, authenticated with a real registered user:
  `POST /api/v1/quant/test` with `provider: "custom"`,
  `baseUrl: "http://gemini-bridge:8090/v1"`, `model: "gemini-flash-latest"`
  → `ok: true`, `understood: 3/3` on the sample question, real token
  usage and the full live model list. `POST /api/v1/quant/chat` with
  "Lọc HOSE có RSI dưới 35" → a real Vietnamese reply, correct
  `screen.conditions`, and 12 real matches computed by the app's own
  market/symbol services (not invented by the model) -- e.g. FPT at
  RSI 17.3/ROE 27.9%, matching phase-h.md's screen-executor design.
- Real browser (Docker build, logged-in user): Settings → Mô hình AI →
  Máy chủ riêng → `http://gemini-bridge:8090/v1` + `gemini-flash-latest`
  → "Kiểm tra" → "Hiểu đúng 3/3 điều kiện" and the top chip flipped to
  "Đã kết nối" → Lưu → header chip showed "Máy chủ riêng ·
  gemini-flash-latest" → Trợ lý Quant: "Loc HOSE co RSI duoi 35" → a real
  reply, condition chips, a 12-row results table sorted by RSI, and the
  footer "gemini-flash-latest · 4,8 giây · 3.556 token". No stub, no
  mocked provider anywhere in this path.
