關於使用 NovaPuraAI API 閘道的常見問題。超出 API 範圍的產品介面說明，請透過你所在部署的主控台支援管道諮詢。

## 什麼是 NovaPuraAI？

NovaPuraAI 是 OpenAI 相容的 API 閘道（基於 new-api 的產品）。你使用 API 金鑰向統一 Base URL 傳送請求；閘道完成驗證、依模型路由、額度計費並記錄用量。

## 文件在哪裡？

官方開發者文件介面位於你部署站點上的 **`/docs`**（例如 `https://www.novapuraai.com/docs`）。

## 應該使用什麼 Base URL？

- **來源站**：`https://www.novapuraai.com`
- **OpenAI SDK 的 `base_url`**：`https://www.novapuraai.com/v1`

詳見 [Base URL 與端點](/docs/base-url)。

## 如何取得 API 金鑰？

登入 → **主控台 → API 金鑰 / 權杖** → 建立金鑰 → 複製 `sk-...` 機密。詳情見 [身份驗證](/docs/authentication)。

## 為什麼出現 401 Unauthorized？

常見原因：缺少 `Authorization` 請求標頭、金鑰被截斷、權杖已停用，或使用了 OpenAI Platform 金鑰而非 NovaPuraAI 金鑰。

## 為什麼提示模型找不到？

模型目錄因部署與分組而異。請呼叫 `GET /v1/models`，並使用回應中的 `id`。管理員的渠道設定也可能需要更新。

## 是否支援 Claude / Gemini 原生 API？

支援：

- Claude Messages：`POST /v1/messages`
- Gemini：`/v1beta/models/{model}:{action}`

對多供應商應用，OpenAI Chat Completions 仍是最常見路徑。

## 計費如何計算？

依閘道上設定的模型定價規則——對話/嵌入通常依 token，影像/音訊則依模態相關單位。詳見 [計費與額度](/docs/billing)，並以主控台用量日誌為準。

## 額度用盡會發生什麼？

請求會因額度不足類錯誤失敗。在主控台儲值或兌換額度後重試。金鑰級限額可能在帳戶餘額用盡之前就先耗盡。

## 速率限制和額度是一回事嗎？

不是。**429** 表示需要減速；額度不足表示需要補充餘額。詳見 [速率限制](/docs/rate-limits)。

## 能否使用官方 OpenAI SDK？

可以——將 API 金鑰設為你的 NovaPuraAI 金鑰，並將 `base_url` / `baseURL` 設為 `{ORIGIN}/v1`。範例：[Python](/docs/sdk-python)、[Node.js](/docs/sdk-node)、[Go](/docs/sdk-go)、[curl](/docs/sdk-curl)。

## 是否支援串流輸出？

在支援串流的模型/渠道上可以。在 Chat Completions 上使用 `"stream": true`，或使用 Claude/Gemini 協定原生的串流端點。

## 正式環境如何保護金鑰？

將金鑰保存在伺服器或密鑰庫中，外洩後立即輪替，在可用時套用 IP 與模型允許清單，並避免把機密嵌入行動端/網頁用戶端。

## 在哪裡查看請求歷史？

在主控台的 **用量日誌**（以及啟用時的儀表板圖表）。向管理員升級問題時請附上時間戳與錯誤主體——切勿傳送原始金鑰。

## 仍然卡住？

1. 使用 [第一個請求](/docs/first-request) 中的 curl 重現。
2. 查看 [錯誤](/docs/api-errors)。
3. 在主控台確認模型、額度與速率限制。
4. 向部署管理員提供已脫敏的請求詳情。
