管理者が Gemini チャネルを有効にしている場合、`/v1beta` 配下の Google 形式パスで利用できます。

> コード例と API パスは技術識別子のため英語のままです。

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
