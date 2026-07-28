Dify 可将 NovaPuraAI 配置为自定义 OpenAI 兼容模型提供商。这样 Dify 中的应用、智能体与工作流即可使用经你网关路由的模型。

## 前置条件

- 具备添加模型提供商权限的 Dify 工作区（自托管或云端）
- NovaPuraAI API 密钥与额度
- 源站例如 `https://www.novapuraai.com`

## 添加 OpenAI-API-compatible 提供商

在 Dify **设置 → 模型提供商**（或 **Model Supplier**）中：

1. 选择 **OpenAI-API-compatible**（名称可能略有差异）。
2. 配置凭证：

| 字段             | 值                              |
| ---------------- | ------------------------------- |
| API Key          | `sk-xxxxxxxx`                   |
| API endpoint URL | `https://www.novapuraai.com/v1` |

3. 添加一个或多个模型，模型名须与 NovaPuraAI 返回的 **完全一致**（例如 `gpt-4o-mini`）。
4. 按模型类型配置上下文长度 / 模式（chat 与 completion）。
5. 保存，并在可用时运行 Dify 的连接测试。

## 端点期望

Dify 通常会请求：

- 对话模型：`POST /v1/chat/completions`
- 配置了嵌入模型时：`POST /v1/embeddings`
- 仅在提供商集成会做发现时：`GET /v1/models`

在排查 Dify 图之前，请先用直接 curl 调用确认。

## 智能体与工作流建议

- 为便宜与高阶的 NovaPuraAI 模型分别创建 Dify 模型条目。
- 在节点配置中设置合理的 max token 上限以控制成本。
- 工具 / 函数调用请选择渠道支持 tools 的模型。

## 常见失败

| 现象             | 可能原因                         |
| ---------------- | -------------------------------- |
| 校验失败         | 端点错误（缺少 `/v1`）或密钥错误 |
| 模型未找到       | 名称与 `GET /v1/models` 不一致   |
| 长链路超时       | 提高超时；减少串行 LLM 跳数      |
| 运行中途额度不足 | 充值余额；在工作流中限制重试次数 |

## 相关文档

- [计费与额度](/docs/billing)
- [嵌入](/docs/api-embeddings)
- [Chat Completions](/docs/api-chat)
