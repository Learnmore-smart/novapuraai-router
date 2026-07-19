`POST /v1/messages` 接受 Anthropic Messages 風格的負載，用於閘道上已設定的 Claude 相容模型。

## 範例

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Explain prepaid API billing briefly."}
    ]
  }'
```

## 說明

- 模型名稱必須存在於你的 NovaPuraAI 目錄中。
- 部分僅 Anthropic 使用的請求標頭在相關時會被接受並轉發。
- 若渠道轉接器支援，通常也可透過 OpenAI 對話格式呼叫同一 Claude 模型——請優先使用你的 SDK 期望的格式。

## 錯誤

無效 schema 或不支援的欄位會回傳帶 JSON 錯誤主體的 `4xx`。請確認 Messages API 要求時已提供 `max_tokens`。
