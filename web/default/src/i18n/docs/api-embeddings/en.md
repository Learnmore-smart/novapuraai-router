`POST /v1/embeddings` creates vector embeddings for search, RAG, and clustering.

## Example

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## Notes

- `input` may be a string or an array of strings (subject to provider limits).
- Dimensions and normalization depend on the upstream embedding model.
- Billing is typically proportional to input tokens.

## Python

```python
from openai import OpenAI
client = OpenAI(api_key="sk-YOUR_KEY", base_url="https://www.novapuraai.com/v1")
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="hello world",
)
print(len(emb.data[0].embedding))
```
