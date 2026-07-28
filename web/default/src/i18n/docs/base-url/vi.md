NovaPuraAI cung cấp một gateway thống nhất. Client trỏ tới origin công khai; gateway định tuyến tới các nhà cung cấp upstream.

## Base URL khuyến nghị

| Loại client                             | Base URL                        |
| --------------------------------------- | ------------------------------- |
| OpenAI SDK / công cụ tương thích OpenAI | `https://www.novapuraai.com/v1` |
| HTTP thuần (đường dẫn đã gồm `/v1/...`) | `https://www.novapuraai.com`    |

## Các endpoint chính

| Phương thức | Đường dẫn                                | Mục đích             |
| ----------- | ---------------------------------------- | -------------------- |
| POST        | `/v1/chat/completions`                   | Chat (OpenAI)        |
| POST        | `/v1/completions`                        | Text completions     |
| POST        | `/v1/responses`                          | OpenAI Responses API |
| POST        | `/v1/messages`                           | Anthropic Messages   |
| POST        | `/v1/embeddings`                         | Embeddings           |
| POST        | `/v1/images/generations`                 | Tạo ảnh              |
| POST        | `/v1/audio/transcriptions`               | Speech-to-text       |
| POST        | `/v1/audio/speech`                       | Text-to-speech       |
| POST        | `/v1/rerank`                             | Rerank               |
| GET         | `/v1/models`                             | Liệt kê mô hình      |
| POST        | `/v1beta/models/{model}:generateContent` | Kiểu Gemini          |

Các route Midjourney và task khác cũng có thể khả dụng tùy cấu hình quản trị.

## Xác thực trên mọi lời gọi

Tất cả đường dẫn trên đều yêu cầu:

```http
Authorization: Bearer sk-YOUR_KEY
```

## Tình trạng gateway

Console quản trị và các endpoint trạng thái công khai báo cáo site đã sẵn sàng hay chưa. Trong production, hãy duy trì tính sẵn sàng cao cho gateway và cơ sở dữ liệu; không dựa vào bộ nhớ cục bộ tạm thời cho dữ liệu quan trọng với thanh toán.
