Khi client yêu cầu một tên mô hình, NovaPuraAI chọn kênh upstream có thể phục vụ mô hình đó, với điều kiện quyền nhóm, tình trạng kênh và quy tắc định tuyến quản trị.

## Tên mô hình

- Dùng định danh mô hình hiển thị trong **Model Square** hoặc `GET /v1/models`.
- Tên phân biệt chữ hoa/thường và phải khớp cấu hình quản trị.
- Cùng một mô hình logic có thể được phục vụ bởi nhiều khóa upstream để failover.

## Cách định tuyến hoạt động (khái niệm)

1. Xác thực API key và xác định nhóm người dùng.
2. Tìm các kênh cung cấp mô hình được yêu cầu cho nhóm đó.
3. Ưu tiên kênh khỏe; bỏ qua kênh tắt hoặc đang lỗi.
4. Chuyển tiếp yêu cầu, chuyển đổi định dạng nhà cung cấp khi cần, và tính phí sử dụng.

## Failover và độ tin cậy

Quản trị viên có thể cấu hình retry, cooldown và quy tắc tự tắt. Từ phía client bạn vẫn gọi một base URL ổn định — NovaPuraAI hấp thụ biến động nhà cung cấp khi có nhiều kênh.

## Nhóm

Người dùng có thể thuộc các nhóm với danh mục mô hình và tỷ lệ khác nhau. Nếu mô hình hoạt động với tài khoản này nhưng không với tài khoản kia, so sánh abilities nhóm và hạn chế khóa.

## Thực hành tốt

- Ghim tên mô hình trong config, không hard-code chuỗi rời rạc trên nhiều dịch vụ.
- Ưu tiên liệt kê mô hình qua API lúc khởi động nếu sản phẩm cần khám phá động.
- Xử lý `5xx` và timeout bằng retry phía client cho đọc idempotent; tránh retry mù cho tác dụng phụ không idempotent.
