Open WebUI 可将 NovaPuraAI 用作 OpenAI 兼容后端。使用 API Key 与 Base URL 配置连接后，即可从 NovaPuraAI 返回的列表中选择模型。

## 前置条件

- 已安装 Open WebUI（常见为 Docker）
- 具备额度的 NovaPuraAI API Key
- 源站例如 `https://www.novapuraai.com`

## 管理端连接设置

在 Open WebUI 管理设置中打开 **Connections** / **OpenAI** 区域（文案因版本而异），添加：

| 字段 | 值 |
| --- | --- |
| API Base URL | `https://www.novapuraai.com/v1` |
| API Key | `sk-xxxxxxxx` |

保存后刷新模型列表。Open WebUI 会对你的网关调用 `GET /v1/models` 与 `POST /v1/chat/completions`。

## Docker 环境变量示例

部分部署通过环境变量注入供应商设置：

```bash
OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
OPENAI_API_KEY=sk-xxxxxxxx
```

确切变量名取决于 Open WebUI 版本——若无法使用 UI，请对照其文档确认。

## 选择模型

- 仅显示对你密钥启用的模型。
- 若缺少模型，请用 curl 请求 `/v1/models` 验证。
- 多模态聊天请选择通道支持视觉能力的模型；能力因通道而异。

## 流式与工具

- 当模型/通道支持时，流式聊天走 OpenAI 兼容路径即可。
- 工具调用 / function 功能需要 Open WebUI 功能开关与支持 tools 的模型同时满足。

## 故障排除

| 问题 | 检查项 |
| --- | --- |
| “Incorrect API key” | 密钥前缀、空格、是否禁用 |
| 模型下拉为空 | Base URL 需包含 `/v1`；网络需可达网关 |
| 多用户负载时 429 | 密钥/分组限流；拆分密钥或提高限额 |
| 首 token 慢 | 上游延迟；可尝试其他模型 |

## 相关文档

- [模型与路由](/docs/routing)
- [速率限制](/docs/rate-limits)
- [聊天补全](/docs/api-chat)
