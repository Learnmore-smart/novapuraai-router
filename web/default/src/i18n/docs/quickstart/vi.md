NovaPuraAI cung cấp HTTP API tương thích OpenAI. Với API key hợp lệ và quota sẵn có, bạn có thể gọi mô hình qua một base URL duy nhất.

## Bạn cần gì

1. Tài khoản NovaPuraAI (ví dụ trên `https://www.novapuraai.com`).
2. API key (`sk-...`) từ **Console → API Keys**.
3. Số dư hoặc quota cho các mô hình bạn muốn dùng.

## Base URL

Với SDK tương thích OpenAI, đặt `base_url` / `baseURL` là origin site cộng `/v1`:

```text
https://www.novapuraai.com/v1
```

## Tạo khóa

1. Đăng nhập console.
2. Mở **API Keys** (tokens).
3. Tạo khóa. Tùy chọn hạn chế mô hình, đặt quota và hạn dùng.
4. Sao chép secret một lần và lưu vào biến môi trường. Không bao giờ commit secret.

## Yêu cầu chat đầu tiên

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
```

## OpenAI SDK chính thức

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(resp.choices[0].message.content)
```

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(resp.choices[0].message.content);
```

## Bước tiếp theo

- [Xác thực](/docs/authentication)
- [Yêu cầu đầu tiên](/docs/first-request)
- [Base URL và endpoint](/docs/base-url)
