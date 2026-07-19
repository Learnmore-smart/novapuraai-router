`POST /v1/chat/completions` 是主要的 OpenAI 兼容对话接口。

## 请求

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

## 重要字段

| Field | Notes |
| --- | --- |
| `model` | Required. Must be enabled for your account |
| `messages` | OpenAI chat messages array |
| `stream` | `true` for SSE token streaming |
| `temperature` / `top_p` | Sampling controls |
| `max_tokens` / `max_completion_tokens` | Output bounds (provider-dependent) |
| `tools` / `tool_choice` | Function calling when the upstream model supports it |

## 流式输出

Set `"stream": true`. The response is `text/event-stream` with `data: {...}` chunks ending in `data: [DONE]`.

## 兼容性

Most tools that accept a custom OpenAI base URL work unchanged. Point them at `https://www.novapuraai.com/v1` and use your NovaPuraAI key.
