Lỗi được trả về dưới dạng JSON kèm mã trạng thái HTTP. Nội dung thông báo có thể được bản địa hóa hoặc phụ thuộc nhà cung cấp.

## Các mã trạng thái thường gặp

| Mã | Ý nghĩa |
| --- | --- |
| 400 | Thân yêu cầu hoặc tham số không hợp lệ |
| 401 | Thiếu hoặc API key không hợp lệ |
| 403 | Không được phép (mô hình, module hoặc quyền) |
| 404 | Route hoặc mô hình không tồn tại |
| 429 | Bị giới hạn tần suất (rate limit) |
| 500 / 502 / 503 | Lỗi gateway hoặc upstream |

## Ví dụ thân lỗi

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

Một số endpoint console dùng `{ "success": false, "message": "..." }`. Các route relay ưu tiên đối tượng lỗi kiểu OpenAI.

## Danh sách kiểm tra gỡ lỗi

1. Ghi lại request id nếu phản hồi hoặc nhật ký console có cung cấp.
2. Thử lại các GET idempotent; cẩn thận khi retry POST.
3. So sánh với lệnh curl đang hoạt động từ thẻ «Yêu cầu API đầu tiên» trên dashboard.
4. Kiểm tra tình trạng kênh với quản trị viên nếu chỉ một số mô hình bị lỗi.
