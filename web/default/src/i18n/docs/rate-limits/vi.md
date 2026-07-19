Giới hạn tần suất bảo vệ nền tảng và nhà cung cấp upstream. Hạn mức có thể áp dụng ở cấp IP, người dùng hoặc token tùy cấu hình quản trị.

## Triệu chứng thường gặp

- HTTP `429 Too Many Requests`
- Thông báo lỗi đề cập rate limit hoặc tần suất

## Hướng dẫn cho client

1. Exponential backoff kèm jitter với `429` và `5xx` tạm thời.
2. Tái sử dụng kết nối HTTP; tránh mở phiên TLS mới cho mỗi yêu cầu nhỏ khi có thể.
3. Gộp việc theo lô khi API hỗ trợ (ví dụ mảng embeddings).
4. Cache danh sách mô hình và cấu hình tĩnh.

## Streaming

Stream kéo dài giữ kết nối. Thiết kế giới hạn đồng thời để không mở nhiều stream song song hơn mức gói cho phép.

## Nút điều chỉnh phía quản trị

Quản trị viên có thể tinh chỉnh rate limit toàn cục và theo mô hình trong cài đặt hệ thống. Liên hệ nhà vận hành site nếu lưu lượng hợp lệ bị chặn quá chặt.
