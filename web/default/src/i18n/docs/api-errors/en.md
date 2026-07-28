Errors are returned as JSON with an HTTP status code. Message text may be localized or provider-specific.

## Common status codes

| Code            | Meaning                                    |
| --------------- | ------------------------------------------ |
| 400             | Invalid request body or parameters         |
| 401             | Missing or invalid API key                 |
| 403             | Not allowed (model, module, or permission) |
| 404             | Unknown route or model                     |
| 429             | Rate limited                               |
| 500 / 502 / 503 | Gateway or upstream failure                |

## Example error body

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

Some endpoints use `{ "success": false, "message": "..." }` for console APIs. Relay routes prefer OpenAI-style error objects.

## Debugging checklist

1. Log the request id if the response or console logs expose one.
2. Retry idempotent GETs; be careful with POST retries.
3. Compare working curl from the dashboard “First API request” card.
4. Verify channel health with your administrator if only some models fail.
