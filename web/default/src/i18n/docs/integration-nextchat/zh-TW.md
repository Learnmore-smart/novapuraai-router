NextChat（ChatGPT-Next-Web 及相容分支）可透過 OpenAI 相容設定連線 NovaPuraAI。一次設定 Base URL、API 金鑰與預設模型後，即可依往常使用 Web UI。

## 前置條件

- 正在執行的 NextChat（自託管或本機）
- NovaPuraAI API 金鑰
- 部署來源站例如 `https://www.novapuraai.com`

## 需要設定的選項

在 NextChat **設定** 中（文案可能因分支/版本而異）：

| 欄位 | 建議值 |
| --- | --- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key | `sk-xxxxxxxx` |
| Model | 來自 `GET /v1/models` 的 ID |

若介面只儲存來源站並自行附加 `/v1`，則使用不含重複 `/v1` 的 `https://www.novapuraai.com`。不確定時，開啟瀏覽器網路工具，確認最終路徑為 `/v1/chat/completions`。

## 環境變數風格部署

許多 NextChat Docker 映像接受：

```bash
OPENAI_API_KEY=sk-xxxxxxxx
BASE_URL=https://www.novapuraai.com/v1
# some images use OPENAI_API_BASE / OPENAI_BASE_URL — check your image docs
```

修改環境變數後請重新啟動容器。

## 煙霧測試

```bash
curl "https://www.novapuraai.com/v1/chat/completions" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

## 常見問題

| 現象 | 原因 | 處理 |
| --- | --- | --- |
| 對話回傳 404 | Base URL 錯誤（缺少或重複了 `/v1`） | 對照網路面板中的最終路徑 |
| 401 | 未傳入金鑰或金鑰錯誤 | 貼上 NovaPuraAI 金鑰，而非 OpenAI 平台金鑰 |
| 模型列表為空 | 前端無法呼叫 `/v1/models` | 檢查 CORS/代理與金鑰權限 |
| 餘額錯誤 | 無額度 | 在 NovaPuraAI 主控台儲值 |

## 安全說明

- 若 NextChat 組建支援伺服器端代理模式，優先用其隱藏瀏覽器中的金鑰。
- 公開展示請使用低額度金鑰並嚴格限制模型允許清單。

## 相關文件

- [Base URL 與端點](/docs/base-url)
- [身份驗證](/docs/authentication)
- [常見問題](/docs/faq)
