速率限制用于保护平台与上游供应商。限制可能按 IP、用户或令牌维度生效，取决于管理员配置。

## 典型现象

- HTTP `429 Too Many Requests`
- Error messages mentioning rate limit or frequency

## 客户端建议

1. Exponential backoff with jitter on `429` and transient `5xx`.
2. Reuse HTTP connections; avoid opening a new TLS session per tiny request when possible.
3. Batch work when the API supports it (for example embeddings arrays).
4. Cache model lists and static configuration.

## 流式输出

Long-lived streams hold a connection. Design concurrency limits so you do not open more parallel streams than your plan allows.

## 管理端配置

Administrators can tune global and model-specific rate limits in system settings. Contact your site operator if legitimate traffic is throttled too aggressively.
