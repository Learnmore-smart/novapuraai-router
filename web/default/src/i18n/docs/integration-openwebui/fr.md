Open WebUI peut cibler des backends compatibles OpenAI.

> Les exemples de code et chemins d’API restent en anglais (identifiants techniques).

Open WebUI can target OpenAI-compatible backends.

## Connection

1. Admin panel → Connections / OpenAI.
2. API Base URL: `https://www.novapuraai.com/v1`
3. API Key: your NovaPuraAI `sk-` key.
4. Save and refresh models.

## Notes

- Disable competing providers if you only want NovaPuraAI routes.
- For streaming UIs, ensure reverse proxies buffer SSE correctly (`proxy_buffering off` on nginx when needed).
