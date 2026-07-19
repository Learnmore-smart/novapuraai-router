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
