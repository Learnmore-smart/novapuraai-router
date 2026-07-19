レート制限はプラットフォームと上流を保護します。IP / ユーザー / トークン単位など、設定により適用範囲が異なります。

> コード例と API パスは技術識別子のため英語のままです。

Rate limits protect the platform and upstream providers. Limits may apply at IP, user, or token level depending on admin settings.

## Typical symptoms

- HTTP `429 Too Many Requests`
- Error messages mentioning rate limit or frequency

## Client guidance

1. Exponential backoff with jitter on `429` and transient `5xx`.
2. Reuse HTTP connections; avoid opening a new TLS session per tiny request when possible.
3. Batch work when the API supports it (for example embeddings arrays).
4. Cache model lists and static configuration.

## Streaming

Long-lived streams hold a connection. Design concurrency limits so you do not open more parallel streams than your plan allows.

## Admin-side knobs

Administrators can tune global and model-specific rate limits in system settings. Contact your site operator if legitimate traffic is throttled too aggressively.
