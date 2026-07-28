> コード例と API パスは技術識別子のため英語のままです。

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
