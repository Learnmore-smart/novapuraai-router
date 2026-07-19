`POST /v1/messages` 接受 Anthropic Messages 风格的负载，用于网关上已配置的 Claude 兼容模型。

## 示例

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

## 说明

- 模型名称必须存在于你的 NovaPuraAI 目录中。
- 部分仅 Anthropic 使用的请求头在相关时会被接受并转发。
- 若渠道适配器支持，通常也可通过 OpenAI 对话格式调用同一 Claude 模型——请优先使用你的 SDK 期望的格式。

## 错误

无效 schema 或不支持的字段会返回带 JSON 错误体的 `4xx`。请确认 Messages API 要求时已提供 `max_tokens`。
