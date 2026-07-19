本页帮助你完成第一次成功调用，并读懂响应结构。

## 检查清单

- [ ] 你已拥有以 `sk-` 开头的密钥
- [ ] 账号余额 / 额度为正
- [ ] 你知道至少一个已启用的模型名称（见 **模型广场** 或 `GET /v1/models`）

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

## 成功响应结构

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "Hello!"},
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

## 流式输出

添加 `"stream": true` 并读取 Server-Sent Events：

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

## 排障

1. 确认模型名称与已启用模型完全一致。
2. 确认 OpenAI SDK 的 Base URL 包含 `/v1`。
3. 确认使用 HTTPS；若使用流式输出，请确保网关或 CDN 允许长连接。
