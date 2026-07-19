NextChat поддерживает пользовательские OpenAI endpoints.

> Примеры кода и пути API сохранены на английском (технические идентификаторы).

NextChat (ChatGPT-Next-Web) supports custom OpenAI endpoints.

## Configuration

Set environment variables (names vary slightly by fork):

```bash
OPENAI_API_KEY=sk-YOUR_KEY
BASE_URL=https://www.novapuraai.com/v1
```

Or configure the same values in the app’s UI if self-hosting with a config panel.

## Model list

Ensure the models you select in NextChat exist on NovaPuraAI. Prefer models returned by `GET /v1/models`.
