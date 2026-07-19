在對應通道啟用時，NovaPuraAI 會代理圖像、音訊與重排等媒體介面。

## 图像

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

## 語音轉寫

`POST /v1/audio/transcriptions` (multipart form)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## 語音合成

`POST /v1/audio/speech` returns audio bytes for supported TTS models.

## 重排

`POST /v1/rerank` accepts query + documents for Cohere/Jina-style rerankers when configured.

## 計費說明

Media endpoints often bill by image count, seconds, or document count — not only tokens. Check Model Square before bulk jobs.
