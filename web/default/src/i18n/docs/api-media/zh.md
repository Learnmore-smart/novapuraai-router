在对应渠道启用时，NovaPuraAI 会代理图像、音频与重排等媒体接口。

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

## 语音转写

`POST /v1/audio/transcriptions` (multipart form)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## 语音合成

`POST /v1/audio/speech` returns audio bytes for supported TTS models.

## 重排

`POST /v1/rerank` accepts query + documents for Cohere/Jina-style rerankers when configured.

## 计费说明

Media endpoints often bill by image count, seconds, or document count — not only tokens. Check Model Square before bulk jobs.
