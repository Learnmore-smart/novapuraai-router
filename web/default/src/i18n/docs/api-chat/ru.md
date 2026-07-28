`POST /v1/chat/completions` — основной OpenAI-совместимый эндпоинт чата.

## Запрос

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are helpful."},
      {"role": "user", "content": "Summarize NovaPuraAI in one sentence."}
    ],
    "temperature": 0.5,
    "max_tokens": 256
  }'
```

## Важные поля

| Поле                                   | Описание                                                   |
| -------------------------------------- | ---------------------------------------------------------- |
| `model`                                | Обязательно. Должна быть включена для вашей учётной записи |
| `messages`                             | Массив сообщений в формате OpenAI chat                     |
| `stream`                               | `true` для потоковой передачи токенов по SSE               |
| `temperature` / `top_p`                | Параметры сэмплирования                                    |
| `max_tokens` / `max_completion_tokens` | Ограничения длины ответа (зависят от провайдера)           |
| `tools` / `tool_choice`                | Function calling, если upstream-модель поддерживает его    |

## Потоковая передача

Укажите `"stream": true`. Ответ — `text/event-stream` с фрагментами `data: {...}`, завершающийся `data: [DONE]`.

## Совместимость

Большинство инструментов, принимающих пользовательский base URL OpenAI, работают без изменений. Укажите `https://www.novapuraai.com/v1` и используйте ключ NovaPuraAI.
