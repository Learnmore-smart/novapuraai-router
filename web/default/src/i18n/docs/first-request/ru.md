На этой странице — полный первый успешный вызов и как читать ответ.

## Чек-лист

- [ ] У вас есть ключ, начинающийся с `sk-`
- [ ] На учётной записи положительный баланс / квота
- [ ] Известно хотя бы одно имя включённой модели (см. **Model Square** или `GET /v1/models`)

## curl

```bash
export NOVAPURA_API_KEY=sk-YOUR_KEY
export NOVAPURA_BASE=https://www.novapuraai.com

curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "system", "content": "You are a concise assistant."},
      {"role": "user", "content": "Say hello in one sentence."}
    ],
    "temperature": 0.7
  }'
```

## Успешный ответ (форма)

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "Hello!"},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
```

## Потоковая передача

Добавьте `"stream": true` и читайте Server-Sent Events:

```bash
curl "$NOVAPURA_BASE/v1/chat/completions" \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -N \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to five."}]
  }'
```

## Устранение неполадок

1. Убедитесь, что имя модели точно совпадает с включённой моделью.
2. Убедитесь, что base URL для OpenAI SDK включает `/v1`.
3. Убедитесь, что используется HTTPS и что reverse proxy / CDN допускает длительные потоки, если вы стримите.
