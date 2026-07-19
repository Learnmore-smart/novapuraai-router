在管理員啟用 Gemini 通道後，可透過 `/v1beta` 下的 Google 風格路徑存取 Gemini。

## 產生內容

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

## 提示

- Exact model IDs depend on your deployment’s channel configuration.
- Multimodal parts (inline data / file URIs) follow Gemini request shapes; keep payloads within gateway body limits.
- You may also access some Gemini models through OpenAI-compatible chat if a channel adapter maps them.

## 鑑權

Use the same NovaPuraAI `sk-` key. Do not send Google AI Studio keys to NovaPuraAI unless you are an admin configuring upstream channels.
