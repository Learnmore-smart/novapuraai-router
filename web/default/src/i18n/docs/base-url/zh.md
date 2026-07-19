NovaPuraAI 是统一网关。客户端只访问你的公网域名，由网关路由到上游供应商。

## 推荐 Base URL

| Client type | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI-compatible tools | `https://www.novapuraai.com/v1` |
| Raw HTTP (path already includes `/v1/...`) | `https://www.novapuraai.com` |

## 主要端点

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/v1/chat/completions` | Chat (OpenAI) |
| POST | `/v1/completions` | Text completions |
| POST | `/v1/responses` | OpenAI Responses API |
| POST | `/v1/messages` | Anthropic Messages |
| POST | `/v1/embeddings` | Embeddings |
| POST | `/v1/images/generations` | Image generation |
| POST | `/v1/audio/transcriptions` | Speech-to-text |
| POST | `/v1/audio/speech` | Text-to-speech |
| POST | `/v1/rerank` | Rerank |
| GET | `/v1/models` | List models |
| POST | `/v1beta/models/{model}:generateContent` | Gemini-style |

Midjourney and other task routes may also be available depending on admin configuration.

## 每次调用都需要鉴权

All of the paths above require:

```http
Authorization: Bearer sk-YOUR_KEY
```

## 网关健康

The admin console and public status endpoints report whether the site is ready. For production on Cloud Run, pair the container with Cloud SQL and Redis — do not rely on container-local SQLite.
