`POST /v1/embeddings` 用于生成向量嵌入，适用于搜索、RAG 与聚类等场景。

## 示例

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## 说明

- `input` 可以是字符串或字符串数组（受提供商限制约束）。
- 维度与归一化方式取决于上游嵌入模型。
- 计费通常与输入 token 量成正比。

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
