当管理员启用 Gemini 渠道后，可通过 `/v1beta` 下的 Google 风格路径访问 Gemini 兼容流量。

## 生成内容

```bash
curl "https://www.novapuraai.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Write a haiku about APIs."}]
    }]
  }'
```

## 提示

- 确切的模型 ID 取决于你部署中的渠道配置。
- 多模态部分（内联数据 / 文件 URI）遵循 Gemini 请求结构；请将负载控制在网关请求体限制内。
- 若渠道适配器已映射，也可通过 OpenAI 兼容的对话接口访问部分 Gemini 模型。

## 鉴权

使用相同的 NovaPuraAI `sk-` 密钥。除非你是正在配置上游渠道的管理员，否则不要向 NovaPuraAI 发送 Google AI Studio 密钥。
