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
