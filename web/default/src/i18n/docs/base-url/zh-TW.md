NovaPuraAI 提供統一閘道。用戶端指向公開來源站；閘道再路由到上游供應商。

## 建議 Base URL

| 用戶端類型                        | Base URL                        |
| --------------------------------- | ------------------------------- |
| OpenAI SDK / OpenAI 相容工具      | `https://www.novapuraai.com/v1` |
| 原始 HTTP（路徑已包含 `/v1/...`） | `https://www.novapuraai.com`    |

## 主要端點

| 方法 | 路徑                                     | 用途                 |
| ---- | ---------------------------------------- | -------------------- |
| POST | `/v1/chat/completions`                   | 對話（OpenAI）       |
| POST | `/v1/completions`                        | 文字補全             |
| POST | `/v1/responses`                          | OpenAI Responses API |
| POST | `/v1/messages`                           | Anthropic Messages   |
| POST | `/v1/embeddings`                         | 嵌入向量             |
| POST | `/v1/images/generations`                 | 影像產生             |
| POST | `/v1/audio/transcriptions`               | 語音轉文字           |
| POST | `/v1/audio/speech`                       | 文字轉語音           |
| POST | `/v1/rerank`                             | 重排序               |
| GET  | `/v1/models`                             | 列出模型             |
| POST | `/v1beta/models/{model}:generateContent` | Gemini 風格          |

依管理員設定，Midjourney 及其他任務路由也可能可用。

## 每次呼叫都需要驗證

上述所有路徑都需要：

```http
Authorization: Bearer sk-YOUR_KEY
```

## 閘道健康狀態

管理主控台與公開狀態端點會回報站點是否就緒。正式環境中請保持閘道與資料庫高可用；計費關鍵資料不要依賴臨時本機儲存。
