`POST /v1/embeddings` создаёт векторные эмбеддинги для поиска, RAG и кластеризации.

## Пример

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## Примечания

- `input` может быть строкой или массивом строк (с учётом лимитов провайдера).
- Размерность и нормализация зависят от upstream-модели эмбеддингов.
- Биллинг обычно пропорционален числу входных токенов.

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
