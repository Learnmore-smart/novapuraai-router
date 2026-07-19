NovaPuraAI предоставляет OpenAI-совместимый HTTP API. С действительным API-ключом и доступной квотой вы можете вызывать модели через один base URL.

## Что нужно

1. Учётная запись NovaPuraAI (например, на `https://www.novapuraai.com`).
2. API-ключ (`sk-...`) из **Console → API Keys**.
3. Баланс или квота для нужных моделей.

## Base URL

Для OpenAI-совместимых SDK задайте `base_url` / `baseURL` как origin сайта плюс `/v1`:

```text
https://www.novapuraai.com/v1
```

## Создание ключа

1. Войдите в консоль.
2. Откройте **API Keys** (tokens).
3. Создайте ключ. При необходимости ограничьте модели, задайте квоту и срок действия.
4. Скопируйте секрет один раз и сохраните в переменной окружения. Никогда не коммитьте его.

## Первый chat-запрос

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello from NovaPuraAI"}]
  }'
```

## Официальный OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)

resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(resp.choices[0].message.content)
```

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.NOVAPURA_API_KEY,
  baseURL: "https://www.novapuraai.com/v1",
});

const resp = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(resp.choices[0].message.content);
```

## Дальнейшие шаги

- [Аутентификация](/docs/authentication)
- [Первый запрос](/docs/first-request)
- [Базовый URL и эндпоинты](/docs/base-url)
