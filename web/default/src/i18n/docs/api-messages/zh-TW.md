`POST /v1/messages` 接受 Anthropic Messages 風格請求體，用於閘道中設定的 Claude 相容模型。

## 範例

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

## 說明

- Model names must exist in your NovaPuraAI catalog.
- Some Anthropic-only headers are accepted and forwarded when relevant.
- You can often call the same Claude models via OpenAI chat format if the channel adapter supports it — prefer the format your SDK expects.

## 錯誤

Invalid schema or unsupported fields return `4xx` with a JSON error body. Check that `max_tokens` is present when required by the Messages API.
