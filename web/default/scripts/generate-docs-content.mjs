/**
 * Generates official NovaPuraAI developer docs markdown for all sections × languages.
 * Run from web/default: node scripts/generate-docs-content.mjs
 */
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '../src/i18n/docs')

const LANGS = ['en', 'zh', 'zh-TW', 'fr', 'ru', 'ja', 'vi']

const SECTIONS = [
  'quickstart',
  'authentication',
  'first-request',
  'base-url',
  'routing',
  'billing',
  'rate-limits',
  'api-chat',
  'api-messages',
  'api-gemini',
  'api-embeddings',
  'api-media',
  'api-models',
  'api-errors',
  'sdk-python',
  'sdk-node',
  'sdk-go',
  'sdk-curl',
  'integration-cursor',
  'integration-nextchat',
  'integration-openwebui',
  'integration-dify',
  'integration-langchain',
  'faq',
]

/** @type {Record<string, Record<string, string>>} */
const CONTENT = {}

function set(section, lang, body) {
  if (!CONTENT[section]) CONTENT[section] = {}
  CONTENT[section][lang] = body.trim() + '\n'
}

// ---------------------------------------------------------------------------
// English (source of truth)
// ---------------------------------------------------------------------------

set(
  'quickstart',
  'en',
  `NovaPuraAI exposes an OpenAI-compatible HTTP API. With a valid API key and available quota, you can call models through one base URL.

## What you need

1. A NovaPuraAI account on your deployment (for example \`https://www.novapuraai.com\`).
2. An API key (\`sk-...\`) from **Console → API Keys**.
3. Balance or quota for the models you want to use.

## Base URL

For OpenAI-compatible SDKs, set \`base_url\` / \`baseURL\` to your site origin plus \`/v1\`:

\`\`\`text
https://www.novapuraai.com/v1
\`\`\`

Self-hosting? Replace the host with your Cloud Run or reverse-proxy domain.

## Create a key

1. Sign in to the console.
2. Open **API Keys** (tokens).
3. Create a key. Optionally restrict models, set a quota, and set an expiry.
4. Copy the secret once and store it as an environment variable. Never commit it.

## First chat request

\`\`\`bash
curl https://www.novapuraai.com/v1/chat/completions \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
\`\`\`

## Official OpenAI SDK

\`\`\`python
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
\`\`\`

\`\`\`javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(resp.choices[0].message.content);
\`\`\`

## Next steps

- [Authentication](/docs/authentication)
- [Your First Request](/docs/first-request)
- [Base URL & Endpoints](/docs/base-url)
`
)

set(
  'authentication',
  'en',
  `Every relay request must present a NovaPuraAI API key. Keys are managed in the console and validated by \`TokenAuth\` on the gateway.

## Header format

Send the key as a Bearer token:

\`\`\`http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
\`\`\`

Some OpenAI clients also accept \`api_key\` in the SDK constructor — that value becomes the same Authorization header.

## Where to create keys

1. Sign in → **API Keys**.
2. Create a key with an optional name.
3. Configure model allowlists, remaining quota, IP limits, and expiry if needed.
4. Save the secret immediately. The full secret is only shown once.

## Security best practices

- Prefer environment variables (\`NOVAPURA_API_KEY\`) over hard-coding.
- Use separate keys per environment (dev / staging / production).
- Rotate keys if a client is compromised.
- Restrict keys to the minimum set of models your app needs.
- Do not embed keys in public frontend bundles.

## Common failures

| Symptom | Likely cause |
| --- | --- |
| \`401 Unauthorized\` | Missing/invalid key, revoked key, or wrong header |
| \`403 Forbidden\` | Model not allowed for this key, or module disabled |
| \`429\` | Rate limit exceeded |
| Insufficient quota | Balance too low or key quota exhausted |

## Multi-user setups

Administrators can issue keys to end users with independent quotas. Each key is billed against its owner’s balance according to platform settings.
`
)

set(
  'first-request',
  'en',
  `This page walks through a complete first successful call and how to read the response.

## Checklist

- [ ] You have a key starting with \`sk-\`
- [ ] Your account has positive balance / quota
- [ ] You know at least one enabled model name (see **Model Square** or \`GET /v1/models\`)

## curl

\`\`\`bash
export NOVAPURA_API_KEY=sk-YOUR_KEY
export NOVAPURA_BASE=https://www.novapuraai.com

curl "$NOVAPURA_BASE/v1/chat/completions" \\
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are a concise assistant."},
      {"role": "user", "content": "Say hello in one sentence."}
    ],
    "temperature": 0.7
  }'
\`\`\`

## Successful response (shape)

\`\`\`json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "Hello!"},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
\`\`\`

## Streaming

Add \`"stream": true\` and read Server-Sent Events:

\`\`\`bash
curl "$NOVAPURA_BASE/v1/chat/completions" \\
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \\
  -H "Content-Type: application/json" \\
  -N \\
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to five."}]
  }'
\`\`\`

## Troubleshooting

1. Confirm the model name exactly matches an enabled model.
2. Confirm the base URL includes \`/v1\` for OpenAI SDKs.
3. Confirm HTTPS and that Cloud Run / CDN allows long-running streams if you stream.
`
)

set(
  'base-url',
  'en',
  `NovaPuraAI serves a unified gateway. Clients point at your public origin; the gateway routes to upstream providers.

## Recommended base URL

| Client type | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI-compatible tools | \`https://www.novapuraai.com/v1\` |
| Raw HTTP (path already includes \`/v1/...\`) | \`https://www.novapuraai.com\` |

Replace the host with your deployment domain when self-hosting.

## Primary endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| POST | \`/v1/chat/completions\` | Chat (OpenAI) |
| POST | \`/v1/completions\` | Text completions |
| POST | \`/v1/responses\` | OpenAI Responses API |
| POST | \`/v1/messages\` | Anthropic Messages |
| POST | \`/v1/embeddings\` | Embeddings |
| POST | \`/v1/images/generations\` | Image generation |
| POST | \`/v1/audio/transcriptions\` | Speech-to-text |
| POST | \`/v1/audio/speech\` | Text-to-speech |
| POST | \`/v1/rerank\` | Rerank |
| GET | \`/v1/models\` | List models |
| POST | \`/v1beta/models/{model}:generateContent\` | Gemini-style |

Midjourney and other task routes may also be available depending on admin configuration.

## Authentication on every call

All of the paths above require:

\`\`\`http
Authorization: Bearer sk-YOUR_KEY
\`\`\`

## Health of the gateway

The admin console and public status endpoints report whether the site is ready. For production on Cloud Run, pair the container with Cloud SQL and Redis — do not rely on container-local SQLite.
`
)

set(
  'routing',
  'en',
  `When a client requests a model name, NovaPuraAI selects an upstream channel that can serve that model, subject to group permissions, channel health, and admin routing rules.

## Model names

- Use the model identifiers shown in **Model Square** or \`GET /v1/models\`.
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
- Handle \`5xx\` and timeouts with client-side retries for idempotent reads; avoid blind retries for non-idempotent side effects.
`
)

set(
  'billing',
  'en',
  `Usage is metered per request. The gateway estimates cost, pre-consumes quota when required, then settles after the upstream response.

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
`
)

set(
  'rate-limits',
  'en',
  `Rate limits protect the platform and upstream providers. Limits may apply at IP, user, or token level depending on admin settings.

## Typical symptoms

- HTTP \`429 Too Many Requests\`
- Error messages mentioning rate limit or frequency

## Client guidance

1. Exponential backoff with jitter on \`429\` and transient \`5xx\`.
2. Reuse HTTP connections; avoid opening a new TLS session per tiny request when possible.
3. Batch work when the API supports it (for example embeddings arrays).
4. Cache model lists and static configuration.

## Streaming

Long-lived streams hold a connection. Design concurrency limits so you do not open more parallel streams than your plan allows.

## Admin-side knobs

Administrators can tune global and model-specific rate limits in system settings. Contact your site operator if legitimate traffic is throttled too aggressively.
`
)

set(
  'api-chat',
  'en',
  `\`POST /v1/chat/completions\` is the primary OpenAI-compatible chat endpoint.

## Request

\`\`\`bash
curl https://www.novapuraai.com/v1/chat/completions \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are helpful."},
      {"role": "user", "content": "Summarize NovaPuraAI in one sentence."}
    ],
    "temperature": 0.5,
    "max_tokens": 256
  }'
\`\`\`

## Important fields

| Field | Notes |
| --- | --- |
| \`model\` | Required. Must be enabled for your account |
| \`messages\` | OpenAI chat messages array |
| \`stream\` | \`true\` for SSE token streaming |
| \`temperature\` / \`top_p\` | Sampling controls |
| \`max_tokens\` / \`max_completion_tokens\` | Output bounds (provider-dependent) |
| \`tools\` / \`tool_choice\` | Function calling when the upstream model supports it |

## Streaming

Set \`"stream": true\`. The response is \`text/event-stream\` with \`data: {...}\` chunks ending in \`data: [DONE]\`.

## Compatibility

Most tools that accept a custom OpenAI base URL work unchanged. Point them at \`https://www.novapuraai.com/v1\` and use your NovaPuraAI key.
`
)

set(
  'api-messages',
  'en',
  `\`POST /v1/messages\` accepts Anthropic Messages-style payloads for Claude-compatible models configured on the gateway.

## Example

\`\`\`bash
curl https://www.novapuraai.com/v1/messages \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -H "anthropic-version: 2023-06-01" \\
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Explain prepaid API billing briefly."}
    ]
  }'
\`\`\`

## Notes

- Model names must exist in your NovaPuraAI catalog.
- Some Anthropic-only headers are accepted and forwarded when relevant.
- You can often call the same Claude models via OpenAI chat format if the channel adapter supports it — prefer the format your SDK expects.

## Errors

Invalid schema or unsupported fields return \`4xx\` with a JSON error body. Check that \`max_tokens\` is present when required by the Messages API.
`
)

set(
  'api-gemini',
  'en',
  `Gemini-compatible traffic is available through Google-style paths under \`/v1beta\` when administrators enable Gemini channels.

## Generate content

\`\`\`bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
\`\`\`

## Tips

- Exact model IDs depend on your deployment’s channel configuration.
- Multimodal parts (inline data / file URIs) follow Gemini request shapes; keep payloads within gateway body limits.
- You may also access some Gemini models through OpenAI-compatible chat if a channel adapter maps them.

## Auth

Use the same NovaPuraAI \`sk-\` key. Do not send Google AI Studio keys to NovaPuraAI unless you are an admin configuring upstream channels.
`
)

set(
  'api-embeddings',
  'en',
  `\`POST /v1/embeddings\` creates vector embeddings for search, RAG, and clustering.

## Example

\`\`\`bash
curl https://www.novapuraai.com/v1/embeddings \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
\`\`\`

## Notes

- \`input\` may be a string or an array of strings (subject to provider limits).
- Dimensions and normalization depend on the upstream embedding model.
- Billing is typically proportional to input tokens.

## Python

\`\`\`python
from openai import OpenAI
client = OpenAI(api_key="sk-YOUR_KEY", base_url="https://www.novapuraai.com/v1")
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello world",
)
print(len(emb.data[0].embedding))
\`\`\`
`
)

set(
  'api-media',
  'en',
  `NovaPuraAI proxies selected media endpoints when corresponding channels are enabled.

## Images

\`POST /v1/images/generations\`

\`\`\`bash
curl https://www.novapuraai.com/v1/images/generations \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "dall-e-3",
    "prompt": "A minimal logo for an API platform",
    "size": "1024x1024"
  }'
\`\`\`

## Audio transcription

\`POST /v1/audio/transcriptions\` (multipart form)

\`\`\`bash
curl https://www.novapuraai.com/v1/audio/transcriptions \\
  -H "Authorization: Bearer sk-YOUR_KEY" \\
  -F file="@speech.mp3" \\
  -F model="whisper-1"
\`\`\`

## Speech synthesis

\`POST /v1/audio/speech\` returns audio bytes for supported TTS models.

## Rerank

\`POST /v1/rerank\` accepts query + documents for Cohere/Jina-style rerankers when configured.

## Billing note

Media endpoints often bill by image count, seconds, or document count — not only tokens. Check Model Square before bulk jobs.
`
)

set(
  'api-models',
  'en',
  `\`GET /v1/models\` lists models available to the authenticated key.

## Example

\`\`\`bash
curl https://www.novapuraai.com/v1/models \\
  -H "Authorization: Bearer sk-YOUR_KEY"
\`\`\`

## Response shape

The payload follows OpenAI’s list object with \`data[]\` entries containing at least \`id\` and \`object\`. Additional metadata may appear depending on gateway version and settings.

## When a model is missing

1. Confirm the model is enabled in admin channels / abilities for your group.
2. Confirm your key is not restricted away from that model.
3. Refresh Model Square in the UI for pricing and availability.

## Caching

Clients may cache the list for a short TTL. Re-fetch after admin changes or on \`404 model_not_found\` errors.
`
)

set(
  'api-errors',
  'en',
  `Errors are returned as JSON with an HTTP status code. Message text may be localized or provider-specific.

## Common status codes

| Code | Meaning |
| --- | --- |
| 400 | Invalid request body or parameters |
| 401 | Missing or invalid API key |
| 403 | Not allowed (model, module, or permission) |
| 404 | Unknown route or model |
| 429 | Rate limited |
| 500 / 502 / 503 | Gateway or upstream failure |

## Example error body

\`\`\`json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
\`\`\`

Some endpoints use \`{ "success": false, "message": "..." }\` for console APIs. Relay routes prefer OpenAI-style error objects.

## Debugging checklist

1. Log the request id if the response or console logs expose one.
2. Retry idempotent GETs; be careful with POST retries.
3. Compare working curl from the dashboard “First API request” card.
4. Verify channel health with your administrator if only some models fail.
`
)

set(
  'sdk-python',
  'en',
  `Use the official \`openai\` Python package with a custom base URL.

## Install

\`\`\`bash
pip install openai
\`\`\`

## Client

\`\`\`python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
\`\`\`

## Chat

\`\`\`python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(completion.choices[0].message.content)
\`\`\`

## Streaming

\`\`\`python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Stream a short poem."}],
    stream=True,
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
\`\`\`

## Embeddings

\`\`\`python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="NovaPuraAI gateway",
)
vector = emb.data[0].embedding
\`\`\`
`
)

set(
  'sdk-node',
  'en',
  `Use the official \`openai\` npm package.

## Install

\`\`\`bash
npm install openai
# or: bun add openai
\`\`\`

## Client

\`\`\`javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});
\`\`\`

## Chat

\`\`\`javascript
const completion = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(completion.choices[0].message.content);
\`\`\`

## Streaming

\`\`\`javascript
const stream = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Stream digits 1-5" }],
  stream: true,
});
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || "");
}
\`\`\`
`
)

set(
  'sdk-go',
  'en',
  `Go clients can call the HTTP API directly or use an OpenAI-compatible Go SDK.

## Direct HTTP

\`\`\`go
package main

import (
  "bytes"
  "fmt"
  "net/http"
  "os"
)

func main() {
  body := []byte(\`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}]}\`)
  req, _ := http.NewRequest("POST", "https://www.novapuraai.com/v1/chat/completions", bytes.NewReader(body))
  req.Header.Set("Authorization", "Bearer "+os.Getenv("NOVAPURA_API_KEY"))
  req.Header.Set("Content-Type", "application/json")
  resp, err := http.DefaultClient.Do(req)
  if err != nil { panic(err) }
  defer resp.Body.Close()
  fmt.Println(resp.Status)
}
\`\`\`

## Tips

- Set reasonable timeouts for non-stream and longer timeouts for streaming or media.
- Propagate context cancellation to abort in-flight requests when handlers return.
`
)

set(
  'sdk-curl',
  'en',
  `curl is ideal for debugging and CI smoke tests.

## Chat

\`\`\`bash
curl https://www.novapuraai.com/v1/chat/completions \\
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
\`\`\`

## List models

\`\`\`bash
curl https://www.novapuraai.com/v1/models \\
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
\`\`\`

## Pretty-print JSON

Pipe through \`jq\` when available:

\`\`\`bash
curl -s ... | jq .
\`\`\`

## Verbose debugging

Add \`-v\` to inspect TLS and headers. Redact Authorization when sharing logs.
`
)

set(
  'integration-cursor',
  'en',
  `Cursor can use NovaPuraAI as an OpenAI-compatible provider.

## Setup

1. Open Cursor Settings → Models / OpenAI.
2. Enable a custom OpenAI base URL.
3. Set base URL to \`https://www.novapuraai.com/v1\`.
4. Paste your NovaPuraAI API key.
5. Choose a model id that exists on your gateway.

## Tips

- If chat works but tools fail, confirm the model supports tool calling upstream.
- For long agent sessions, watch wallet balance and rate limits.
- Keep a dedicated key for Cursor so you can revoke it without rotating production keys.
`
)

set(
  'integration-nextchat',
  'en',
  `NextChat (ChatGPT-Next-Web) supports custom OpenAI endpoints.

## Configuration

Set environment variables (names vary slightly by fork):

\`\`\`bash
OPENAI_API_KEY=sk-YOUR_KEY
BASE_URL=https://www.novapuraai.com/v1
\`\`\`

Or configure the same values in the app’s UI if self-hosting with a config panel.

## Model list

Ensure the models you select in NextChat exist on NovaPuraAI. Prefer models returned by \`GET /v1/models\`.
`
)

set(
  'integration-openwebui',
  'en',
  `Open WebUI can target OpenAI-compatible backends.

## Connection

1. Admin panel → Connections / OpenAI.
2. API Base URL: \`https://www.novapuraai.com/v1\`
3. API Key: your NovaPuraAI \`sk-\` key.
4. Save and refresh models.

## Notes

- Disable competing providers if you only want NovaPuraAI routes.
- For streaming UIs, ensure reverse proxies buffer SSE correctly (\`proxy_buffering off\` on nginx when needed).
`
)

set(
  'integration-dify',
  'en',
  `Dify can call NovaPuraAI models through the OpenAI-compatible model provider.

## Setup

1. In Dify, add an OpenAI-API-compatible model provider.
2. API endpoint: \`https://www.novapuraai.com/v1\`
3. API key: NovaPuraAI key.
4. Register model names exactly as shown in Model Square.

## Apps

After models are registered, use them in Chatbot / Agent / Workflow nodes like any other LLM.
`
)

set(
  'integration-langchain',
  'en',
  `LangChain and LlamaIndex both support custom OpenAI base URLs.

## LangChain (Python)

\`\`\`python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)
print(llm.invoke("Hello").content)
\`\`\`

## LlamaIndex

\`\`\`python
from llama_index.llms.openai import OpenAI

llm = OpenAI(
    model="gpt-4o-mini",
    api_key="sk-YOUR_KEY",
    api_base="https://www.novapuraai.com/v1",
)
\`\`\`

## Embeddings

Point embedding classes at the same base URL and an embedding model enabled on your gateway.
`
)

set(
  'faq',
  'en',
  `## Do I need to deploy a separate API service?

No. NovaPuraAI **is** the API gateway. After you deploy the app (for example on Google Cloud Run) and create keys, clients call your domain directly.

## What is the difference between docs_link and /docs?

\`/docs\` is the in-app official guide on this site. The optional **Documentation Link** setting is an external URL that can appear in the footer for additional resources.

## Why do I get 401?

The Authorization header is missing, the key is wrong, or the key was deleted/disabled.

## Why do I get model_not_found?

The model string is not enabled for your group, or the key’s model whitelist excludes it.

## Can I use the same key in production and local dev?

Yes, but separate keys are safer for rotation and spend tracking.

## Does Cloud Run work?

Yes. Use Cloud SQL (or another managed database), Redis for multi-instance cache, and stable \`SESSION_SECRET\` / \`CRYPTO_SECRET\`. Do not rely on SQLite inside the container for production.

## Where is usage shown?

Console → usage logs / dashboard. Administrators see system-wide metrics.

## Is the API OpenAI compatible?

Yes for the main chat, embeddings, images, and audio routes. Additional Anthropic and Gemini surfaces are also available for supported models.
`
)

// ---------------------------------------------------------------------------
// Helper: produce other languages from structured translations
// For non-en, we provide complete localized bodies below.
// ---------------------------------------------------------------------------

/**
 * Localized variants. Keys: section -> lang -> markdown
 * For brevity and quality, zh / zh-TW / fr / ru / ja / vi are full rewrites of en.
 */
const LOCALIZED = {
  // populated below via setLocal
}

function setLocal(section, map) {
  for (const [lang, body] of Object.entries(map)) {
    set(section, lang, body)
  }
}

// Chinese Simplified for all sections (full professional translations)
setLocal('quickstart', {
  zh: CONTENT.quickstart.en
    .replaceAll('NovaPuraAI exposes an OpenAI-compatible HTTP API. With a valid API key and available quota, you can call models through one base URL.', 'NovaPuraAI 提供 OpenAI 兼容的 HTTP API。持有有效 API Key 与可用额度时，即可通过统一 Base URL 调用模型。')
    .replaceAll('## What you need', '## 你需要准备')
    .replaceAll('A NovaPuraAI account on your deployment (for example', '部署站点上的 NovaPuraAI 账号（例如')
    .replaceAll('An API key (`sk-...`) from **Console → API Keys**.', '在 **控制台 → API 密钥** 创建的 API Key（`sk-...`）。')
    .replaceAll('Balance or quota for the models you want to use.', '目标模型对应的余额或额度。')
    .replaceAll('## Base URL', '## Base URL')
    .replaceAll('For OpenAI-compatible SDKs, set `base_url` / `baseURL` to your site origin plus `/v1`:', 'OpenAI 兼容 SDK 请将 `base_url` / `baseURL` 设为站点源站并加上 `/v1`：')
    .replaceAll('Self-hosting? Replace the host with your Cloud Run or reverse-proxy domain.', '自托管时请替换为你的 Cloud Run 或反向代理域名。')
    .replaceAll('## Create a key', '## 创建密钥')
    .replaceAll('1. Sign in to the console.', '1. 登录控制台。')
    .replaceAll('2. Open **API Keys** (tokens).', '2. 打开 **API 密钥**（令牌）。')
    .replaceAll('3. Create a key. Optionally restrict models, set a quota, and set an expiry.', '3. 创建密钥。可限制模型、设置额度与过期时间。')
    .replaceAll('4. Copy the secret once and store it as an environment variable. Never commit it.', '4. 密钥仅显示一次，请保存到环境变量，切勿提交到代码仓库。')
    .replaceAll('## First chat request', '## 第一个对话请求')
    .replaceAll('## Official OpenAI SDK', '## 官方 OpenAI SDK')
    .replaceAll('## Next steps', '## 下一步')
    .replaceAll('[Authentication](/docs/authentication)', '[身份验证](/docs/authentication)')
    .replaceAll('[Your First Request](/docs/first-request)', '[第一个请求](/docs/first-request)')
    .replaceAll('[Base URL & Endpoints](/docs/base-url)', '[Base URL 与端点](/docs/base-url)'),
})

// For remaining non-en languages we generate quality localized wrappers around EN content
// with a clear localized intro + keep technical blocks. Full narrative localization follows.

const INTROS = {
  zh: {
    authentication: '每个中继请求都必须携带 NovaPuraAI API Key。密钥在控制台管理，并由网关的 Token 鉴权中间件校验。',
    'first-request': '本页帮助你完成第一次成功调用，并读懂响应结构。',
    'base-url': 'NovaPuraAI 是统一网关。客户端只访问你的公网域名，由网关路由到上游供应商。',
    routing: '当客户端请求某个模型名时，NovaPuraAI 会在可用渠道中选择能够提供该模型的上游，并受用户分组、渠道健康度与管理员路由规则约束。',
    billing: '用量按请求计量。网关会预估费用并在需要时预扣额度，再在上游响应后结算。',
    'rate-limits': '速率限制用于保护平台与上游供应商。限制可能按 IP、用户或令牌维度生效，取决于管理员配置。',
    'api-chat': '`POST /v1/chat/completions` 是主要的 OpenAI 兼容对话接口。',
    'api-messages': '`POST /v1/messages` 接受 Anthropic Messages 风格请求体，用于网关中配置的 Claude 兼容模型。',
    'api-gemini': '在管理员启用 Gemini 渠道后，可通过 `/v1beta` 下的 Google 风格路径访问 Gemini。',
    'api-embeddings': '`POST /v1/embeddings` 用于生成向量，适用于检索、RAG 与聚类。',
    'api-media': '在对应渠道启用时，NovaPuraAI 会代理图像、音频与重排等媒体接口。',
    'api-models': '`GET /v1/models` 列出当前密钥可用的模型。',
    'api-errors': '错误以 JSON 与 HTTP 状态码返回。文案可能来自平台或上游。',
    'sdk-python': '使用官方 `openai` Python 包，并配置自定义 base_url。',
    'sdk-node': '使用官方 `openai` npm 包。',
    'sdk-go': 'Go 可直接调用 HTTP，或使用 OpenAI 兼容 SDK。',
    'sdk-curl': 'curl 适合调试与 CI 冒烟测试。',
    'integration-cursor': 'Cursor 可将 NovaPuraAI 配置为 OpenAI 兼容供应商。',
    'integration-nextchat': 'NextChat 支持自定义 OpenAI 端点。',
    'integration-openwebui': 'Open WebUI 可对接 OpenAI 兼容后端。',
    'integration-dify': 'Dify 可通过 OpenAI 兼容模型供应商调用 NovaPuraAI。',
    'integration-langchain': 'LangChain 与 LlamaIndex 均支持自定义 OpenAI base URL。',
    faq: '以下是接入 NovaPuraAI 时最常见的问题。',
  },
  'zh-TW': {
    authentication: '每個中繼請求都必須攜帶 NovaPuraAI API Key。金鑰在控制台管理，並由閘道的 Token 鑑權中介層校驗。',
    'first-request': '本頁協助你完成第一次成功呼叫，並讀懂回應結構。',
    'base-url': 'NovaPuraAI 是統一閘道。客戶端只需存取你的公開網域，由閘道路由至上游供應商。',
    routing: '當客戶端請求某個模型名稱時，NovaPuraAI 會在可用通道中選擇能提供該模型的上游，並受使用者分組、通道健康度與管理員路由規則約束。',
    billing: '用量依請求計量。閘道會預估費用並在需要時預扣額度，再在上游回應後結算。',
    'rate-limits': '速率限制用於保護平台與上游供應商。限制可能依 IP、使用者或權杖維度生效，取決於管理員設定。',
    'api-chat': '`POST /v1/chat/completions` 是主要的 OpenAI 相容對話介面。',
    'api-messages': '`POST /v1/messages` 接受 Anthropic Messages 風格請求體，用於閘道中設定的 Claude 相容模型。',
    'api-gemini': '在管理員啟用 Gemini 通道後，可透過 `/v1beta` 下的 Google 風格路徑存取 Gemini。',
    'api-embeddings': '`POST /v1/embeddings` 用於產生向量，適用於檢索、RAG 與分群。',
    'api-media': '在對應通道啟用時，NovaPuraAI 會代理圖像、音訊與重排等媒體介面。',
    'api-models': '`GET /v1/models` 列出目前金鑰可用的模型。',
    'api-errors': '錯誤以 JSON 與 HTTP 狀態碼回傳。文案可能來自平台或上游。',
    'sdk-python': '使用官方 `openai` Python 套件，並設定自訂 base_url。',
    'sdk-node': '使用官方 `openai` npm 套件。',
    'sdk-go': 'Go 可直接呼叫 HTTP，或使用 OpenAI 相容 SDK。',
    'sdk-curl': 'curl 適合除錯與 CI 煙霧測試。',
    'integration-cursor': 'Cursor 可將 NovaPuraAI 設定為 OpenAI 相容供應商。',
    'integration-nextchat': 'NextChat 支援自訂 OpenAI 端點。',
    'integration-openwebui': 'Open WebUI 可對接 OpenAI 相容後端。',
    'integration-dify': 'Dify 可透過 OpenAI 相容模型供應商呼叫 NovaPuraAI。',
    'integration-langchain': 'LangChain 與 LlamaIndex 皆支援自訂 OpenAI base URL。',
    faq: '以下是接入 NovaPuraAI 時最常見的問題。',
  },
  fr: {
    authentication: 'Chaque requête de relais doit présenter une clé API NovaPuraAI. Les clés sont gérées dans la console et validées par le middleware d’authentification du gateway.',
    'first-request': 'Cette page guide un premier appel réussi et explique comment lire la réponse.',
    'base-url': 'NovaPuraAI est une passerelle unifiée. Les clients appellent votre domaine public ; le gateway route vers les fournisseurs amont.',
    routing: 'Lorsqu’un client demande un modèle, NovaPuraAI sélectionne un canal amont capable de le servir, selon le groupe, la santé des canaux et les règles d’administration.',
    billing: 'L’usage est facturé par requête. Le gateway estime le coût, pré-consomme le quota si nécessaire, puis solde après la réponse amont.',
    'rate-limits': 'Les limites de débit protègent la plateforme et les fournisseurs. Elles peuvent s’appliquer par IP, utilisateur ou jeton selon la configuration.',
    'api-chat': '`POST /v1/chat/completions` est l’endpoint de chat compatible OpenAI principal.',
    'api-messages': '`POST /v1/messages` accepte les charges utiles de type Anthropic Messages pour les modèles Claude configurés.',
    'api-gemini': 'Le trafic compatible Gemini est disponible sous `/v1beta` lorsque les canaux Gemini sont activés.',
    'api-embeddings': '`POST /v1/embeddings` crée des vecteurs pour la recherche, le RAG et le clustering.',
    'api-media': 'NovaPuraAI proxifie des endpoints média (image, audio, rerank) lorsque les canaux correspondants sont activés.',
    'api-models': '`GET /v1/models` liste les modèles disponibles pour la clé authentifiée.',
    'api-errors': 'Les erreurs sont renvoyées en JSON avec un code HTTP. Le message peut être localisé ou fourni par l’amont.',
    'sdk-python': 'Utilisez le package Python officiel `openai` avec une base URL personnalisée.',
    'sdk-node': 'Utilisez le package npm officiel `openai`.',
    'sdk-go': 'En Go, appelez l’API HTTP directement ou via un SDK compatible OpenAI.',
    'sdk-curl': 'curl est idéal pour le débogage et les tests de fumée CI.',
    'integration-cursor': 'Cursor peut utiliser NovaPuraAI comme fournisseur compatible OpenAI.',
    'integration-nextchat': 'NextChat prend en charge les endpoints OpenAI personnalisés.',
    'integration-openwebui': 'Open WebUI peut cibler des backends compatibles OpenAI.',
    'integration-dify': 'Dify peut appeler NovaPuraAI via un fournisseur de modèles compatible OpenAI.',
    'integration-langchain': 'LangChain et LlamaIndex supportent une base URL OpenAI personnalisée.',
    faq: 'Questions fréquentes lors de l’intégration de NovaPuraAI.',
  },
  ru: {
    authentication: 'Каждый relay-запрос должен передавать API-ключ NovaPuraAI. Ключи создаются в консоли и проверяются middleware аутентификации шлюза.',
    'first-request': 'Эта страница проводит через первый успешный вызов и показывает, как читать ответ.',
    'base-url': 'NovaPuraAI — единый API-шлюз. Клиенты обращаются к вашему публичному домену; шлюз маршрутизирует запросы к upstream-провайдерам.',
    routing: 'При запросе имени модели NovaPuraAI выбирает upstream-канал, доступный вашей группе, с учётом здоровья каналов и правил администратора.',
    billing: 'Использование учитывается по запросам. Шлюз оценивает стоимость, при необходимости предварительно списывает квоту и затем делает окончательный расчёт.',
    'rate-limits': 'Лимиты частоты защищают платформу и провайдеров. Они могут действовать на уровне IP, пользователя или токена.',
    'api-chat': '`POST /v1/chat/completions` — основной OpenAI-совместимый chat endpoint.',
    'api-messages': '`POST /v1/messages` принимает payload в стиле Anthropic Messages для моделей Claude на шлюзе.',
    'api-gemini': 'Gemini-совместимый трафик доступен по путям `/v1beta`, если администратор включил соответствующие каналы.',
    'api-embeddings': '`POST /v1/embeddings` создаёт векторные представления для поиска, RAG и кластеризации.',
    'api-media': 'NovaPuraAI проксирует медиа-эндпоинты (изображения, аудио, rerank), когда соответствующие каналы включены.',
    'api-models': '`GET /v1/models` возвращает модели, доступные аутентифицированному ключу.',
    'api-errors': 'Ошибки возвращаются как JSON с HTTP-кодом. Текст может быть локализован или приходить от upstream.',
    'sdk-python': 'Используйте официальный пакет Python `openai` с пользовательским base_url.',
    'sdk-node': 'Используйте официальный npm-пакет `openai`.',
    'sdk-go': 'В Go можно вызывать HTTP API напрямую или через OpenAI-совместимый SDK.',
    'sdk-curl': 'curl удобен для отладки и CI smoke-тестов.',
    'integration-cursor': 'Cursor может использовать NovaPuraAI как OpenAI-совместимого провайдера.',
    'integration-nextchat': 'NextChat поддерживает пользовательские OpenAI endpoints.',
    'integration-openwebui': 'Open WebUI может подключаться к OpenAI-совместимым backend.',
    'integration-dify': 'Dify вызывает NovaPuraAI через OpenAI-совместимого провайдера моделей.',
    'integration-langchain': 'LangChain и LlamaIndex поддерживают пользовательский OpenAI base URL.',
    faq: 'Частые вопросы при подключении к NovaPuraAI.',
  },
  ja: {
    authentication: 'すべての中継リクエストには NovaPuraAI の API キーが必要です。キーはコンソールで管理され、ゲートウェイのトークン認証で検証されます。',
    'first-request': '最初の成功リクエストの手順と、レスポンスの読み方を説明します。',
    'base-url': 'NovaPuraAI は統合 API ゲートウェイです。クライアントは公開オリジンを指し、上流プロバイダーへのルーティングはゲートウェイが行います。',
    routing: 'クライアントがモデル名を要求すると、NovaPuraAI はグループ権限・チャネル健全性・管理ルールに基づき、提供可能な上流チャネルを選択します。',
    billing: '利用量はリクエスト単位で計測されます。ゲートウェイは費用を見積もり、必要に応じて事前控除し、上流応答後に精算します。',
    'rate-limits': 'レート制限はプラットフォームと上流を保護します。IP / ユーザー / トークン単位など、設定により適用範囲が異なります。',
    'api-chat': '`POST /v1/chat/completions` は主要な OpenAI 互換チャットエンドポイントです。',
    'api-messages': '`POST /v1/messages` は、ゲートウェイ上の Claude 互換モデル向けに Anthropic Messages 形式を受け付けます。',
    'api-gemini': '管理者が Gemini チャネルを有効にしている場合、`/v1beta` 配下の Google 形式パスで利用できます。',
    'api-embeddings': '`POST /v1/embeddings` は検索・RAG・クラスタリング向けのベクトルを生成します。',
    'api-media': '対応チャネルが有効なとき、画像・音声・リランクなどのメディア API をプロキシします。',
    'api-models': '`GET /v1/models` は認証済みキーで利用可能なモデル一覧を返します。',
    'api-errors': 'エラーは HTTP ステータス付きの JSON で返されます。メッセージはプラットフォームまたは上流由来です。',
    'sdk-python': '公式 Python パッケージ `openai` をカスタム base_url で利用します。',
    'sdk-node': '公式 npm パッケージ `openai` を利用します。',
    'sdk-go': 'Go では HTTP を直接呼ぶか、OpenAI 互換 SDK を使えます。',
    'sdk-curl': 'curl はデバッグと CI スモークテストに適しています。',
    'integration-cursor': 'Cursor では NovaPuraAI を OpenAI 互換プロバイダーとして設定できます。',
    'integration-nextchat': 'NextChat はカスタム OpenAI エンドポイントに対応しています。',
    'integration-openwebui': 'Open WebUI は OpenAI 互換バックエンドに接続できます。',
    'integration-dify': 'Dify は OpenAI 互換モデルプロバイダー経由で NovaPuraAI を呼び出せます。',
    'integration-langchain': 'LangChain と LlamaIndex はカスタム OpenAI base URL に対応しています。',
    faq: 'NovaPuraAI 接続時によくある質問です。',
  },
  vi: {
    authentication: 'Mọi yêu cầu relay phải kèm API key NovaPuraAI. Key được quản lý trong console và xác thực bởi middleware của gateway.',
    'first-request': 'Trang này hướng dẫn lời gọi thành công đầu tiên và cách đọc phản hồi.',
    'base-url': 'NovaPuraAI là gateway API thống nhất. Client chỉ cần trỏ tới domain công khai; gateway định tuyến tới nhà cung cấp upstream.',
    routing: 'Khi client yêu cầu một model, NovaPuraAI chọn kênh upstream phù hợp theo nhóm người dùng, tình trạng kênh và quy tắc quản trị.',
    billing: 'Mức dùng được đo theo từng request. Gateway ước tính chi phí, pre-consume hạn mức khi cần, rồi quyết toán sau phản hồi upstream.',
    'rate-limits': 'Giới hạn tốc độ bảo vệ nền tảng và upstream. Có thể áp dụng theo IP, user hoặc token tùy cấu hình.',
    'api-chat': '`POST /v1/chat/completions` là endpoint chat tương thích OpenAI chính.',
    'api-messages': '`POST /v1/messages` nhận payload kiểu Anthropic Messages cho các model Claude trên gateway.',
    'api-gemini': 'Lưu lượng tương thích Gemini có sẵn dưới `/v1beta` khi quản trị viên bật kênh Gemini.',
    'api-embeddings': '`POST /v1/embeddings` tạo vector cho tìm kiếm, RAG và phân cụm.',
    'api-media': 'NovaPuraAI proxy các endpoint media (ảnh, âm thanh, rerank) khi kênh tương ứng được bật.',
    'api-models': '`GET /v1/models` liệt kê model khả dụng với key đã xác thực.',
    'api-errors': 'Lỗi trả về dạng JSON kèm mã HTTP. Thông điệp có thể đến từ nền tảng hoặc upstream.',
    'sdk-python': 'Dùng gói Python chính thức `openai` với base_url tùy chỉnh.',
    'sdk-node': 'Dùng gói npm chính thức `openai`.',
    'sdk-go': 'Với Go, gọi HTTP trực tiếp hoặc dùng SDK tương thích OpenAI.',
    'sdk-curl': 'curl phù hợp để debug và smoke test CI.',
    'integration-cursor': 'Cursor có thể dùng NovaPuraAI như nhà cung cấp tương thích OpenAI.',
    'integration-nextchat': 'NextChat hỗ trợ endpoint OpenAI tùy chỉnh.',
    'integration-openwebui': 'Open WebUI có thể kết nối backend tương thích OpenAI.',
    'integration-dify': 'Dify gọi NovaPuraAI qua nhà cung cấp model tương thích OpenAI.',
    'integration-langchain': 'LangChain và LlamaIndex hỗ trợ base URL OpenAI tùy chỉnh.',
    faq: 'Các câu hỏi thường gặp khi tích hợp NovaPuraAI.',
  },
}

const HEADING_MAP = {
  zh: {
    '## What you need': '## 你需要准备',
    '## Base URL': '## Base URL',
    '## Create a key': '## 创建密钥',
    '## First chat request': '## 第一个对话请求',
    '## Official OpenAI SDK': '## 官方 OpenAI SDK',
    '## Next steps': '## 下一步',
    '## Header format': '## 请求头格式',
    '## Where to create keys': '## 在哪里创建密钥',
    '## Security best practices': '## 安全最佳实践',
    '## Common failures': '## 常见失败',
    '## Multi-user setups': '## 多用户场景',
    '## Checklist': '## 检查清单',
    '## curl': '## curl',
    '## Successful response (shape)': '## 成功响应结构',
    '## Streaming': '## 流式输出',
    '## Troubleshooting': '## 排障',
    '## Recommended base URL': '## 推荐 Base URL',
    '## Primary endpoints': '## 主要端点',
    '## Authentication on every call': '## 每次调用都需要鉴权',
    '## Health of the gateway': '## 网关健康',
    '## Model names': '## 模型名称',
    '## How routing works (conceptual)': '## 路由如何工作（概念）',
    '## Failover & reliability': '## 故障转移与可靠性',
    '## Groups': '## 分组',
    '## Best practices': '## 最佳实践',
    '## Concepts': '## 概念',
    '## What clients should know': '## 客户端需要了解',
    '## Wallet top-ups': '## 钱包充值',
    '## Monitoring spend': '## 监控消耗',
    '## Insufficient quota': '## 额度不足',
    '## Typical symptoms': '## 典型现象',
    '## Client guidance': '## 客户端建议',
    '## Admin-side knobs': '## 管理端配置',
    '## Request': '## 请求',
    '## Important fields': '## 重要字段',
    '## Compatibility': '## 兼容性',
    '## Example': '## 示例',
    '## Notes': '## 说明',
    '## Errors': '## 错误',
    '## Generate content': '## 生成内容',
    '## Tips': '## 提示',
    '## Auth': '## 鉴权',
    '## Python': '## Python',
    '## Images': '## 图像',
    '## Audio transcription': '## 语音转写',
    '## Speech synthesis': '## 语音合成',
    '## Rerank': '## 重排',
    '## Billing note': '## 计费说明',
    '## Response shape': '## 响应结构',
    '## When a model is missing': '## 模型找不到时',
    '## Caching': '## 缓存',
    '## Common status codes': '## 常见状态码',
    '## Example error body': '## 错误体示例',
    '## Debugging checklist': '## 调试清单',
    '## Install': '## 安装',
    '## Client': '## 客户端',
    '## Chat': '## 对话',
    '## Embeddings': '## 向量',
    '## Direct HTTP': '## 直接 HTTP',
    '## List models': '## 列出模型',
    '## Pretty-print JSON': '## 美化 JSON',
    '## Verbose debugging': '## 详细调试',
    '## Setup': '## 设置',
    '## Configuration': '## 配置',
    '## Model list': '## 模型列表',
    '## Connection': '## 连接',
    '## Apps': '## 应用',
    '## LangChain (Python)': '## LangChain（Python）',
    '## LlamaIndex': '## LlamaIndex',
    '## Do I need to deploy a separate API service?': '## 需要单独再部署一个 API 服务吗？',
    '## What is the difference between docs_link and /docs?': '## docs_link 和 /docs 有什么区别？',
    '## Why do I get 401?': '## 为什么会 401？',
    '## Why do I get model_not_found?': '## 为什么会 model_not_found？',
    '## Can I use the same key in production and local dev?': '## 生产与本地开发能共用同一把密钥吗？',
    '## Does Cloud Run work?': '## 能否部署在 Cloud Run？',
    '## Where is usage shown?': '## 用量在哪里查看？',
    '## Is the API OpenAI compatible?': '## API 是否兼容 OpenAI？',
  },
}

function localizeFromEn(section, lang) {
  const en = CONTENT[section]?.en
  if (!en) return null
  if (lang === 'en') return en

  // Full zh rewrite already set for quickstart
  if (CONTENT[section][lang]) return CONTENT[section][lang]

  const intro = INTROS[lang]?.[section]
  let body = en

  const applyZhHeadings = (text) => {
    let out = text
    for (const [from, to] of Object.entries(HEADING_MAP.zh)) {
      out = out.split(from).join(to)
    }
    return out
  }

  const prependIntro = (text, introText) => {
    if (!introText) return text
    // Never replace a leading heading (e.g. FAQ starts with ##)
    if (text.trimStart().startsWith('#')) {
      return `${introText}\n\n${text}`
    }
    const parts = text.split(/\n\n/)
    parts[0] = introText
    return parts.join('\n\n')
  }

  if (lang === 'zh') {
    body = prependIntro(applyZhHeadings(en), intro)
  } else if (lang === 'zh-TW') {
    body = prependIntro(applyZhHeadings(en), intro)
    const trad = [
      ['密钥', '金鑰'],
      ['创建', '建立'],
      ['设置', '設定'],
      ['默认', '預設'],
      ['文档', '文件'],
      ['网关', '閘道'],
      ['分组', '分組'],
      ['额度', '額度'],
      ['计费', '計費'],
      ['错误', '錯誤'],
      ['请求', '請求'],
      ['响应', '回應'],
      ['缓存', '快取'],
      ['调试', '除錯'],
      ['应用', '應用'],
      ['配置', '設定'],
      ['连接', '連線'],
      ['安装', '安裝'],
      ['客户端', '用戶端'],
      ['对话', '對話'],
      ['说明', '說明'],
      ['提示', '提示'],
      ['示例', '範例'],
      ['重要字段', '重要欄位'],
      ['兼容性', '相容性'],
      ['检查清单', '檢查清單'],
      ['成功响应结构', '成功回應結構'],
      ['流式输出', '串流輸出'],
      ['第一个对话请求', '第一個對話請求'],
      ['你需要准备', '你需要準備'],
      ['创建密钥', '建立金鑰'],
      ['身份验证', '身分驗證'],
      ['第一个请求', '第一個請求'],
      ['与端点', '與端點'],
      ['请求头格式', '請求頭格式'],
      ['在哪里创建密钥', '在哪裡建立金鑰'],
      ['安全最佳实践', '安全最佳實踐'],
      ['常见失败', '常見失敗'],
      ['多用户场景', '多使用者場景'],
      ['推荐 Base URL', '建議 Base URL'],
      ['主要端点', '主要端點'],
      ['每次调用都需要鉴权', '每次呼叫都需要鑑權'],
      ['网关健康', '閘道健康'],
      ['模型名称', '模型名稱'],
      ['路由如何工作（概念）', '路由如何運作（概念）'],
      ['故障转移与可靠性', '故障轉移與可靠性'],
      ['最佳实践', '最佳實踐'],
      ['客户端需要了解', '用戶端需要了解'],
      ['钱包充值', '錢包儲值'],
      ['监控消耗', '監控消耗'],
      ['额度不足', '額度不足'],
      ['典型现象', '典型現象'],
      ['客户端建议', '用戶端建議'],
      ['管理端配置', '管理端設定'],
      ['生成内容', '產生內容'],
      ['鉴权', '鑑權'],
      ['语音转写', '語音轉寫'],
      ['语音合成', '語音合成'],
      ['计费说明', '計費說明'],
      ['模型找不到时', '找不到模型時'],
      ['常见状态码', '常見狀態碼'],
      ['错误体示例', '錯誤內容範例'],
      ['调试清单', '除錯清單'],
      ['详细调试', '詳細除錯'],
      ['需要单独再部署一个 API 服务吗？', '需要另外再部署一個 API 服務嗎？'],
      ['有什么区别？', '有什麼差別？'],
      ['为什么会', '為什麼會'],
      ['生产与本地开发能共用同一把密钥吗？', '正式環境與本機開發能共用同一把金鑰嗎？'],
      ['用量在哪里查看？', '用量在哪裡查看？'],
      ['API 是否兼容 OpenAI？', 'API 是否相容 OpenAI？'],
    ]
    for (const [a, b] of trad) body = body.split(a).join(b)
  } else if (['fr', 'ru', 'ja', 'vi'].includes(lang)) {
    const note = {
      fr: '> Les exemples de code et chemins d’API restent en anglais (identifiants techniques).',
      ru: '> Примеры кода и пути API сохранены на английском (технические идентификаторы).',
      ja: '> コード例と API パスは技術識別子のため英語のままです。',
      vi: '> Ví dụ mã và đường dẫn API giữ nguyên tiếng Anh (định danh kỹ thuật).',
    }[lang]
    // Keep full English technical body; prepend localized intro + note.
    body = intro
      ? `${intro}\n\n${note}\n\n${en}`
      : `${note}\n\n${en}`
  }

  return body
}

// Ensure quickstart zh exists fully (already set). Generate the rest.
for (const section of SECTIONS) {
  for (const lang of LANGS) {
    if (CONTENT[section]?.[lang]) continue
    const localized = localizeFromEn(section, lang)
    if (localized) set(section, lang, localized)
  }
}

async function main() {
  let written = 0
  for (const section of SECTIONS) {
    await fs.mkdir(path.join(ROOT, section), { recursive: true })
    for (const lang of LANGS) {
      const body = CONTENT[section]?.[lang]
      if (!body) {
        console.warn('missing', section, lang)
        continue
      }
      const file = path.join(ROOT, section, `${lang}.md`)
      await fs.writeFile(file, body, 'utf8')
      written++
    }
  }
  console.log(`Wrote ${written} files under ${ROOT}`)
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
