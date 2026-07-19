Lưu lượng tương thích Gemini có sẵn qua các đường dẫn kiểu Google dưới `/v1beta` khi quản trị viên bật kênh Gemini.

## Tạo nội dung

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
```

## Mẹo

- ID mô hình chính xác phụ thuộc cấu hình kênh trên hệ thống của bạn.
- Các phần đa phương thức (inline data / file URI) theo hình dạng yêu cầu Gemini; giữ payload trong giới hạn thân yêu cầu của gateway.
- Bạn cũng có thể truy cập một số mô hình Gemini qua chat tương thích OpenAI nếu bộ chuyển kênh ánh xạ chúng.

## Xác thực

Dùng cùng khóa NovaPuraAI tiền tố `sk-`. Không gửi khóa Google AI Studio tới NovaPuraAI trừ khi bạn là quản trị viên đang cấu hình kênh upstream.
