NovaPuraAI проксирует выбранные медиа-эндпоинты, когда соответствующие каналы включены.

## Изображения

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

## Транскрипция аудио

`POST /v1/audio/transcriptions` (multipart form)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## Синтез речи

`POST /v1/audio/speech` возвращает аудиоданные для поддерживаемых TTS-моделей.

## Ранжирование (Rerank)

`POST /v1/rerank` принимает запрос и документы для reranker в стиле Cohere/Jina, если они настроены.

## Примечание по биллингу

Медиа-эндпоинты часто тарифицируются по числу изображений, секундам или количеству документов — не только по токенам. Перед массовыми заданиями проверьте Model Square.
