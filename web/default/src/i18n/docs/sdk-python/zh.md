使用官方 `openai` Python 包，并配置自定义 base_url。

## 安装

```bash
pip install openai
```

## 客户端

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## 对话

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(completion.choices[0].message.content)
```

## 流式输出

```python
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Stream a short poem."}],
    stream=True,
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

## 向量

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="NovaPuraAI gateway",
)
vector = emb.data[0].embedding
```
