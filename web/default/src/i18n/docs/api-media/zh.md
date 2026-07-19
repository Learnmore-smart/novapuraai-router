当对应渠道已启用时，NovaPuraAI 会代理选定的媒体端点。

## 图像

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

## 音频转写

`POST /v1/audio/transcriptions`（multipart 表单）

```bash
curl https://www.novapuraai.com/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -F file="@speech.mp3" \
  -F model="whisper-1"
```

## 语音合成

`POST /v1/audio/speech` 对受支持的 TTS 模型返回音频字节。

## 重排序（Rerank）

`POST /v1/rerank` 在已配置时接受 query 与 documents，用于 Cohere / Jina 风格的重排序器。

## 计费说明

媒体端点通常按图像数量、秒数或文档数量计费，而不仅是 token。批量任务前请先查看模型广场。
