`GET /v1/models` 列出目前金鑰可用的模型。

## 範例

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## 回應结构

The payload follows OpenAI’s list object with `data[]` entries containing at least `id` and `object`. Additional metadata may appear depending on gateway version and settings.

## 找不到模型時

1. Confirm the model is enabled in admin channels / abilities for your group.
2. Confirm your key is not restricted away from that model.
3. Refresh Model Square in the UI for pricing and availability.

## 快取

Clients may cache the list for a short TTL. Re-fetch after admin changes or on `404 model_not_found` errors.
