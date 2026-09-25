# OpenAI-compatible bridge in front of Google Gemini -- the reference
# "Máy chủ riêng" (self-hosted) provider for Trợ lý Quant
# (backend/internal/quant/openai_client.go's `custom` provider speaks
# exactly the two endpoints below; see docs/roadmap/phases/
# phase-quant-gemini-bridge.md for the full wire-translation decisions).
# Deliberately generic: this file knows nothing about Quant's own JSON
# schema, only the OpenAI <-> Gemini wire shapes, so it stays a stand-in
# for any self-hosted OpenAI-compatible server.
import os
import time

import httpx
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

GEMINI_API_KEY = os.environ["GEMINI_API_KEY"]
GEMINI_BASE = "https://generativelanguage.googleapis.com/v1beta"

app = FastAPI()


def gemini_error_response(status: int, message: str) -> JSONResponse:
    return JSONResponse(status_code=status, content={"error": {"message": message}})


@app.get("/healthz")
async def healthz():
    return {"status": "ok"}


@app.get("/v1/models")
async def list_models():
    async with httpx.AsyncClient(timeout=15) as client:
        try:
            resp = await client.get(f"{GEMINI_BASE}/models", params={"key": GEMINI_API_KEY})
        except httpx.HTTPError as exc:
            return gemini_error_response(502, f"could not reach Gemini: {exc}")

    if resp.status_code != 200:
        return gemini_error_response(resp.status_code, _gemini_error_message(resp))

    body = resp.json()
    data = []
    for m in body.get("models", []):
        if "generateContent" not in m.get("supportedGenerationMethods", []):
            continue
        model_id = m.get("name", "").removeprefix("models/")
        if model_id:
            data.append({"id": model_id, "object": "model"})
    return {"object": "list", "data": data}


@app.post("/v1/chat/completions")
async def chat_completions(request: Request):
    body = await request.json()
    model = body.get("model")
    messages = body.get("messages", [])
    temperature = body.get("temperature")

    system_parts = [m["content"] for m in messages if m.get("role") == "system" and m.get("content")]
    contents = [
        {"role": "model" if m.get("role") == "assistant" else "user", "parts": [{"text": m.get("content", "")}]}
        for m in messages
        if m.get("role") in ("user", "assistant")
    ]

    gemini_req: dict = {"contents": contents}
    if system_parts:
        gemini_req["systemInstruction"] = {"parts": [{"text": "\n\n".join(system_parts)}]}
    if temperature is not None:
        gemini_req["generationConfig"] = {"temperature": temperature}

    async with httpx.AsyncClient(timeout=90) as client:
        try:
            resp = await client.post(
                f"{GEMINI_BASE}/models/{model}:generateContent",
                params={"key": GEMINI_API_KEY},
                json=gemini_req,
            )
        except httpx.HTTPError as exc:
            return gemini_error_response(502, f"could not reach Gemini: {exc}")

    if resp.status_code != 200:
        return gemini_error_response(resp.status_code, _gemini_error_message(resp))

    return _to_openai_completion(resp.json(), model)


def _gemini_error_message(resp: httpx.Response) -> str:
    try:
        return resp.json().get("error", {}).get("message", resp.text[:300])
    except ValueError:
        return resp.text[:300]


_FINISH_REASONS = {
    "STOP": "stop",
    "MAX_TOKENS": "length",
    "SAFETY": "content_filter",
    "RECITATION": "content_filter",
}


def _to_openai_completion(gemini_resp: dict, model: str) -> dict:
    usage_meta = gemini_resp.get("usageMetadata", {})
    usage = {
        "prompt_tokens": usage_meta.get("promptTokenCount", 0),
        "completion_tokens": usage_meta.get("candidatesTokenCount", 0),
        "total_tokens": usage_meta.get("totalTokenCount", 0),
    }
    candidates = gemini_resp.get("candidates", [])

    if not candidates:
        block_reason = gemini_resp.get("promptFeedback", {}).get("blockReason", "blocked")
        message = {"role": "assistant", "content": "", "refusal": f"Gemini blocked this request: {block_reason}"}
        finish_reason = "content_filter"
    else:
        candidate = candidates[0]
        text = "".join(p.get("text", "") for p in candidate.get("content", {}).get("parts", []))
        message = {"role": "assistant", "content": text, "refusal": None}
        finish_reason = _FINISH_REASONS.get(candidate.get("finishReason", "STOP"), "stop")

    return {
        "id": "chatcmpl-gemini-bridge",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": model,
        "choices": [{"index": 0, "message": message, "finish_reason": finish_reason}],
        "usage": usage,
    }
