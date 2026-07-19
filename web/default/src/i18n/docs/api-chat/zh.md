`POST /v1/chat/completions` 是主要的 OpenAI 兼容对话端点。

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

| 字段 | 说明 |
| --- | --- |
| `model` | 必填。必须已为你的账号启用 |
| `messages` | OpenAI 对话消息数组 |
| `stream` | 设为 `true` 启用 SSE 令牌流式输出 |
| `temperature` / `top_p` | 采样控制 |
| `max_tokens` / `max_completion_tokens` | 输出上限（取决于上游提供商） |
| `tools` / `tool_choice` | 在上游模型支持时用于函数调用 |

## 流式输出

设置 `"stream": true`。响应为 `text/event-stream`，包含 `data: {...}` 分片，并以 `data: [DONE]` 结束。

## 兼容性

大多数支持自定义 OpenAI Base URL 的工具可直接使用。将地址指向 `https://www.novapuraai.com/v1`，并使用你的 NovaPuraAI 密钥即可。
