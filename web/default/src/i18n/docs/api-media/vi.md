NovaPuraAI proxy các endpoint media đã chọn khi kênh tương ứng được bật.

## Hình ảnh

`POST /v1/images/generations`

```bash
curl https://www.novapuraai.com/v1/images/generations \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A minimal logo for an API platform",
    "size": "1024x1024"
  }'
```

## Phiên âm âm thanh

`POST /v1/audio/transcriptions` (multipart form)

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## Tổng hợp giọng nói

`POST /v1/audio/speech` trả về dữ liệu âm thanh cho các mô hình TTS được hỗ trợ.

## Xếp hạng lại (Rerank)

`POST /v1/rerank` nhận query và documents cho reranker kiểu Cohere/Jina khi đã cấu hình.

## Ghi chú thanh toán

Endpoint media thường tính phí theo số ảnh, giây hoặc số tài liệu — không chỉ token. Kiểm tra Model Square trước các tác vụ hàng loạt.
