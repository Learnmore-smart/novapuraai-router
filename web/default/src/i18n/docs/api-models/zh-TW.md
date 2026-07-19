`GET /v1/models` 列出目前驗證金鑰可用的模型。

## 範例

```bash
curl https://www.novapuraai.com/v1/models \
  -H "Authorization: Bearer sk-YOUR_KEY"
```

## 回應結構

負載遵循 OpenAI 的列表物件格式，`data[]` 中至少包含 `id` 與 `object`。依閘道版本與設定，可能還會出現額外中繼資料。

## 模型缺失時

1. 確認該模型已在管理後台的渠道 / 能力中為你的分組啟用。
2. 確認你的金鑰未被限制使用該模型。
3. 在介面中重新整理模型廣場，查看定價與可用性。

## 快取

用戶端可對列表做短 TTL 快取。管理員變更後，或出現 `404 model_not_found` 錯誤時，請重新擷取。
