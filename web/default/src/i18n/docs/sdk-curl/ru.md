curl удобен для отладки и CI smoke-тестов.

> Примеры кода и пути API сохранены на английском (технические идентификаторы).

curl is ideal for debugging and CI smoke tests.

## Chat

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

## List models

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## Pretty-print JSON

Pipe through `jq` when available:

```bash
curl -s ... | jq .
```

## Verbose debugging

Add `-v` to inspect TLS and headers. Redact Authorization when sharing logs.
