curl 适合调试与 CI 冒烟测试。

## 对话

```bash
curl https://www.novapuraai.com/v1/chat/completions \
  -H "Authorization: Bearer $NOVAPURA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

## 列出模型

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer $NOVAPURA_API_KEY"
```

## 美化 JSON

Pipe through `jq` when available:

```bash
curl -s ... | jq .
```

## 详细调试

Add `-v` to inspect TLS and headers. Redact Authorization when sharing logs.
