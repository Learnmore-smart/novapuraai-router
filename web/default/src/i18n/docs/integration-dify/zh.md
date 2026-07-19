Dify 可将 NovaPuraAI 添加为自定义的 OpenAI 兼容模型供应商，从而在应用、Agent 与工作流中使用经网关路由的模型。

## 前置条件

- 具备添加模型供应商权限的 Dify 工作区（自托管或云版）
- NovaPuraAI API Key 与额度
- 源站例如 `https://www.novapuraai.com`

## 添加 OpenAI-API-compatible 供应商

在 Dify **设置 → 模型供应商**（或 **Model Supplier**）：

1. 选择 **OpenAI-API-compatible**（名称可能略有差异）。
2. 填写凭证：

| 字段 | 值 |
| --- | --- |
| API Key | `sk-xxxxxxxx` |
| API endpoint URL | `https://www.novapuraai.com/v1` |

3. 添加一个或多个模型，**模型名必须与** NovaPuraAI 返回的名称完全一致（例如 `gpt-4o-mini`）。
4. 配置上下文长度 / 模式（chat 与 completion），与模型类型匹配。
5. 保存，并在可用时运行 Dify 的连接测试。

## 端点预期

Dify 通常会请求：

- 聊天模型：`POST /v1/chat/completions`
- 配置了 embedding 模型时：`POST /v1/embeddings`
- 仅当供应商集成做发现时：`GET /v1/models`

在调试 Dify 流程前，请先用 curl 直接验证。

## Agent 与工作流建议

- 为便宜与高价 NovaPuraAI 模型分别创建 Dify 模型条目。
- 在节点配置中设置合理的 max token，以控制成本。
- 工具 / function calling 请选择通道支持 tools 的模型。

## 常见失败

| 现象 | 可能原因 |
| --- | --- |
| 校验失败 | 端点错误（缺少 `/v1`）或密钥错误 |
| Model not found | 名称与 `GET /v1/models` 不一致 |
| 长链路超时 | 增大超时；减少串联 LLM 跳数 |
| 运行中途额度不足 | 充值；限制工作流重试次数 |

## 相关文档

- [计费与额度](/docs/billing)
- [Embeddings](/docs/api-embeddings)
- [聊天补全](/docs/api-chat)
