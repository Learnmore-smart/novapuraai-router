Cursor có thể dùng NovaPuraAI như nhà cung cấp tương thích OpenAI.

> Ví dụ mã và đường dẫn API giữ nguyên tiếng Anh (định danh kỹ thuật).

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
