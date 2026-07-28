Open WebUI 可将 NovaPuraAI 用作 OpenAI 兼容后端。使用你的 API 密钥与 Base URL 配置连接后，即可从 NovaPuraAI 返回的列表中选择模型。

## 前置条件

- 已安装 Open WebUI（常用 Docker）
- 带额度的 NovaPuraAI API 密钥
- 源站例如 `https://www.novapuraai.com`

## 管理端连接设置

在 Open WebUI 管理设置中，打开 **Connections** / **OpenAI** 部分（标签因版本而异）并添加：

| 字段         | 值                              |
| ------------ | ------------------------------- |
| API Base URL | `https://www.novapuraai.com/v1` |
| API Key      | `sk-xxxxxxxx`                   |

保存后刷新模型。Open WebUI 会向你的网关调用 `GET /v1/models` 与 `POST /v1/chat/completions`。

## Docker 环境示例

部分部署通过环境变量注入提供商设置：

```bash
OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
OPENAI_API_KEY=sk-xxxxxxxx
```

确切变量名取决于你的 Open WebUI 版本——若无法使用界面，请以其文档为准。

## 选择模型

- 仅显示对你密钥已启用的模型。
- 若模型缺失，请用 curl 对 `/v1/models` 验证。
- 多模态对话请选择渠道支持视觉能力的模型；能力取决于渠道。

## 流式与工具

- 当模型/渠道支持时，可通过 OpenAI 兼容路径使用流式对话。
- 工具调用 / 函数功能需要同时满足 Open WebUI 功能开关与模型对 tools 的支持。

## 排障

| 问题                 | 检查项                                    |
| -------------------- | ----------------------------------------- |
| “Incorrect API key”  | 密钥前缀、空白字符、令牌是否已禁用        |
| 模型下拉为空         | Base URL 必须包含 `/v1`；网络须可达网关   |
| 多用户负载下出现 429 | 密钥/分组速率限制；创建独立密钥或提高限额 |
| 首 token 很慢        | 上游延迟；尝试其他模型                    |

## 相关文档

- [模型与路由](/docs/routing)
- [速率限制](/docs/rate-limits)
- [Chat Completions](/docs/api-chat)
