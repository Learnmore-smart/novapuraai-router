NovaPuraAI là gateway API thống nhất. Client chỉ cần trỏ tới domain công khai; gateway định tuyến tới nhà cung cấp upstream.

> Ví dụ mã và đường dẫn API giữ nguyên tiếng Anh (định danh kỹ thuật).

NovaPuraAI serves a unified gateway. Clients point at your public origin; the gateway routes to upstream providers.

## Recommended base URL

| Client type | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI-compatible tools | `https://www.novapuraai.com/v1` |
| Raw HTTP (path already includes `/v1/...`) | `https://www.novapuraai.com` |

Replace the host with your deployment domain when self-hosting.

## Primary endpoints

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

## Authentication on every call

All of the paths above require:

```http
Authorization: Bearer sk-YOUR_KEY
```

## Health of the gateway

The admin console and public status endpoints report whether the site is ready. For production on Cloud Run, pair the container with Cloud SQL and Redis — do not rely on container-local SQLite.
