关于使用 NovaPuraAI API 网关的常见问题。超出 API 范围的产品界面帮助，请通过你所在部署的控制台支持渠道咨询。

## 什么是 NovaPuraAI？

NovaPuraAI 是 OpenAI 兼容的 API 网关（基于 new-api 的产品）。你使用 API 密钥向统一 Base URL 发送请求；网关完成鉴权、按模型路由、额度计费并记录用量。

## 文档在哪里？

官方开发者文档界面位于你部署站点上的 **`/docs`**（例如 `https://www.novapuraai.com/docs`）。

## 应该使用什么 Base URL？

- **源站**：`https://www.novapuraai.com`
- **OpenAI SDK 的 `base_url`**：`https://www.novapuraai.com/v1`

详见 [Base URL 与端点](/docs/base-url)。

## 如何获取 API 密钥？

登录 → **控制台 → API 密钥 / 令牌** → 创建密钥 → 复制 `sk-...` 机密。详情见 [身份验证](/docs/authentication)。

## 为什么出现 401 Unauthorized？

常见原因：缺少 `Authorization` 请求头、密钥被截断、令牌已禁用，或使用了 OpenAI Platform 密钥而非 NovaPuraAI 密钥。

## 为什么提示模型找不到？

模型目录因部署与分组而异。请调用 `GET /v1/models`，并使用响应中的 `id`。管理员的渠道配置也可能需要更新。

## 是否支持 Claude / Gemini 原生 API？

支持：

- Claude Messages：`POST /v1/messages`
- Gemini：`/v1beta/models/{model}:{action}`

对多提供商应用，OpenAI Chat Completions 仍是最常见路径。

## 计费如何计算？

按网关上配置的模型定价规则——对话/嵌入通常按 token，图像/音频则按模态相关单位。详见 [计费与额度](/docs/billing)，并以控制台用量日志为准。

## 额度用尽会发生什么？

请求会因额度不足类错误失败。在控制台充值或兑换额度后重试。密钥级限额可能在账户余额用尽之前就先耗尽。

## 速率限制和额度是一回事吗？

不是。**429** 表示需要减速；额度不足表示需要补充余额。详见 [速率限制](/docs/rate-limits)。

## 能否使用官方 OpenAI SDK？

可以——将 API 密钥设为你的 NovaPuraAI 密钥，并将 `base_url` / `baseURL` 设为 `{ORIGIN}/v1`。示例：[Python](/docs/sdk-python)、[Node.js](/docs/sdk-node)、[Go](/docs/sdk-go)、[curl](/docs/sdk-curl)。

## 是否支持流式输出？

在支持流式的模型/渠道上可以。在 Chat Completions 上使用 `"stream": true`，或使用 Claude/Gemini 协议原生的流式端点。

## 生产环境如何保护密钥？

将密钥保存在服务器或密钥库中，泄露后立即轮换，在可用时应用 IP 与模型白名单，并避免把机密嵌入移动端/网页客户端。

## 在哪里查看请求历史？

在控制台的 **用量日志**（以及启用时的仪表盘图表）。向管理员升级问题时请附上时间戳与错误体——切勿发送原始密钥。

## 仍然卡住？

1. 使用 [第一个请求](/docs/first-request) 中的 curl 复现。
2. 查看 [错误](/docs/api-errors)。
3. 在控制台确认模型、额度与速率限制。
4. 向部署管理员提供已脱敏的请求详情。
