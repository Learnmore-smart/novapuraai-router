NovaPuraAI 是統一閘道。客戶端只需存取你的公開網域，由閘道路由至上游供應商。

## 建議 Base URL

| Client type | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI-compatible tools | `https://www.novapuraai.com/v1` |
| Raw HTTP (path already includes `/v1/...`) | `https://www.novapuraai.com` |

Replace the host with your deployment domain when self-hosting.

## 主要端點

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

## 每次呼叫都需要鑑權

All of the paths above require:

```http
Authorization: Bearer sk-YOUR_KEY
```

## 閘道健康

The admin console and public status endpoints report whether the site is ready. For production on Cloud Run, pair the container with Cloud SQL and Redis — do not rely on container-local SQLite.
