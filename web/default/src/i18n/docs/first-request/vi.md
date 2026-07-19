Trang này hướng dẫn một lời gọi thành công đầu tiên đầy đủ và cách đọc phản hồi.

## Danh sách kiểm tra

- [ ] Bạn có khóa bắt đầu bằng `sk-`
- [ ] Tài khoản có số dư / quota dương
- [ ] Bạn biết ít nhất một tên mô hình đã bật (xem **Model Square** hoặc `GET /v1/models`)

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

## Phản hồi thành công (hình dạng)

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

## Streaming

Thêm `"stream": true` và đọc Server-Sent Events:

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

## Khắc phục sự cố

1. Xác nhận tên mô hình khớp chính xác với mô hình đã bật.
2. Xác nhận base URL có `/v1` cho OpenAI SDK.
3. Xác nhận HTTPS và reverse proxy / CDN cho phép stream dài nếu bạn streaming.
