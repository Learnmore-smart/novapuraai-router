LangChain and LlamaIndex can call NovaPuraAI through their OpenAI integrations by overriding the base URL and API key. The gateway then routes model names to configured channels.

## Shared configuration

```bash
export NOVAPURA_API_KEY="sk-xxxxxxxx"
export NOVAPURA_BASE_URL="https://www.novapuraai.com"   # origin only
```

SDK clients generally need `base_url` / `base_url` **with** `/v1`.

## LangChain (Python)

```bash
pip install langchain-openai
```

```python
import os
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
    temperature=0.2,
)

print(llm.invoke("Hello from NovaPuraAI").content)
```

### Embeddings with LangChain

```python
from langchain_openai import OpenAIEmbeddings

emb = OpenAIEmbeddings(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    base_url=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)
vector = emb.embed_query("gateway documentation")
```

## LangChain.js

```bash
npm install @langchain/openai
```

```typescript
import { ChatOpenAI } from "@langchain/openai";

const llm = new ChatOpenAI({
  model: "gpt-4o-mini",
  apiKey: process.env.NOVAPURA_API_KEY,
  configuration: {
    baseURL: `${process.env.NOVAPURA_BASE_URL}/v1`,
  },
});

const res = await llm.invoke("Hello from NovaPuraAI");
console.log(res.content);
```

## LlamaIndex (Python)

```bash
pip install llama-index-llms-openai llama-index-embeddings-openai
```

```python
import os
from llama_index.llms.openai import OpenAI
from llama_index.embeddings.openai import OpenAIEmbedding

llm = OpenAI(
    model="gpt-4o-mini",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

embed = OpenAIEmbedding(
    model="text-embedding-3-small",
    api_key=os.environ["NOVAPURA_API_KEY"],
    api_base=os.environ["NOVAPURA_BASE_URL"].rstrip("/") + "/v1",
)

print(llm.complete("Hello from NovaPuraAI"))
```

Parameter names (`api_base` vs `base_url`) differ slightly across LlamaIndex versions—prefer the keyword accepted by your installed package.

## RAG checklist

1. Use the **same** embedding model for index and query time.
2. Store the model ID alongside the vector index metadata.
3. Cap concurrency to respect [Rate Limits](/docs/rate-limits).
4. Monitor NovaPuraAI usage logs while evaluating retrieval quality so cost stays predictable.

## Related

- [Python SDK](/docs/sdk-python)
- [Embeddings](/docs/api-embeddings)
- [Billing & Quota](/docs/billing)
