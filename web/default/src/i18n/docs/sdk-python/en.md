Use the official `openai` Python package with a custom base URL.

## Install

```bash
pip install openai
```

## Client

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url="https://www.novapuraai.com/v1",
)
```

## Chat

```python
completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
)
print(completion.choices[0].message.content)
```

## Streaming

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

## Embeddings

```python
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="NovaPuraAI gateway",
)
vector = emb.data[0].embedding
```
