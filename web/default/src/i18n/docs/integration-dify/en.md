Dify can call NovaPuraAI as a custom OpenAI-compatible model provider. This lets apps, agents, and workflows in Dify use models routed through your gateway.

## Prerequisites

- A Dify workspace (self-hosted or cloud) with permission to add model providers
- NovaPuraAI API key and quota
- Origin such as `https://www.novapuraai.com`

## Add an OpenAI-API-compatible provider

In Dify **Settings → Model Providers** (or **Model Supplier**):

1. Choose **OpenAI-API-compatible** (name may vary slightly).
2. Set credentials:

| Field | Value |
| --- | --- |
| API Key | `sk-xxxxxxxx` |
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

| Symptom | Likely cause |
| --- | --- |
| Validation failed | Wrong endpoint (missing `/v1`) or key |
| Model not found | Name mismatch vs `GET /v1/models` |
| Timeout in long chains | Increase timeouts; reduce sequential LLM hops |
| Insufficient quota mid-run | Top up balance; cap retries in workflow |

## Related

- [Billing & Quota](/docs/billing)
- [Embeddings](/docs/api-embeddings)
- [Chat Completions](/docs/api-chat)
