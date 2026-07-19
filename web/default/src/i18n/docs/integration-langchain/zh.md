LangChain 与 LlamaIndex 可通过其 OpenAI 集成调用 NovaPuraAI，只需覆盖 Base URL 与 API 密钥。网关随后会将模型名称路由到已配置的渠道。

## 共用配置

```bash
export NOVAPURA_API_KEY="sk-xxxxxxxx"
export NOVAPURA_BASE_URL="https://www.novapuraai.com"   # origin only
```

SDK 客户端通常需要带 `/v1` 的 `base_url` / `baseURL`。

## LangChain（Python）

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

### 在 LangChain 中使用嵌入

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

## LlamaIndex（Python）

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

参数名（`api_base` 与 `base_url`）在不同 LlamaIndex 版本中略有差异——请使用你已安装包所接受的关键字参数。

## RAG 清单

1. 建索引与查询阶段使用 **相同** 的嵌入模型。
2. 将模型 ID 与向量索引元数据一并保存。
3. 限制并发以遵守 [速率限制](/docs/rate-limits)。
4. 在评估检索质量时监控 NovaPuraAI 用量日志，使成本可预期。

## 相关文档

- [Python SDK](/docs/sdk-python)
- [嵌入](/docs/api-embeddings)
- [计费与额度](/docs/billing)
