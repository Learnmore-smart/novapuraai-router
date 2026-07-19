`POST /v1/messages` accepts Anthropic Messages-style payloads for Claude-compatible models configured on the gateway.

## Example

```bash
curl https://www.novapuraai.com/v1/messages \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Explain prepaid API billing briefly."}
    ]
  }'
```

## Notes

- Model names must exist in your NovaPuraAI catalog.
- Some Anthropic-only headers are accepted and forwarded when relevant.
- You can often call the same Claude models via OpenAI chat format if the channel adapter supports it — prefer the format your SDK expects.

## Errors

Invalid schema or unsupported fields return `4xx` with a JSON error body. Check that `max_tokens` is present when required by the Messages API.
