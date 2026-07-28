Mọi yêu cầu relay phải trình bày API key NovaPuraAI. Khóa được quản lý trong console và được `TokenAuth` xác thực trên gateway.

## Định dạng header

Gửi khóa dưới dạng Bearer token:

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

Một số client OpenAI cũng chấp nhận `api_key` trong constructor SDK — giá trị đó trở thành cùng header Authorization.

## Nơi tạo khóa

1. Đăng nhập → **API Keys**.
2. Tạo khóa với tên tùy chọn.
3. Cấu hình allowlist mô hình, quota còn lại, giới hạn IP và hạn dùng nếu cần.
4. Lưu secret ngay. Secret đầy đủ chỉ hiện một lần.

## Thực hành bảo mật tốt

- Ưu tiên biến môi trường (`NOVAPURA_API_KEY`) thay vì hard-code.
- Dùng khóa riêng cho từng môi trường (dev / staging / production).
- Xoay vòng khóa nếu client bị lộ.
- Giới hạn khóa ở tập mô hình tối thiểu mà ứng dụng cần.
- Không nhúng khóa vào bundle frontend công khai.

## Lỗi thường gặp

| Triệu chứng        | Nguyên nhân có thể                                       |
| ------------------ | -------------------------------------------------------- |
| `401 Unauthorized` | Thiếu/khóa không hợp lệ, khóa bị thu hồi hoặc sai header |
| `403 Forbidden`    | Mô hình không được phép cho khóa này, hoặc module bị tắt |
| `429`              | Vượt rate limit                                          |
| Không đủ quota     | Số dư quá thấp hoặc quota khóa đã hết                    |

## Thiết lập nhiều người dùng

Quản trị viên có thể cấp khóa cho người dùng cuối với quota độc lập. Mỗi khóa được tính phí vào số dư chủ sở hữu theo cấu hình nền tảng.
