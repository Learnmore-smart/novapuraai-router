Cursor 可将 NovaPuraAI 用作 OpenAI 兼容的模型提供商。将 Cursor 指向部署源站的 `/v1` API，并使用 NovaPuraAI API 密钥完成鉴权。

## 你需要准备

- 来自 **控制台 → API 密钥 / 令牌** 的 NovaPuraAI API 密钥
- 你的部署源站，例如 `https://www.novapuraai.com`
- 在 Cursor 中所选模型所需的足够额度

## 配置 OpenAI 兼容访问

Cursor 的界面文案会随版本变化。请查找 **OpenAI API**、**OpenAI Compatible**、**Override OpenAI Base URL** 或 **Custom model provider** 等相关设置。

典型取值：

| 设置项   | 值                                    |
| -------- | ------------------------------------- |
| API key  | `sk-xxxxxxxx`（你的 NovaPuraAI 密钥） |
| Base URL | `https://www.novapuraai.com/v1`       |
| Model    | 来自 `GET /v1/models` 的模型 ID       |

若某字段要求填写不含 `/v1` 的 “OpenAI base URL”，可两种形式都试一次，并用简短提示确认。可用形式几乎总是 **`{ORIGIN}/v1`**。

## 先在 Cursor 外验证

```bash
export NOVAPURA_BASE_URL="https://www.novapuraai.com"
export NOVAPURA_API_KEY="sk-xxxxxxxx"

curl "${NOVAPURA_BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${NOVAPURA_API_KEY}"
```

若此处失败，请先修复密钥/额度，再排查 IDE。

## 模型选择建议

- 若使用代理功能，优先选择支持工具调用 / 长上下文的模型。
- 若 Cursor 显示 “model not found”，说明该 ID 未对你的分组启用——请列出模型并选择可用的 `id`。
- 在成本敏感场景下，自动补全类用法可用更便宜的模型，代理模式再用更强模型。

## 排障

| 问题                    | 处理                                                       |
| ----------------------- | ---------------------------------------------------------- |
| Cursor 中鉴权错误       | 重新粘贴完整密钥，包含 `sk-`                               |
| 网络 / 类似 CORS 的失败 | Cursor 为桌面客户端，通常与 CORS 无关——检查 URL 拼写与 VPN |
| 空响应                  | 确认额度，并用 curl 以相同模型复测                         |
| 速率限制                | 减少并行代理运行；见 [速率限制](/docs/rate-limits)         |

## 安全

- 不要共享嵌入了机密的工作区设置文件。
- 机器遗失或密钥被提交到仓库后，请立即轮换密钥。

## 相关文档

- [身份验证](/docs/authentication)
- [模型列表](/docs/api-models)
- [Chat Completions](/docs/api-chat)
