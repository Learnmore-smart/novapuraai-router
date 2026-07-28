Cursor 可將 NovaPuraAI 用作 OpenAI 相容的模型供應商。將 Cursor 指向部署來源站的 `/v1` API，並使用 NovaPuraAI API 金鑰完成驗證。

## 你需要準備

- 來自 **主控台 → API 金鑰 / 權杖** 的 NovaPuraAI API 金鑰
- 你的部署來源站，例如 `https://www.novapuraai.com`
- 在 Cursor 中所選模型所需的足夠額度

## 設定 OpenAI 相容存取

Cursor 的介面文案會隨版本變化。請尋找 **OpenAI API**、**OpenAI Compatible**、**Override OpenAI Base URL** 或 **Custom model provider** 等相關設定。

典型取值：

| 設定項   | 值                                    |
| -------- | ------------------------------------- |
| API key  | `sk-xxxxxxxx`（你的 NovaPuraAI 金鑰） |
| Base URL | `https://www.novapuraai.com/v1`       |
| Model    | 來自 `GET /v1/models` 的模型 ID       |

若某欄位要求填寫不含 `/v1` 的 “OpenAI base URL”，可兩種形式都試一次，並用簡短提示確認。可用形式幾乎總是 **`{ORIGIN}/v1`**。

## 先在 Cursor 外驗證

```bash
export NOVAPURA_BASE_URL="https://www.novapuraai.com"
export NOVAPURA_API_KEY="sk-xxxxxxxx"

curl "${NOVAPURA_BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${NOVAPURA_API_KEY}"
```

若此處失敗，請先修復金鑰/額度，再排查 IDE。

## 模型選擇建議

- 若使用代理功能，優先選擇支援工具呼叫 / 長上下文的模型。
- 若 Cursor 顯示 “model not found”，表示該 ID 未對你的分組啟用——請列出模型並選擇可用的 `id`。
- 在成本敏感場景下，自動補全類用法可用更便宜的模型，代理模式再用更強模型。

## 疑難排解

| 問題                    | 處理                                                       |
| ----------------------- | ---------------------------------------------------------- |
| Cursor 中驗證錯誤       | 重新貼上完整金鑰，包含 `sk-`                               |
| 網路 / 類似 CORS 的失敗 | Cursor 為桌面用戶端，通常與 CORS 無關——檢查 URL 拼字與 VPN |
| 空回應                  | 確認額度，並用 curl 以相同模型複測                         |
| 速率限制                | 減少並行代理執行；見 [速率限制](/docs/rate-limits)         |

## 安全

- 不要共用嵌入了機密的工作區設定檔。
- 機器遺失或金鑰被提交到儲存庫後，請立即輪替金鑰。

## 相關文件

- [身份驗證](/docs/authentication)
- [模型列表](/docs/api-models)
- [Chat Completions](/docs/api-chat)
