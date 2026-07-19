本页帮助你完成第一次成功调用，并读懂响应结构。

## 检查清单

- [ ] You have a key starting with `sk-`
- [ ] Your account has positive balance / quota
- [ ] You know at least one enabled model name (see **Model Square** or `GET /v1/models`)

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

Add `"stream": true` and read Server-Sent Events:

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

1. Confirm the model name exactly matches an enabled model.
2. Confirm the base URL includes `/v1` for OpenAI SDKs.
3. Confirm HTTPS and that Cloud Run / CDN allows long-running streams if you stream.
