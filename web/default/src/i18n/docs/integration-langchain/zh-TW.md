LangChain 與 LlamaIndex 可透過其 OpenAI 整合呼叫 NovaPuraAI，只需覆寫 Base URL 與 API 金鑰。閘道隨後會將模型名稱路由到已設定的渠道。

## 共用設定

```bash
export NOVAPURA_API_KEY="sk-xxxxxxxx"
export NOVAPURA_BASE_URL="https://www.novapuraai.com"   # origin only
```

SDK 用戶端通常需要帶 `/v1` 的 `base_url` / `baseURL`。

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

參數名（`api_base` 與 `base_url`）在不同 LlamaIndex 版本中略有差異——請使用你已安裝套件所接受的關鍵字參數。

## RAG 清單

1. 建索引與查詢階段使用 **相同** 的嵌入模型。
2. 將模型 ID 與向量索引中繼資料一併儲存。
3. 限制並行以遵守 [速率限制](/docs/rate-limits)。
4. 在評估檢索品質時監控 NovaPuraAI 用量日誌，使成本可預期。

## 相關文件

- [Python SDK](/docs/sdk-python)
- [嵌入](/docs/api-embeddings)
- [計費與額度](/docs/billing)
