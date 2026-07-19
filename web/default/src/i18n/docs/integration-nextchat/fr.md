NextChat prend en charge les endpoints OpenAI personnalisés.

> Les exemples de code et chemins d’API restent en anglais (identifiants techniques).

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
