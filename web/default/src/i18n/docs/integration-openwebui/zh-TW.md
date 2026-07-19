Open WebUI 可對接 OpenAI 相容後端。

## 連線

1. Admin panel → Connections / OpenAI.
2. API Base URL: `https://www.novapuraai.com/v1`
3. API Key: your NovaPuraAI `sk-` key.
4. Save and refresh models.

## 說明

- Disable competing providers if you only want NovaPuraAI routes.
- For streaming UIs, ensure reverse proxies buffer SSE correctly (`proxy_buffering off` on nginx when needed).
