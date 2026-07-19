Cursor 可將 NovaPuraAI 設定為 OpenAI 相容供應商。

## 設定

1. Open Cursor Settings → Models / OpenAI.
2. Enable a custom OpenAI base URL.
3. Set base URL to `https://www.novapuraai.com/v1`.
4. Paste your NovaPuraAI API key.
5. Choose a model id that exists on your gateway.

## 提示

- If chat works but tools fail, confirm the model supports tool calling upstream.
- For long agent sessions, watch wallet balance and rate limits.
- Keep a dedicated key for Cursor so you can revoke it without rotating production keys.
