Cursor では NovaPuraAI を OpenAI 互換プロバイダーとして設定できます。

> コード例と API パスは技術識別子のため英語のままです。

Cursor can use NovaPuraAI as an OpenAI-compatible provider.

## Setup

1. Open Cursor Settings → Models / OpenAI.
2. Enable a custom OpenAI base URL.
3. Set base URL to `https://www.novapuraai.com/v1`.
4. Paste your NovaPuraAI API key.
5. Choose a model id that exists on your gateway.

## Tips

- If chat works but tools fail, confirm the model supports tool calling upstream.
- For long agent sessions, watch wallet balance and rate limits.
- Keep a dedicated key for Cursor so you can revoke it without rotating production keys.
