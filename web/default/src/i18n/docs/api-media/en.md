NovaPuraAI proxies selected media endpoints when corresponding channels are enabled.

## Images

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

## Audio transcription

`POST /v1/audio/transcriptions` (multipart form)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## Speech synthesis

`POST /v1/audio/speech` returns audio bytes for supported TTS models.

## Rerank

`POST /v1/rerank` accepts query + documents for Cohere/Jina-style rerankers when configured.

## Billing note

Media endpoints often bill by image count, seconds, or document count — not only tokens. Check Model Square before bulk jobs.
