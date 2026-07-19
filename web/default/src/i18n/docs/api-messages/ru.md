`POST /v1/messages` принимает полезную нагрузку в стиле Anthropic Messages для Claude-совместимых моделей, настроенных на шлюзе.

## Пример

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

## Примечания

- Имена моделей должны присутствовать в каталоге NovaPuraAI.
- Некоторые заголовки, специфичные для Anthropic, принимаются и пересылаются при необходимости.
- Те же модели Claude часто можно вызывать через формат OpenAI chat, если адаптер канала это поддерживает — выбирайте формат, ожидаемый вашим SDK.

## Ошибки

Неверная схема или неподдерживаемые поля возвращают `4xx` с JSON-телом ошибки. Убедитесь, что `max_tokens` указан, когда этого требует Messages API.
