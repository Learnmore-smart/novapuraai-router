`POST /v1/embeddings` tạo vector embedding cho tìm kiếm, RAG và phân cụm.

## Ví dụ

```bash
curl https://www.novapuraai.com/v1/embeddings \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["NovaPuraAI is an API gateway", "second document"]
  }'
```

## Ghi chú

- `input` có thể là chuỗi hoặc mảng chuỗi (tuân theo giới hạn của nhà cung cấp).
- Số chiều và chuẩn hóa phụ thuộc mô hình embedding upstream.
- Thanh toán thường tỷ lệ với số token đầu vào.

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
