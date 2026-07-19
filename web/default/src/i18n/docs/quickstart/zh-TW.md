NovaPuraAI 提供 OpenAI 相容的 HTTP API。持有有效 API 金鑰與可用額度時，即可透過統一 Base URL 呼叫模型。

## 你需要準備

1. 部署站點上的 NovaPuraAI 帳號（例如 `https://www.novapuraai.com`）。
2. 在 **主控台 → API 金鑰** 建立的 API 金鑰（`sk-...`）。
3. 目標模型對應的餘額或額度。

## Base URL

OpenAI 相容 SDK 請將 `base_url` / `baseURL` 設為站點來源站並加上 `/v1`：

```text
https://www.novapuraai.com/v1
```

## 建立金鑰

1. 登入主控台。
2. 開啟 **API 金鑰**（權杖）。
3. 建立金鑰。可限制模型、設定額度與到期時間。
4. 金鑰僅顯示一次，請儲存到環境變數，切勿提交到程式碼儲存庫。

## 第一個對話請求

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
```

## 官方 OpenAI SDK

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

## 下一步

- [身份驗證](/docs/authentication)
- [第一個請求](/docs/first-request)
- [Base URL 與端點](/docs/base-url)
