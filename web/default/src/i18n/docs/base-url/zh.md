NovaPuraAI 提供统一网关。客户端指向公开源站；网关再路由到上游提供商。

## 推荐 Base URL

| 客户端类型 | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI 兼容工具 | `https://www.novapuraai.com/v1` |
| 原始 HTTP（路径已包含 `/v1/...`） | `https://www.novapuraai.com` |

## 主要端点

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `/v1/chat/completions` | 对话（OpenAI） |
| POST | `/v1/completions` | 文本补全 |
| POST | `/v1/responses` | OpenAI Responses API |
| POST | `/v1/messages` | Anthropic Messages |
| POST | `/v1/embeddings` | 嵌入向量 |
| POST | `/v1/images/generations` | 图像生成 |
| POST | `/v1/audio/transcriptions` | 语音转文字 |
| POST | `/v1/audio/speech` | 文字转语音 |
| POST | `/v1/rerank` | 重排序 |
| GET | `/v1/models` | 列出模型 |
| POST | `/v1beta/models/{model}:generateContent` | Gemini 风格 |

根据管理员配置，Midjourney 及其他任务路由也可能可用。

## 每次调用都需要鉴权

上述所有路径都需要：

```http
Authorization: Bearer sk-YOUR_KEY
```

## 网关健康状态

管理控制台与公开状态端点会报告站点是否就绪。生产环境中请保持网关与数据库高可用；计费关键数据不要依赖临时本地存储。
