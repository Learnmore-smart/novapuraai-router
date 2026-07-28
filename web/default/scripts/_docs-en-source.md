===== SECTION: api-chat =====

`POST /v1/chat/completions` is the primary OpenAI-compatible chat endpoint.

## Request

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are helpful."},
      {"role": "user", "content": "Summarize NovaPuraAI in one sentence."}
    ],
    "temperature": 0.5,
    "max_tokens": 256
  }'
```

## Important fields

| Field                                  | Notes                                                |
| -------------------------------------- | ---------------------------------------------------- |
| `model`                                | Required. Must be enabled for your account           |
| `messages`                             | OpenAI chat messages array                           |
| `stream`                               | `true` for SSE token streaming                       |
| `temperature` / `top_p`                | Sampling controls                                    |
| `max_tokens` / `max_completion_tokens` | Output bounds (provider-dependent)                   |
| `tools` / `tool_choice`                | Function calling when the upstream model supports it |

## Streaming

Set `"stream": true`. The response is `text/event-stream` with `data: {...}` chunks ending in `data: [DONE]`.

## Compatibility

Most tools that accept a custom OpenAI base URL work unchanged. Point them at `https://www.novapuraai.com/v1` and use your NovaPuraAI key.

===== SECTION: api-embeddings =====

`POST /v1/embeddings` creates vector embeddings for search, RAG, and clustering.

## Example

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## Notes

- `input` may be a string or an array of strings (subject to provider limits).
- Dimensions and normalization depend on the upstream embedding model.
- Billing is typically proportional to input tokens.

## Python

```python
from openai import OpenAI
client = OpenAI(api_key="sk-YOUR_KEY", base_url="https://www.novapuraai.com/v1")
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello world",
)
print(len(emb.data[0].embedding))
```

===== SECTION: api-errors =====

Errors are returned as JSON with an HTTP status code. Message text may be localized or provider-specific.

## Common status codes

| Code            | Meaning                                    |
| --------------- | ------------------------------------------ |
| 400             | Invalid request body or parameters         |
| 401             | Missing or invalid API key                 |
| 403             | Not allowed (model, module, or permission) |
| 404             | Unknown route or model                     |
| 429             | Rate limited                               |
| 500 / 502 / 503 | Gateway or upstream failure                |

## Example error body

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

Some endpoints use `{ "success": false, "message": "..." }` for console APIs. Relay routes prefer OpenAI-style error objects.

## Debugging checklist

1. Log the request id if the response or console logs expose one.
2. Retry idempotent GETs; be careful with POST retries.
3. Compare working curl from the dashboard “First API request” card.
4. Verify channel health with your administrator if only some models fail.

===== SECTION: api-gemini =====

Gemini-compatible traffic is available through Google-style paths under `/v1beta` when administrators enable Gemini channels.

## Generate content

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
```

## Tips

- Exact model IDs depend on your deployment’s channel configuration.
- Multimodal parts (inline data / file URIs) follow Gemini request shapes; keep payloads within gateway body limits.
- You may also access some Gemini models through OpenAI-compatible chat if a channel adapter maps them.

## Auth

Use the same NovaPuraAI `sk-` key. Do not send Google AI Studio keys to NovaPuraAI unless you are an admin configuring upstream channels.

===== SECTION: api-media =====

NovaPuraAI proxies selected media endpoints when corresponding channels are enabled.

## Images

`POST /v1/images/generations`

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A minimal logo for an API platform",
    "size": "1024x1024"
  }'
```

## Audio transcription

`POST /v1/audio/transcriptions` (multipart form)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## Speech synthesis

`POST /v1/audio/speech` returns audio bytes for supported TTS models.

## Rerank

`POST /v1/rerank` accepts query + documents for Cohere/Jina-style rerankers when configured.

## Billing note

Media endpoints often bill by image count, seconds, or document count — not only tokens. Check Model Square before bulk jobs.

===== SECTION: api-messages =====

`POST /v1/messages` accepts Anthropic Messages-style payloads for Claude-compatible models configured on the gateway.

## Example

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Explain prepaid API billing briefly."}
    ]
  }'
```

## Notes

- Model names must exist in your NovaPuraAI catalog.
- Some Anthropic-only headers are accepted and forwarded when relevant.
- You can often call the same Claude models via OpenAI chat format if the channel adapter supports it — prefer the format your SDK expects.

## Errors

Invalid schema or unsupported fields return `4xx` with a JSON error body. Check that `max_tokens` is present when required by the Messages API.

===== SECTION: api-models =====

`GET /v1/models` lists models available to the authenticated key.

## Example

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## Response shape

The payload follows OpenAI’s list object with `data[]` entries containing at least `id` and `object`. Additional metadata may appear depending on gateway version and settings.

## When a model is missing

1. Confirm the model is enabled in admin channels / abilities for your group.
2. Confirm your key is not restricted away from that model.
3. Refresh Model Square in the UI for pricing and availability.

## Caching

Clients may cache the list for a short TTL. Re-fetch after admin changes or on `404 model_not_found` errors.

===== SECTION: authentication =====

Every relay request must present a NovaPuraAI API key. Keys are managed in the console and validated by `TokenAuth` on the gateway.

## Header format

Send the key as a Bearer token:

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

Some OpenAI clients also accept `api_key` in the SDK constructor — that value becomes the same Authorization header.

## Where to create keys

1. Sign in → **API Keys**.
2. Create a key with an optional name.
3. Configure model allowlists, remaining quota, IP limits, and expiry if needed.
4. Save the secret immediately. The full secret is only shown once.

## Security best practices

- Prefer environment variables (`NOVAPURA_API_KEY`) over hard-coding.
- Use separate keys per environment (dev / staging / production).
- Rotate keys if a client is compromised.
- Restrict keys to the minimum set of models your app needs.
- Do not embed keys in public frontend bundles.

## Common failures

| Symptom            | Likely cause                                       |
| ------------------ | -------------------------------------------------- |
| `401 Unauthorized` | Missing/invalid key, revoked key, or wrong header  |
| `403 Forbidden`    | Model not allowed for this key, or module disabled |
| `429`              | Rate limit exceeded                                |
| Insufficient quota | Balance too low or key quota exhausted             |

## Multi-user setups

Administrators can issue keys to end users with independent quotas. Each key is billed against its owner’s balance according to platform settings.

===== SECTION: base-url =====

NovaPuraAI serves a unified gateway. Clients point at your public origin; the gateway routes to upstream providers.

## Recommended base URL

| Client type                                | Base URL                        |
| ------------------------------------------ | ------------------------------- |
| OpenAI SDK / OpenAI-compatible tools       | `https://www.novapuraai.com/v1` |
| Raw HTTP (path already includes `/v1/...`) | `https://www.novapuraai.com`    |

## Primary endpoints

| Method | Path                                     | Purpose              |
| ------ | ---------------------------------------- | -------------------- |
| POST   | `/v1/chat/completions`                   | Chat (OpenAI)        |
| POST   | `/v1/completions`                        | Text completions     |
| POST   | `/v1/responses`                          | OpenAI Responses API |
| POST   | `/v1/messages`                           | Anthropic Messages   |
| POST   | `/v1/embeddings`                         | Embeddings           |
| POST   | `/v1/images/generations`                 | Image generation     |
| POST   | `/v1/audio/transcriptions`               | Speech-to-text       |
| POST   | `/v1/audio/speech`                       | Text-to-speech       |
| POST   | `/v1/rerank`                             | Rerank               |
| GET    | `/v1/models`                             | List models          |
| POST   | `/v1beta/models/{model}:generateContent` | Gemini-style         |

Midjourney and other task routes may also be available depending on admin configuration.

## Authentication on every call

All of the paths above require:

```http
Authorization: Bearer sk-YOUR_KEY
```

## Health of the gateway

The admin console and public status endpoints report whether the site is ready. For production on Cloud Run, pair the container with Cloud SQL and Redis — do not rely on container-local SQLite.

===== SECTION: billing =====

Usage is metered per request. The gateway estimates cost, pre-consumes quota when required, then settles after the upstream response.

## Concepts

- **Balance / quota** — prepaid credit on the user account (and optionally per key).
- **Model pricing** — configured by administrators (token ratios, fixed prices, or expression-based rules).
- **Pre-consume** — holds estimated quota so concurrent requests cannot overspend.
- **Settle** — adjusts the final charge from actual token usage when available.

## What clients should know

1. A successful API call may still fail later if upstream returns an error after pre-consume (unused hold is refunded according to platform logic).
2. Streaming responses bill on completed usage when the provider reports tokens.
3. Image, audio, and video products may bill by count, duration, or resolution multipliers — always check Model Square.

## Wallet top-ups

End users top up from the **Wallet** page when payment gateways are enabled (for example Stripe). Administrators control currencies, promotions, and limits.

## Monitoring spend

- Console usage logs show model, tokens, and quota delta per request.
- Keep separate API keys per product so spend is easy to attribute.

## Insufficient quota

If a request is rejected for quota, top up the wallet or ask an admin to increase limits. Creating a new key does not create free balance.

===== SECTION: faq =====

Answers to common questions about using the NovaPuraAI API gateway. For product UI help beyond the API, use your deployment’s console support channels.

## What is NovaPuraAI?

NovaPuraAI is an OpenAI-compatible API gateway (a new-api-based product). You send requests to a single base URL with an API key; the gateway authenticates, routes by model, bills quota, and logs usage.

## Where are the docs?

The official developer documentation UI is at **`/docs`** on your deployment (for example `https://www.novapuraai.com/docs`).

## What base URL should I use?

- **Origin**: `https://www.novapuraai.com` (or your self-hosted origin)
- **OpenAI SDK `base_url`**: `https://www.novapuraai.com/v1`

See [Base URL & Endpoints](/docs/base-url).

## How do I get an API key?

Sign in → **Dashboard → API Keys / Tokens** → create a key → copy the `sk-...` secret. Details: [Authentication](/docs/authentication).

## Why do I get 401 Unauthorized?

Common causes: missing `Authorization` header, truncated key, disabled token, or using an OpenAI Platform key instead of a NovaPuraAI key.

## Why is my model not found?

Model catalogs are deployment- and group-specific. Call `GET /v1/models` and use an `id` from the response. Admin channel configuration may also need updates.

## Do you support Claude / Gemini native APIs?

Yes:

- Claude Messages: `POST /v1/messages`
- Gemini: `/v1beta/models/{model}:{action}`

OpenAI Chat Completions remains the most common path for multi-provider apps.

## How is billing calculated?

By model pricing rules configured on the gateway—usually tokens for chat/embeddings, and modality-specific units for images/audio. See [Billing & Quota](/docs/billing) and your console usage logs for authoritative amounts.

## What happens if I run out of quota?

Requests fail with an insufficient-quota style error. Top up or redeem credits in the console, then retry. Per-key limits can exhaust before the account balance does.

## Are rate limits the same as quota?

No. **429** means slow down; insufficient quota means add balance. See [Rate Limits](/docs/rate-limits).

## Can I use the official OpenAI SDKs?

Yes—set the API key to your NovaPuraAI key and `base_url` / `baseURL` to `{ORIGIN}/v1`. Examples: [Python](/docs/sdk-python), [Node.js](/docs/sdk-node), [Go](/docs/sdk-go), [curl](/docs/sdk-curl).

## Can I self-host?

NovaPuraAI can be deployed on your own infrastructure (for example Cloud Run). Replace example hosts in these docs with **your deployment origin**. Feature availability still depends on your admin configuration.

## Is streaming supported?

Yes for models/channels that support streaming. Use `"stream": true` on Chat Completions or the protocol-native streaming endpoints for Claude/Gemini.

## How do I secure keys in production?

Keep keys on servers or secret stores, rotate after leaks, apply IP and model allowlists when available, and avoid embedding secrets in mobile/web clients.

## Where can I see request history?

In the console **usage logs** (and dashboard charts when enabled). Include timestamps and error bodies when escalating to an administrator—never send raw keys.

## Still stuck?

1. Reproduce with curl from [Your First Request](/docs/first-request).
2. Check [Errors](/docs/api-errors).
3. Confirm models, quota, and rate limits in the console.
4. Contact your deployment administrator with redacted request details.

===== SECTION: first-request =====

This page walks through a complete first successful call and how to read the response.

## Checklist

- [ ] You have a key starting with `sk-`
- [ ] Your account has positive balance / quota
- [ ] You know at least one enabled model name (see **Model Square** or `GET /v1/models`)

## curl

```bash
export NOVAPURA_API_KEY=sk-YOUR_KEY
export NOVAPURA_BASE=https://www.novapuraai.com

curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are a concise assistant."},
      {"role": "user", "content": "Say hello in one sentence."}
    ],
    "temperature": 0.7
  }'
```

## Successful response (shape)

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": { "role": "assistant", "content": "Hello!" },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
```

## Streaming

Add `"stream": true` and read Server-Sent Events:

```bash
curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to five."}]
  }'
```

## Troubleshooting

1. Confirm the model name exactly matches an enabled model.
2. Confirm the base URL includes `/v1` for OpenAI SDKs.
3. Confirm HTTPS and that Cloud Run / CDN allows long-running streams if you stream.

===== SECTION: integration-cursor =====

Cursor can use NovaPuraAI as an OpenAI-compatible model provider. You point Cursor at your deployment origin’s `/v1` API and authenticate with a NovaPuraAI API key.

## What you need

- A NovaPuraAI API key from **Dashboard → API Keys / Tokens**
- Your deployment origin, for example `https://www.novapuraai.com`
- Sufficient quota for the models you select in Cursor

## Configure OpenAI-compatible access

Cursor’s UI labels change over releases. Look for **OpenAI API**, **OpenAI Compatible**, **Override OpenAI Base URL**, or **Custom model provider** settings.

Typical values:

| Setting  | Value                               |
| -------- | ----------------------------------- |
| API key  | `sk-xxxxxxxx` (your NovaPuraAI key) |
| Base URL | `https://www.novapuraai.com/v1`     |
| Model    | A model ID from `GET /v1/models`    |

If a field asks for the “OpenAI base URL” without `/v1`, try both forms and confirm with a tiny prompt. The working form is almost always **`{ORIGIN}/v1`**.

## Verify outside Cursor first

```bash
export NOVAPURA_BASE_URL="https://www.novapuraai.com"
export NOVAPURA_API_KEY="sk-xxxxxxxx"

curl "${NOVAPURA_BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${NOVAPURA_API_KEY}"
```

If this fails, fix key/quota before debugging the IDE.

## Model selection tips

- Prefer models that support tools / long context if you use agent features.
- If Cursor shows “model not found”, the ID is not enabled for your group—list models and pick an available `id`.
- Keep a cheaper model for autocomplete-style usage and a stronger model for agent mode when cost matters.

## Troubleshooting

| Issue                         | Fix                                                                  |
| ----------------------------- | -------------------------------------------------------------------- |
| Auth errors in Cursor         | Re-paste the full key including `sk-`                                |
| Network / CORS style failures | Cursor is a desktop client; usually not CORS—check URL typos and VPN |
| Empty responses               | Confirm quota and try the same model via curl                        |
| Rate limits                   | Reduce parallel agent runs; see [Rate Limits](/docs/rate-limits)     |

## Security

- Do not share workspace settings files that embed secrets.
- Rotate keys if a machine is lost or a key was committed.

## Related

- [Authentication](/docs/authentication)
- [Models List](/docs/api-models)
- [Chat Completions](/docs/api-chat)

===== SECTION: integration-dify =====

Dify can call NovaPuraAI as a custom OpenAI-compatible model provider. This lets apps, agents, and workflows in Dify use models routed through your gateway.

## Prerequisites

- A Dify workspace (self-hosted or cloud) with permission to add model providers
- NovaPuraAI API key and quota
- Origin such as `https://www.novapuraai.com`

## Add an OpenAI-API-compatible provider

In Dify **Settings → Model Providers** (or **Model Supplier**):

1. Choose **OpenAI-API-compatible** (name may vary slightly).
2. Set credentials:

| Field            | Value                           |
| ---------------- | ------------------------------- |
| API Key          | `sk-xxxxxxxx`                   |
| API endpoint URL | `https://www.novapuraai.com/v1` |

3. Add one or more models with the **exact model name** returned by NovaPuraAI (for example `gpt-4o-mini`).
4. Configure context length / mode (chat vs completion) to match the model type.
5. Save and run Dify’s connection test if available.

## Endpoint expectations

Dify will typically request:

- `POST /v1/chat/completions` for chat models
- `POST /v1/embeddings` when embedding models are configured
- `GET /v1/models` only if the provider integration performs discovery

Confirm with a direct curl call before debugging Dify graphs.

## Agent and workflow tips

- Create separate Dify model entries for cheap vs premium NovaPuraAI models.
- Set sensible max token limits in the node configuration to control cost.
- For tools/function calling, pick models whose channels support tools.

## Common failures

| Symptom                    | Likely cause                                  |
| -------------------------- | --------------------------------------------- |
| Validation failed          | Wrong endpoint (missing `/v1`) or key         |
| Model not found            | Name mismatch vs `GET /v1/models`             |
| Timeout in long chains     | Increase timeouts; reduce sequential LLM hops |
| Insufficient quota mid-run | Top up balance; cap retries in workflow       |

## Related

- [Billing & Quota](/docs/billing)
- [Embeddings](/docs/api-embeddings)
- [Chat Completions](/docs/api-chat)

===== SECTION: integration-langchain =====

LangChain and LlamaIndex can call NovaPuraAI through their OpenAI integrations by overriding the base URL and API key. The gateway then routes model names to configured channels.

## Shared configuration

```bash
export NOVAPURA_API_KEY="sk-xxxxxxxx"
export NOVAPURA_BASE_URL="https://www.novapuraai.com"   # origin only
```

SDK clients generally need `base_url` / `base_url` **with** `/v1`.

## LangChain (Python)

```bash
pip install langchain-openai
```

```python
import os
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
    temperature=0.2,
)

print(llm.invoke("Hello from NovaPuraAI").content)
```

### Embeddings with LangChain

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)
vector = emb.embed_query("gateway documentation")
```

## LangChain.js

```bash
npm install @langchain/openai
```

```typescript
import { ChatOpenAI } from '@langchain/openai'

const llm = new ChatOpenAI({
  model: 'gpt-4o-mini',
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: `${process.env.NOVAPURA_BASE_URL}/v1`,
  },
})

const res = await llm.invoke('Hello from NovaPuraAI')
console.log(res.content)
```

## LlamaIndex (Python)

```bash
pip install llama-index-llms-openai llama-index-embeddings-openai
```

```python
import os
from llama_index.llms.openai import OpenAI
from llama_index.embeddings.openai import OpenAIEmbedding

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

embed = OpenAIEmbedding(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

print(llm.complete("Hello from NovaPuraAI"))
```

Parameter names (`api_base` vs `base_url`) differ slightly across LlamaIndex versions—prefer the keyword accepted by your installed package.

## RAG checklist

1. Use the **same** embedding model for index and query time.
2. Store the model ID alongside the vector index metadata.
3. Cap concurrency to respect [Rate Limits](/docs/rate-limits).
4. Monitor NovaPuraAI usage logs while evaluating retrieval quality so cost stays predictable.

## Related

- [Python SDK](/docs/sdk-python)
- [Embeddings](/docs/api-embeddings)
- [Billing & Quota](/docs/billing)

===== SECTION: integration-nextchat =====

NextChat (ChatGPT-Next-Web and compatible forks) can talk to NovaPuraAI through OpenAI-compatible settings. You configure the base URL, API key, and default model once, then use the web UI as usual.

## Prerequisites

- Running NextChat (self-hosted or local)
- NovaPuraAI API key
- Deployment origin such as `https://www.novapuraai.com`

## Settings to apply

In NextChat **Settings** (wording may vary by fork/version):

| Field               | Recommended value               |
| ------------------- | ------------------------------- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key             | `sk-xxxxxxxx`                   |
| Model               | An ID from `GET /v1/models`     |

If the UI stores only the origin and always appends `/v1` itself, use `https://www.novapuraai.com` without duplicating `/v1`. When in doubt, open browser network tools and confirm the final path is `/v1/chat/completions`.

## Environment-variable style deployments

Many NextChat Docker images accept:

```bash
OPENAI_API_KEY=sk-xxxxxxxx
BASE_URL=https://www.novapuraai.com/v1
# some images use OPENAI_API_BASE / OPENAI_BASE_URL — check your image docs
```

Restart the container after changing env vars.

## Smoke test

```bash
curl "https://www.novapuraai.com/v1/chat/completions" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

## Common issues

| Symptom          | Cause                                     | Fix                                           |
| ---------------- | ----------------------------------------- | --------------------------------------------- |
| 404 on chat      | Wrong base URL (`/v1` missing or doubled) | Align with network tab path                   |
| 401              | Key not passed or wrong                   | Paste NovaPuraAI key, not OpenAI platform key |
| Model list empty | Frontend cannot call `/v1/models`         | Check CORS/proxy and key permissions          |
| Balance errors   | No quota                                  | Top up in NovaPuraAI console                  |

## Security notes

- Prefer server-side proxy mode if your NextChat build supports hiding keys from the browser.
- For public demos, use low-quota keys and strict model allowlists.

## Related

- [Base URL & Endpoints](/docs/base-url)
- [Authentication](/docs/authentication)
- [FAQ](/docs/faq)

===== SECTION: integration-openwebui =====

Open WebUI can use NovaPuraAI as an OpenAI-compatible backend. Configure a connection with your API key and base URL, then select models from the list returned by NovaPuraAI.

## Prerequisites

- Open WebUI installed (Docker is common)
- NovaPuraAI API key with quota
- Origin such as `https://www.novapuraai.com`

## Admin connection settings

In Open WebUI admin settings, open the **Connections** / **OpenAI** section (labels vary by version) and add:

| Field        | Value                           |
| ------------ | ------------------------------- |
| API Base URL | `https://www.novapuraai.com/v1` |
| API Key      | `sk-xxxxxxxx`                   |

Save, then refresh models. Open WebUI will call `GET /v1/models` and `POST /v1/chat/completions` against your gateway.

## Docker environment example

Some deployments inject provider settings via env:

```bash
OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
OPENAI_API_KEY=sk-xxxxxxxx
```

Exact variable names depend on your Open WebUI version—confirm in its documentation if the UI is unavailable.

## Selecting models

- Only models enabled for your key appear.
- If a model is missing, verify with curl against `/v1/models`.
- For multimodal chat, choose models your channels support for vision; capability is channel-dependent.

## Streaming and tools

- Streaming chat works through the OpenAI-compatible path when the model/channel supports it.
- Tool calling / function features require both Open WebUI feature flags and a model that supports tools.

## Troubleshooting

| Issue                      | What to check                                                   |
| -------------------------- | --------------------------------------------------------------- |
| “Incorrect API key”        | Key prefix, whitespace, disabled token                          |
| Empty model dropdown       | Base URL must include `/v1`; network must reach the gateway     |
| 429 during multi-user load | Rate limits per key/group; create separate keys or raise limits |
| Slow first token           | Upstream latency; try another model                             |

## Related

- [Models & Routing](/docs/routing)
- [Rate Limits](/docs/rate-limits)
- [Chat Completions](/docs/api-chat)

===== SECTION: quickstart =====

NovaPuraAI exposes an OpenAI-compatible HTTP API. With a valid API key and available quota, you can call models through one base URL.

## What you need

1. A NovaPuraAI account on your deployment (for example `https://www.novapuraai.com`).
2. An API key (`sk-...`) from **Console → API Keys**.
3. Balance or quota for the models you want to use.

## Base URL

For OpenAI-compatible SDKs, set `base_url` / `baseURL` to your site origin plus `/v1`:

```text
https://www.novapuraai.com/v1
```

Self-hosting? Replace the host with your Cloud Run or reverse-proxy domain.

## Create a key

1. Sign in to the console.
2. Open **API Keys** (tokens).
3. Create a key. Optionally restrict models, set a quota, and set an expiry.
4. Copy the secret once and store it as an environment variable. Never commit it.

## First chat request

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
```

## Official OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(resp.choices[0].message.content)
```

```javascript
import OpenAI from 'openai'

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: 'https://www.novapuraai.com/v1',
})

const resp = await client.chat.completions.create({
  model: 'gpt-4o-mini',
  messages: [{ role: 'user', content: 'Hello' }],
})
console.log(resp.choices[0].message.content)
```

## Next steps

- [Authentication](/docs/authentication)
- [Your First Request](/docs/first-request)
- [Base URL & Endpoints](/docs/base-url)

===== SECTION: rate-limits =====

Rate limits protect the platform and upstream providers. Limits may apply at IP, user, or token level depending on admin settings.

## Typical symptoms

- HTTP `429 Too Many Requests`
- Error messages mentioning rate limit or frequency

## Client guidance

1. Exponential backoff with jitter on `429` and transient `5xx`.
2. Reuse HTTP connections; avoid opening a new TLS session per tiny request when possible.
3. Batch work when the API supports it (for example embeddings arrays).
4. Cache model lists and static configuration.

## Streaming

Long-lived streams hold a connection. Design concurrency limits so you do not open more parallel streams than your plan allows.

## Admin-side knobs

Administrators can tune global and model-specific rate limits in system settings. Contact your site operator if legitimate traffic is throttled too aggressively.

===== SECTION: routing =====

When a client requests a model name, NovaPuraAI selects an upstream channel that can serve that model, subject to group permissions, channel health, and admin routing rules.

## Model names

- Use the model identifiers shown in **Model Square** or `GET /v1/models`.
- Names are case-sensitive and must match admin configuration.
- The same logical model may be served by multiple upstream keys for failover.

## How routing works (conceptual)

1. Authenticate the API key and resolve the user group.
2. Find channels that expose the requested model for that group.
3. Prefer healthy channels; skip disabled or failing channels.
4. Forward the request, translate provider formats when needed, and bill usage.

## Failover & reliability

Administrators can configure retries, cooldowns, and auto-disable rules. From the client perspective you still call one stable base URL — NovaPuraAI absorbs provider churn when multiple channels are available.

## Groups

Users may belong to groups with different model catalogs and ratios. If a model works for one account but not another, compare group abilities and key restrictions.

## Best practices

- Pin model names in config, not hard-coded one-off strings across many services.
- Prefer listing models via API at startup if your product needs dynamic discovery.
- Handle `5xx` and timeouts with client-side retries for idempotent reads; avoid blind retries for non-idempotent side effects.

===== SECTION: sdk-curl =====

curl is ideal for debugging and CI smoke tests.

## Chat

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

## List models

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## Pretty-print JSON

Pipe through `jq` when available:

```bash
curl -s ... | jq .
```

## Verbose debugging

Add `-v` to inspect TLS and headers. Redact Authorization when sharing logs.

===== SECTION: sdk-go =====

Go clients can call the HTTP API directly or use an OpenAI-compatible Go SDK.

## Direct HTTP

```go
package main

import (
  "bytes"
  "fmt"
  "net/http"
  "os"
)

func main() {
  body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}]}`)
  req, _ := http.NewRequest("POST", "https://www.novapuraai.com/v1/chat/completions", bytes.NewReader(body))
  req.Header.Set("Authorization", "Bearer "+os.Getenv("NOVAPURA_API_KEY"))
  req.Header.Set("Content-Type", "application/json")
  resp, err := http.DefaultClient.Do(req)
  if err != nil { panic(err) }
  defer resp.Body.Close()
  fmt.Println(resp.Status)
}
```

## Tips

- Set reasonable timeouts for non-stream and longer timeouts for streaming or media.
- Propagate context cancellation to abort in-flight requests when handlers return.

===== SECTION: sdk-node =====

Use the official `openai` npm package.

## Install

```bash
npm install openai
# or: bun add openai
```

## Client

```javascript
import OpenAI from 'openai'

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: 'https://www.novapuraai.com/v1',
})
```

## Chat

```javascript
const completion = await client.chat.completions.create({
  model: 'gpt-4o-mini',
  messages: [{ role: 'user', content: 'Hello' }],
})
console.log(completion.choices[0].message.content)
```

## Streaming

```javascript
const stream = await client.chat.completions.create({
  model: 'gpt-4o-mini',
  messages: [{ role: 'user', content: 'Stream digits 1-5' }],
  stream: true,
})
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '')
}
```

===== SECTION: sdk-python =====

Use the official `openai` Python package with a custom base URL.

## Install

```bash
pip install openai
```

## Client

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Chat

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(completion.choices[0].message.content)
```

## Streaming

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Stream a short poem."}],
    stream=True,
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="NovaPuraAI gateway",
)
vector = emb.data[0].embedding
```
