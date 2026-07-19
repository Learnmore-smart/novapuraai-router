Câu trả lời cho các câu hỏi thường gặp về API gateway NovaPuraAI. Với trợ giúp giao diện sản phẩm ngoài API, hãy dùng kênh hỗ trợ console của hệ thống.

## NovaPuraAI là gì?

NovaPuraAI là API gateway tương thích OpenAI (sản phẩm dựa trên new-api). Bạn gửi yêu cầu tới một base URL duy nhất kèm API key; gateway xác thực, định tuyến theo mô hình, tính quota và ghi nhật ký sử dụng.

## Tài liệu ở đâu?

Giao diện tài liệu nhà phát triển chính thức nằm tại **`/docs`** (ví dụ `https://www.novapuraai.com/docs`).

## Nên dùng base URL nào?

- **Origin**: `https://www.novapuraai.com`
- **OpenAI SDK `base_url`**: `https://www.novapuraai.com/v1`

Xem [Base URL và endpoint](/docs/base-url).

## Làm sao lấy API key?

Đăng nhập → **Dashboard → API Keys / Tokens** → tạo khóa → sao chép secret `sk-...`. Chi tiết: [Xác thực](/docs/authentication).

## Vì sao nhận 401 Unauthorized?

Nguyên nhân thường gặp: thiếu header `Authorization`, khóa bị cắt, token bị tắt, hoặc đang dùng khóa OpenAI Platform thay vì khóa NovaPuraAI.

## Vì sao mô hình không tìm thấy?

Danh mục mô hình phụ thuộc triển khai và nhóm. Gọi `GET /v1/models` và dùng `id` trong phản hồi. Cấu hình kênh quản trị cũng có thể cần cập nhật.

## Có hỗ trợ API gốc Claude / Gemini không?

Có:

- Claude Messages: `POST /v1/messages`
- Gemini: `/v1beta/models/{model}:{action}`

OpenAI Chat Completions vẫn là đường dẫn phổ biến nhất cho ứng dụng đa nhà cung cấp.

## Thanh toán được tính như thế nào?

Theo quy tắc giá mô hình cấu hình trên gateway — thường là token cho chat/embeddings và đơn vị theo modality cho ảnh/âm thanh. Xem [Thanh toán và quota](/docs/billing) cùng nhật ký sử dụng trên console để biết số liệu chính thức.

## Hết quota thì sao?

Yêu cầu thất bại với lỗi kiểu insufficient-quota. Nạp hoặc đổi tín dụng trong console rồi thử lại. Hạn mức theo khóa có thể hết trước số dư tài khoản.

## Rate limit có giống quota không?

Không. **429** nghĩa là cần chậm lại; insufficient quota nghĩa là cần nạp số dư. Xem [Giới hạn tần suất](/docs/rate-limits).

## Có dùng được SDK OpenAI chính thức không?

Có — đặt API key thành khóa NovaPuraAI và `base_url` / `baseURL` thành `{ORIGIN}/v1`. Ví dụ: [Python](/docs/sdk-python), [Node.js](/docs/sdk-node), [Go](/docs/sdk-go), [curl](/docs/sdk-curl).

## Có hỗ trợ streaming không?

Có với mô hình/kênh hỗ trợ streaming. Dùng `"stream": true` trên Chat Completions hoặc endpoint streaming gốc của Claude/Gemini.

## Bảo mật khóa trong production thế nào?

Giữ khóa trên máy chủ hoặc secret store, xoay vòng sau khi lộ, áp dụng allowlist IP và mô hình khi có, tránh nhúng secret vào client mobile/web.

## Xem lịch sử yêu cầu ở đâu?

Trong **nhật ký sử dụng** trên console (và biểu đồ dashboard khi được bật). Khi báo quản trị viên, kèm timestamp và thân lỗi — không bao giờ gửi khóa thô.

## Vẫn kẹt?

1. Tái hiện bằng curl từ [Yêu cầu đầu tiên](/docs/first-request).
2. Kiểm tra [Lỗi](/docs/api-errors).
3. Xác nhận mô hình, quota và rate limit trong console.
4. Liên hệ quản trị viên với chi tiết yêu cầu đã che thông tin nhạy cảm.
