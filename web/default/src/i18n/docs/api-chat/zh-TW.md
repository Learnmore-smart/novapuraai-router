`POST /v1/chat/completions` 是主要的 OpenAI 相容對話端點。

## 請求

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are helpful."},
      {"role": "user", "content": "Summarize NovaPuraAI in one sentence."}
    ],
    "temperature": 0.5,
    "max_tokens": 256
  }'
```

## 重要欄位

| 欄位 | 說明 |
| --- | --- |
| `model` | 必填。必須已為你的帳號啟用 |
| `messages` | OpenAI 對話訊息陣列 |
| `stream` | 設為 `true` 啟用 SSE 權杖串流輸出 |
| `temperature` / `top_p` | 取樣控制 |
| `max_tokens` / `max_completion_tokens` | 輸出上限（取決於上游供應商） |
| `tools` / `tool_choice` | 在上游模型支援時用於函式呼叫 |

## 串流輸出

設定 `"stream": true`。回應為 `text/event-stream`，包含 `data: {...}` 分片，並以 `data: [DONE]` 結束。

## 相容性

大多數支援自訂 OpenAI Base URL 的工具可直接使用。將位址指向 `https://www.novapuraai.com/v1`，並使用你的 NovaPuraAI 金鑰即可。
