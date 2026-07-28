本頁協助你完成第一次成功呼叫，並讀懂回應結構。

## 檢查清單

- [ ] 你已擁有以 `sk-` 開頭的金鑰
- [ ] 帳號餘額 / 額度為正
- [ ] 你知道至少一個已啟用的模型名稱（見 **模型廣場** 或 `GET /v1/models`）

## curl

```bash
export NOVAPURA_API_KEY=sk-YOUR_KEY
export NOVAPURA_BASE=https://www.novapuraai.com

curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are a concise assistant."},
      {"role": "user", "content": "Say hello in one sentence."}
    ],
    "temperature": 0.7
  }'
```

## 成功回應結構

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": { "role": "assistant", "content": "Hello!" },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
```

## 串流輸出

新增 `"stream": true` 並讀取 Server-Sent Events：

```bash
curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to five."}]
  }'
```

## 疑難排解

1. 確認模型名稱與已啟用模型完全一致。
2. 確認 OpenAI SDK 的 Base URL 包含 `/v1`。
3. 確認使用 HTTPS；若使用串流輸出，請確保閘道或 CDN 允許長連線。
