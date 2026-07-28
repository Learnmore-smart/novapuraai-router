Dify 可將 NovaPuraAI 設定為自訂 OpenAI 相容模型供應商。這樣 Dify 中的應用、智慧代理與工作流程即可使用經你閘道路由的模型。

## 前置條件

- 具備新增模型供應商權限的 Dify 工作區（自託管或雲端）
- NovaPuraAI API 金鑰與額度
- 來源站例如 `https://www.novapuraai.com`

## 新增 OpenAI-API-compatible 供應商

在 Dify **設定 → 模型供應商**（或 **Model Supplier**）中：

1. 選擇 **OpenAI-API-compatible**（名稱可能略有差異）。
2. 設定憑證：

| 欄位             | 值                              |
| ---------------- | ------------------------------- |
| API Key          | `sk-xxxxxxxx`                   |
| API endpoint URL | `https://www.novapuraai.com/v1` |

3. 新增一個或多個模型，模型名須與 NovaPuraAI 回傳的 **完全一致**（例如 `gpt-4o-mini`）。
4. 依模型類型設定上下文長度 / 模式（chat 與 completion）。
5. 儲存，並在可用時執行 Dify 的連線測試。

## 端點期望

Dify 通常會請求：

- 對話模型：`POST /v1/chat/completions`
- 設定了嵌入模型時：`POST /v1/embeddings`
- 僅在供應商整合會做探索時：`GET /v1/models`

在排查 Dify 圖之前，請先用直接 curl 呼叫確認。

## 智慧代理與工作流程建議

- 為便宜與高階的 NovaPuraAI 模型分別建立 Dify 模型項目。
- 在節點設定中設定合理的 max token 上限以控制成本。
- 工具 / 函式呼叫請選擇渠道支援 tools 的模型。

## 常見失敗

| 現象             | 可能原因                           |
| ---------------- | ---------------------------------- |
| 驗證失敗         | 端點錯誤（缺少 `/v1`）或金鑰錯誤   |
| 模型未找到       | 名稱與 `GET /v1/models` 不一致     |
| 長鏈路逾時       | 提高逾時；減少序列 LLM 跳數        |
| 執行中途額度不足 | 儲值餘額；在工作流程中限制重試次數 |

## 相關文件

- [計費與額度](/docs/billing)
- [嵌入](/docs/api-embeddings)
- [Chat Completions](/docs/api-chat)
