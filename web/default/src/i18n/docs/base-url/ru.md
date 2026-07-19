NovaPuraAI предоставляет единый шлюз. Клиенты обращаются к публичному origin; шлюз маршрутизирует запросы к upstream-провайдерам.

## Рекомендуемый base URL

| Тип клиента | Base URL |
| --- | --- |
| OpenAI SDK / OpenAI-совместимые инструменты | `https://www.novapuraai.com/v1` |
| «Сырой» HTTP (путь уже содержит `/v1/...`) | `https://www.novapuraai.com` |

## Основные эндпоинты

| Метод | Путь | Назначение |
| --- | --- | --- |
| POST | `/v1/chat/completions` | Чат (OpenAI) |
| POST | `/v1/completions` | Текстовые completions |
| POST | `/v1/responses` | OpenAI Responses API |
| POST | `/v1/messages` | Anthropic Messages |
| POST | `/v1/embeddings` | Эмбеддинги |
| POST | `/v1/images/generations` | Генерация изображений |
| POST | `/v1/audio/transcriptions` | Речь в текст |
| POST | `/v1/audio/speech` | Текст в речь |
| POST | `/v1/rerank` | Ранжирование |
| GET | `/v1/models` | Список моделей |
| POST | `/v1beta/models/{model}:generateContent` | Стиль Gemini |

Маршруты Midjourney и другие task-маршруты также могут быть доступны в зависимости от конфигурации администратора.

## Аутентификация при каждом вызове

Все перечисленные пути требуют:

```http
Authorization: Bearer sk-YOUR_KEY
```

## Состояние шлюза

Консоль администратора и публичные status-эндпоинты показывают, готов ли сайт. В production обеспечивайте высокую доступность шлюза и базы данных; не полагайтесь на эфемерное локальное хранилище для данных, критичных для биллинга.
