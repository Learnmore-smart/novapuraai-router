NovaPuraAI 提供 OpenAI 兼容的 HTTP API。持有有效 API Key 与可用额度时，即可通过统一 Base URL 调用模型。

## 你需要准备

1. 部署站点上的 NovaPuraAI 账号（例如 `https://www.novapuraai.com`).
2. 在 **控制台 → API 密钥** 创建的 API Key（`sk-...`）。
3. 目标模型对应的余额或额度。

## Base URL

OpenAI 兼容 SDK 请将 `base_url` / `baseURL` 设为站点源站并加上 `/v1`：

```text
https://www.novapuraai.com/v1
```

## 创建密钥

1. 登录控制台。
2. 打开 **API 密钥**（令牌）。
3. 创建密钥。可限制模型、设置额度与过期时间。
4. 密钥仅显示一次，请保存到环境变量，切勿提交到代码仓库。

## 第一个对话请求

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

- [身份验证](/docs/authentication)
- [第一个请求](/docs/first-request)
- [Base URL 与端点](/docs/base-url)
