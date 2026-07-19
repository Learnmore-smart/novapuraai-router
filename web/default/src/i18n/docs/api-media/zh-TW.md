當對應渠道已啟用時，NovaPuraAI 會代理選定的媒體端點。

## 影像

`POST /v1/images/generations`

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A minimal logo for an API platform",
    "size": "1024x1024"
  }'
```

## 音訊轉寫

`POST /v1/audio/transcriptions`（multipart 表單）

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## 語音合成

`POST /v1/audio/speech` 對受支援的 TTS 模型回傳音訊位元組。

## 重排序（Rerank）

`POST /v1/rerank` 在已設定時接受 query 與 documents，用於 Cohere / Jina 風格的重排序器。

## 計費說明

媒體端點通常依影像數量、秒數或文件數量計費，而不僅是 token。大量任務前請先查看模型廣場。
