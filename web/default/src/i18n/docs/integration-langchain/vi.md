LangChain và LlamaIndex hỗ trợ base URL OpenAI tùy chỉnh.

> Ví dụ mã và đường dẫn API giữ nguyên tiếng Anh (định danh kỹ thuật).

LangChain and LlamaIndex both support custom OpenAI base URLs.

## LangChain (Python)

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o-mini",
    api_key="sk-YOUR_KEY",
    base_url="https://www.novapuraai.com/v1",
)
print(llm.invoke("Hello").content)
```

## LlamaIndex

```python
from llama_index.llms.openai import OpenAI

llm = OpenAI(
    model="gpt-4o-mini",
    api_key="sk-YOUR_KEY",
    api_base="https://www.novapuraai.com/v1",
)
```

## Embeddings

Point embedding classes at the same base URL and an embedding model enabled on your gateway.
