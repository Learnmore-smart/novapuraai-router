Khi client yêu cầu một model, NovaPuraAI chọn kênh upstream phù hợp theo nhóm người dùng, tình trạng kênh và quy tắc quản trị.

> Ví dụ mã và đường dẫn API giữ nguyên tiếng Anh (định danh kỹ thuật).

When a client requests a model name, NovaPuraAI selects an upstream channel that can serve that model, subject to group permissions, channel health, and admin routing rules.

## Model names

- Use the model identifiers shown in **Model Square** or `GET /v1/models`.
- Names are case-sensitive and must match admin configuration.
- The same logical model may be served by multiple upstream keys for failover.

## How routing works (conceptual)

1. Authenticate the API key and resolve the user group.
2. Find channels that expose the requested model for that group.
3. Prefer healthy channels; skip disabled or failing channels.
4. Forward the request, translate provider formats when needed, and bill usage.

## Failover & reliability

Administrators can configure retries, cooldowns, and auto-disable rules. From the client perspective you still call one stable base URL — NovaPuraAI absorbs provider churn when multiple channels are available.

## Groups

Users may belong to groups with different model catalogs and ratios. If a model works for one account but not another, compare group abilities and key restrictions.

## Best practices

- Pin model names in config, not hard-coded one-off strings across many services.
- Prefer listing models via API at startup if your product needs dynamic discovery.
- Handle `5xx` and timeouts with client-side retries for idempotent reads; avoid blind retries for non-idempotent side effects.
