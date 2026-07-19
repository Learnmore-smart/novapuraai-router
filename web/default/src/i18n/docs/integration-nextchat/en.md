NextChat (ChatGPT-Next-Web and compatible forks) can talk to NovaPuraAI through OpenAI-compatible settings. You configure the base URL, API key, and default model once, then use the web UI as usual.

## Prerequisites

- Running NextChat (self-hosted or local)
- NovaPuraAI API key
- Deployment origin such as `https://www.novapuraai.com`

## Settings to apply

In NextChat **Settings** (wording may vary by fork/version):

| Field | Recommended value |
| --- | --- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key | `sk-xxxxxxxx` |
| Model | An ID from `GET /v1/models` |

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

| Symptom | Cause | Fix |
| --- | --- | --- |
| 404 on chat | Wrong base URL (`/v1` missing or doubled) | Align with network tab path |
| 401 | Key not passed or wrong | Paste NovaPuraAI key, not OpenAI platform key |
| Model list empty | Frontend cannot call `/v1/models` | Check CORS/proxy and key permissions |
| Balance errors | No quota | Top up in NovaPuraAI console |

## Security notes

- Prefer server-side proxy mode if your NextChat build supports hiding keys from the browser.
- For public demos, use low-quota keys and strict model allowlists.

## Related

- [Base URL & Endpoints](/docs/base-url)
- [Authentication](/docs/authentication)
- [FAQ](/docs/faq)
