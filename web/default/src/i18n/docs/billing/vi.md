Mức sử dụng được đo theo từng yêu cầu. Gateway ước tính chi phí, pre-consume quota khi cần, rồi settle sau phản hồi upstream.

## Khái niệm

- **Số dư / quota** — tín dụng trả trước trên tài khoản người dùng (và tùy chọn theo khóa).
- **Giá mô hình** — do quản trị viên cấu hình (tỷ lệ token, giá cố định hoặc quy tắc dựa trên biểu thức).
- **Pre-consume** — giữ quota ước tính để các yêu cầu đồng thời không chi vượt.
- **Settle** — điều chỉnh khoản trừ cuối theo số token thực tế khi có.

## Client cần biết

1. Lời gọi API thành công vẫn có thể lỗi sau nếu upstream trả lỗi sau pre-consume (phần giữ chưa dùng được hoàn theo logic nền tảng).
2. Phản hồi streaming tính phí theo mức dùng hoàn tất khi nhà cung cấp báo token.
3. Sản phẩm ảnh, âm thanh và video có thể tính theo số lượng, thời lượng hoặc hệ số độ phân giải — luôn kiểm tra Model Square.

## Nạp ví

Người dùng cuối nạp từ trang **Wallet** khi cổng thanh toán được bật (ví dụ Stripe). Quản trị viên kiểm soát tiền tệ, khuyến mãi và hạn mức.

## Theo dõi chi tiêu

- Nhật ký sử dụng trên console hiển thị mô hình, token và chênh lệch quota theo từng yêu cầu.
- Tách API key theo sản phẩm để dễ quy chi phí.

## Không đủ quota

Nếu yêu cầu bị từ chối vì quota, hãy nạp ví hoặc nhờ quản trị viên tăng hạn mức. Tạo khóa mới không tạo số dư miễn phí.
