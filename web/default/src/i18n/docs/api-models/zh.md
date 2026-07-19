`GET /v1/models` 列出当前认证密钥可用的模型。

## 示例

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## 响应结构

负载遵循 OpenAI 的列表对象格式，`data[]` 中至少包含 `id` 与 `object`。根据网关版本与设置，可能还会出现额外元数据。

## 模型缺失时

1. 确认该模型已在管理后台的渠道 / 能力中为你的分组启用。
2. 确认你的密钥未被限制使用该模型。
3. 在界面中刷新模型广场，查看定价与可用性。

## 缓存

客户端可对列表做短 TTL 缓存。管理员变更后，或出现 `404 model_not_found` 错误时，请重新拉取。
