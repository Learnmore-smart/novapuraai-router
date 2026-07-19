`GET /v1/models` liệt kê các mô hình khả dụng với khóa đã xác thực.

## Ví dụ

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## Hình dạng phản hồi

Payload theo đối tượng danh sách OpenAI với các mục `data[]` chứa ít nhất `id` và `object`. Metadata bổ sung có thể xuất hiện tùy phiên bản gateway và cấu hình.

## Khi thiếu mô hình

1. Xác nhận mô hình đã được bật trong kênh / abilities quản trị cho nhóm của bạn.
2. Xác nhận khóa không bị hạn chế khỏi mô hình đó.
3. Làm mới Model Square trên giao diện để xem giá và tình trạng sẵn có.

## Bộ nhớ đệm

Client có thể cache danh sách trong TTL ngắn. Gọi lại sau thay đổi quản trị hoặc khi gặp lỗi `404 model_not_found`.
