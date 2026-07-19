`POST /v1/messages` nhận payload kiểu Anthropic Messages cho các mô hình tương thích Claude đã cấu hình trên gateway.

## Ví dụ

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

## Ghi chú

- Tên mô hình phải tồn tại trong danh mục NovaPuraAI.
- Một số header chỉ dành cho Anthropic được chấp nhận và chuyển tiếp khi phù hợp.
- Bạn thường có thể gọi cùng mô hình Claude qua định dạng chat OpenAI nếu bộ chuyển kênh hỗ trợ — ưu tiên định dạng mà SDK của bạn mong đợi.

## Lỗi

Schema không hợp lệ hoặc trường không được hỗ trợ trả về `4xx` kèm thân lỗi JSON. Kiểm tra rằng `max_tokens` có mặt khi Messages API yêu cầu.
