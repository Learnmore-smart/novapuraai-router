當管理員啟用 Gemini 渠道後，可透過 `/v1beta` 下的 Google 風格路徑存取 Gemini 相容流量。

## 產生內容

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
```

## 提示

- 確切的模型 ID 取決於你部署中的渠道設定。
- 多模態部分（內嵌資料 / 檔案 URI）遵循 Gemini 請求結構；請將負載控制在閘道請求主體限制內。
- 若渠道轉接器已對應，也可透過 OpenAI 相容的對話介面存取部分 Gemini 模型。

## 驗證

使用相同的 NovaPuraAI `sk-` 金鑰。除非你是正在設定上游渠道的管理員，否則不要向 NovaPuraAI 傳送 Google AI Studio 金鑰。
