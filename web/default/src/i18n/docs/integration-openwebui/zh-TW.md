Open WebUI 可將 NovaPuraAI 用作 OpenAI 相容後端。使用你的 API 金鑰與 Base URL 設定連線後，即可從 NovaPuraAI 回傳的列表中選擇模型。

## 前置條件

- 已安裝 Open WebUI（常用 Docker）
- 帶額度的 NovaPuraAI API 金鑰
- 來源站例如 `https://www.novapuraai.com`

## 管理端連線設定

在 Open WebUI 管理設定中，開啟 **Connections** / **OpenAI** 區段（標籤因版本而異）並新增：

| 欄位         | 值                              |
| ------------ | ------------------------------- |
| API Base URL | `https://www.novapuraai.com/v1` |
| API Key      | `sk-xxxxxxxx`                   |

儲存後重新整理模型。Open WebUI 會向你的閘道呼叫 `GET /v1/models` 與 `POST /v1/chat/completions`。

## Docker 環境範例

部分部署透過環境變數注入供應商設定：

```bash
OPENAI_API_BASE_URL=https://www.novapuraai.com/v1
OPENAI_API_KEY=sk-xxxxxxxx
```

確切變數名取決於你的 Open WebUI 版本——若無法使用介面，請以其文件為準。

## 選擇模型

- 僅顯示對你金鑰已啟用的模型。
- 若模型缺失，請用 curl 對 `/v1/models` 驗證。
- 多模態對話請選擇渠道支援視覺能力的模型；能力取決於渠道。

## 串流與工具

- 當模型/渠道支援時，可透過 OpenAI 相容路徑使用串流對話。
- 工具呼叫 / 函式功能需要同時滿足 Open WebUI 功能開關與模型對 tools 的支援。

## 疑難排解

| 問題                   | 檢查項                                    |
| ---------------------- | ----------------------------------------- |
| “Incorrect API key”    | 金鑰前綴、空白字元、權杖是否已停用        |
| 模型下拉為空           | Base URL 必須包含 `/v1`；網路須可達閘道   |
| 多使用者負載下出現 429 | 金鑰/分組速率限制；建立獨立金鑰或提高限額 |
| 首 token 很慢          | 上游延遲；嘗試其他模型                    |

## 相關文件

- [模型與路由](/docs/routing)
- [速率限制](/docs/rate-limits)
- [Chat Completions](/docs/api-chat)
