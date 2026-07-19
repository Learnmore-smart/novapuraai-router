Gemini-совместимый трафик доступен через пути в стиле Google под `/v1beta`, когда администраторы включают каналы Gemini.

## Генерация контента

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

## Советы

- Точные идентификаторы моделей зависят от конфигурации каналов на вашей стороне.
- Мультимодальные части (inline data / file URI) следуют формату запросов Gemini; соблюдайте лимиты размера тела запроса шлюза.
- Некоторые модели Gemini также доступны через OpenAI-совместимый чат, если адаптер канала их сопоставляет.

## Аутентификация

Используйте тот же ключ NovaPuraAI с префиксом `sk-`. Не отправляйте ключи Google AI Studio в NovaPuraAI, если вы не администратор, настраивающий upstream-каналы.
