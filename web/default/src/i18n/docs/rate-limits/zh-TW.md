速率限制用於保護平台與上游供應商。限制可能依 IP、使用者或權杖維度生效，取決於管理員設定。

## 典型現象

- HTTP `429 Too Many Requests`
- Error messages mentioning rate limit or frequency

## 用戶端建议

1. Exponential backoff with jitter on `429` and transient `5xx`.
2. Reuse HTTP connections; avoid opening a new TLS session per tiny request when possible.
3. Batch work when the API supports it (for example embeddings arrays).
4. Cache model lists and static configuration.

## 串流輸出

Long-lived streams hold a connection. Design concurrency limits so you do not open more parallel streams than your plan allows.

## 管理端設定

Administrators can tune global and model-specific rate limits in system settings. Contact your site operator if legitimate traffic is throttled too aggressively.
