当客户端请求某个模型名时，NovaPuraAI 会在可用渠道中选择能够提供该模型的上游，并受用户分组、渠道健康度与管理员路由规则约束。

## 模型名称

- Use the model identifiers shown in **Model Square** or `GET /v1/models`.
- Names are case-sensitive and must match admin configuration.
- The same logical model may be served by multiple upstream keys for failover.

## 路由如何工作（概念）

1. Authenticate the API key and resolve the user group.
2. Find channels that expose the requested model for that group.
3. Prefer healthy channels; skip disabled or failing channels.
4. Forward the request, translate provider formats when needed, and bill usage.

## 故障转移与可靠性

Administrators can configure retries, cooldowns, and auto-disable rules. From the client perspective you still call one stable base URL — NovaPuraAI absorbs provider churn when multiple channels are available.

## 分组

Users may belong to groups with different model catalogs and ratios. If a model works for one account but not another, compare group abilities and key restrictions.

## 最佳实践

- Pin model names in config, not hard-coded one-off strings across many services.
- Prefer listing models via API at startup if your product needs dynamic discovery.
- Handle `5xx` and timeouts with client-side retries for idempotent reads; avoid blind retries for non-idempotent side effects.
