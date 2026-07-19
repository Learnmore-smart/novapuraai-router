`POST /v1/embeddings` 用於產生向量嵌入，適用於搜尋、RAG 與分群等場景。

## 範例

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## 說明

- `input` 可以是字串或字串陣列（受供應商限制約束）。
- 維度與正規化方式取決於上游嵌入模型。
- 計費通常與輸入 token 量成正比。

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
