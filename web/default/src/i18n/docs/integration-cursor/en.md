Cursor can use NovaPuraAI as an OpenAI-compatible model provider. You point Cursor at your deployment origin’s `/v1` API and authenticate with a NovaPuraAI API key.

## What you need

- A NovaPuraAI API key from **Dashboard → API Keys / Tokens**
- Your deployment origin, for example `https://www.novapuraai.com`
- Sufficient quota for the models you select in Cursor

## Configure OpenAI-compatible access

Cursor’s UI labels change over releases. Look for **OpenAI API**, **OpenAI Compatible**, **Override OpenAI Base URL**, or **Custom model provider** settings.

Typical values:

| Setting | Value |
| --- | --- |
| API key | `sk-xxxxxxxx` (your NovaPuraAI key) |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | A model ID from `GET /v1/models` |

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

| Issue | Fix |
| --- | --- |
| Auth errors in Cursor | Re-paste the full key including `sk-` |
| Network / CORS style failures | Cursor is a desktop client; usually not CORS—check URL typos and VPN |
| Empty responses | Confirm quota and try the same model via curl |
| Rate limits | Reduce parallel agent runs; see [Rate Limits](/docs/rate-limits) |

## Security

- Do not share workspace settings files that embed secrets.
- Rotate keys if a machine is lost or a key was committed.

## Related

- [Authentication](/docs/authentication)
- [Models List](/docs/api-models)
- [Chat Completions](/docs/api-chat)
