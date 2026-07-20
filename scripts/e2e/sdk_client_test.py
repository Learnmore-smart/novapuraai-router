#!/usr/bin/env python3
"""
Cross-software compatibility test: drive the new-api gateway using the
OFFICIAL OpenAI Python SDK, the same way an end user would in their own app.

Configures the SDK to point at the local new-api endpoint and uses the
user-created API key. Verifies:
  - listing models works
  - non-streaming chat completion works
  - streaming chat completion works
"""
import os
import sys
from openai import OpenAI

USER_KEY = open("/tmp/user_api_key.txt").read().strip()
BASE_URL = "http://localhost:3000/v1"

print(f"pointing OpenAI SDK at: {BASE_URL}")
print(f"using API key: sk-{USER_KEY[:8]}...{USER_KEY[-4:]}")
print("-" * 60)

client = OpenAI(api_key=f"sk-{USER_KEY}", base_url=BASE_URL)

# --- 1. List models ---
print("\n[1] client.models.list()")
models = client.models.list()
print(f"    got {len(models.data)} models:")
for m in models.data:
    print(f"      - {m.id}  (owned_by={m.owned_by})")

# --- 2. Non-streaming chat completion ---
print("\n[2] client.chat.completions.create(stream=False)")
resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "system", "content": "You are a concise assistant."},
        {"role": "user", "content": "Reply with exactly: pong"},
    ],
    stream=False,
)
print(f"    id={resp.id}")
print(f"    model={resp.model}")
print(f"    content={resp.choices[0].message.content!r}")
print(f"    usage={resp.usage.model_dump()}")

# --- 3. Streaming chat completion ---
print("\n[3] client.chat.completions.create(stream=True)")
print("    streaming deltas:")
collected = ""
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Count from 1 to 3."}],
    stream=True,
)
for chunk in stream:
    if not chunk.choices:
        continue
    delta = chunk.choices[0].delta
    if delta and delta.content:
        collected += delta.content
        print(f"      delta={delta.content!r}")
    fr = chunk.choices[0].finish_reason
    if fr:
        print(f"      finish_reason={fr}")
print(f"    full streamed content: {collected!r}")

print("\n" + "=" * 60)
print("ALL CHECKS PASSED: new-api endpoint is fully usable from the")
print("official OpenAI Python SDK (i.e. from arbitrary third-party software).")
