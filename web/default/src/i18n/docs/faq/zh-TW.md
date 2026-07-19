每個中繼請求前，先確認金鑰、額度與模型是否可用。下面彙總接取時最常見的問題。

## 需要另外再部署一個 API 服務嗎？

不需要。NovaPuraAI **本身就是** API 閘道。將應用部署到例如 Google Cloud Run 並建立金鑰後，用戶端直接呼叫你的網域即可。

## docs_link 和 /docs 有什麼差別？

`/docs` 是本站內建官方指南。可選的「文件連結」設定是一個外部 URL，可在頁尾以 API 文件形式展示。

## 為什麼會 401？

缺少 Authorization 頭、金鑰錯誤，或金鑰已被刪除/停用。

## 為什麼會 model_not_found？

模型未對你的分組啟用，或金鑰的模型白名單排除了該模型。

## 正式環境與本機開發能共用同一把金鑰嗎？

可以，但更建議分開金鑰，便於輪換與區分消耗。

## 能否部署在 Cloud Run？

可以。請使用 Cloud SQL（或其他託管資料庫）、Redis 做多實例快取，並設定穩定的 `SESSION_SECRET` / `CRYPTO_SECRET`。正式環境不要依賴容器內 SQLite。

## 用量在哪裡查看？

控制台 → 用量日誌 / 儀表板。管理員可查看全站指標。

## API 是否相容 OpenAI？

是。主流 chat、embeddings、images、audio 路由相容 OpenAI。支援的模型還可透過 Anthropic Messages 與 Gemini 路徑存取。
