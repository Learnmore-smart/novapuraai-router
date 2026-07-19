Cursor 可将 NovaPuraAI 用作 OpenAI 兼容的模型供应商。把 Cursor 指向部署源站的 `/v1` API，并用 NovaPuraAI API Key 鉴权即可。

## 你需要准备

- 在 **控制台 → API Keys / 令牌** 创建的 NovaPuraAI API Key
- 部署源站，例如 `https://www.novapuraai.com`
- 所选模型对应的足够额度

## 配置 OpenAI 兼容访问

Cursor 界面文案会随版本变化。请查找 **OpenAI API**、**OpenAI Compatible**、**Override OpenAI Base URL** 或 **Custom model provider** 等设置。

典型取值：

| 设置项 | 值 |
| --- | --- |
| API key | `sk-xxxxxxxx`（你的 NovaPuraAI 密钥） |
| Base URL | `https://www.novapuraai.com/v1` |
| Model | 来自 `GET /v1/models` 的模型 ID |

若字段要求填写“不含 `/v1` 的 OpenAI base URL”，可两种形式都试，并用极短 prompt 验证。可用形式几乎总是 **`{ORIGIN}/v1`**。

## 先在 Cursor 外验证

```bash
export NOVAPURA_BASE_URL="https://www.novapuraai.com"
export NOVAPURA_API_KEY="sk-xxxxxxxx"

curl "${NOVAPURA_BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${NOVAPURA_API_KEY}"
```

若此步骤失败，请先修复密钥/额度，再排查 IDE。

## 模型选择建议

- 使用 Agent 功能时，优先选择支持工具调用 / 长上下文的模型。
- 若 Cursor 提示 “model not found”，说明该 ID 对你的分组不可用——列出模型并选择可用 `id`。
- 在意成本时，自动补全类用途用更便宜模型，Agent 模式用更强模型。

## 故障排除

| 问题 | 处理 |
| --- | --- |
| Cursor 鉴权错误 | 重新粘贴完整密钥（含 `sk-`） |
| 网络 / 类似 CORS 失败 | Cursor 为桌面客户端，通常不是 CORS——检查 URL 拼写与 VPN |
| 空响应 | 确认额度，并用 curl 复现同一模型 |
| 触发限流 | 减少并行 Agent；见 [速率限制](/docs/rate-limits) |

## 安全

- 不要分享嵌入密钥的工作区配置文件。
- 设备丢失或密钥被提交到仓库后应立即轮换。

## 相关文档

- [鉴权](/docs/authentication)
- [模型列表](/docs/api-models)
- [聊天补全](/docs/api-chat)
