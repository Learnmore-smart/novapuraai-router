每个中继请求都必须携带 NovaPuraAI API Key。密钥在控制台管理，并由网关的 Token 鉴权中间件校验。

## 请求头格式

Send the key as a Bearer token:

```http
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

Some OpenAI clients also accept `api_key` in the SDK constructor — that value becomes the same Authorization header.

## 在哪里创建密钥

1. Sign in → **API Keys**.
2. Create a key with an optional name.
3. Configure model allowlists, remaining quota, IP limits, and expiry if needed.
4. Save the secret immediately. The full secret is only shown once.

## 安全最佳实践

- Prefer environment variables (`NOVAPURA_API_KEY`) over hard-coding.
- Use separate keys per environment (dev / staging / production).
- Rotate keys if a client is compromised.
- Restrict keys to the minimum set of models your app needs.
- Do not embed keys in public frontend bundles.

## 常见失败

| Symptom | Likely cause |
| --- | --- |
| `401 Unauthorized` | Missing/invalid key, revoked key, or wrong header |
| `403 Forbidden` | Model not allowed for this key, or module disabled |
| `429` | Rate limit exceeded |
| Insufficient quota | Balance too low or key quota exhausted |

## 多用户场景

Administrators can issue keys to end users with independent quotas. Each key is billed against its owner’s balance according to platform settings.
