`POST /v1/chat/completions` là endpoint chat chính tương thích OpenAI.

## Yêu cầu

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

## Các trường quan trọng

| Trường                                 | Ghi chú                                       |
| -------------------------------------- | --------------------------------------------- |
| `model`                                | Bắt buộc. Phải được bật cho tài khoản của bạn |
| `messages`                             | Mảng tin nhắn chat theo định dạng OpenAI      |
| `stream`                               | `true` để streaming token qua SSE             |
| `temperature` / `top_p`                | Điều khiển sampling                           |
| `max_tokens` / `max_completion_tokens` | Giới hạn đầu ra (phụ thuộc nhà cung cấp)      |
| `tools` / `tool_choice`                | Function calling khi mô hình upstream hỗ trợ  |

## Streaming

Đặt `"stream": true`. Phản hồi là `text/event-stream` với các đoạn `data: {...}`, kết thúc bằng `data: [DONE]`.

## Tương thích

Hầu hết công cụ chấp nhận base URL OpenAI tùy chỉnh đều hoạt động không cần chỉnh sửa. Trỏ tới `https://www.novapuraai.com/v1` và dùng khóa NovaPuraAI của bạn.
