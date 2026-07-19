每个中继请求前，先确认密钥、额度与模型是否可用。下面汇总接入时最常见的问题。

## 需要单独再部署一个 API 服务吗？

不需要。NovaPuraAI **本身就是** API 网关。将应用部署到例如 Google Cloud Run 并创建密钥后，客户端直接调用你的域名即可。

## docs_link 和 /docs 有什么区别？

`/docs` 是本站内置官方指南。可选的「文档链接」设置是一个外部 URL，可在页脚以 API 文档形式展示。

## 为什么会 401？

缺少 Authorization 头、密钥错误，或密钥已被删除/禁用。

## 为什么会 model_not_found？

模型未对你的分组启用，或密钥的模型白名单排除了该模型。

## 生产与本地开发能共用同一把密钥吗？

可以，但更推荐分开密钥，便于轮换与区分消耗。

## 能否部署在 Cloud Run？

可以。请使用 Cloud SQL（或其他托管数据库）、Redis 做多实例缓存，并配置稳定的 `SESSION_SECRET` / `CRYPTO_SECRET`。生产环境不要依赖容器内 SQLite。

## 用量在哪里查看？

控制台 → 用量日志 / 仪表盘。管理员可查看全站指标。

## API 是否兼容 OpenAI？

是。主流 chat、embeddings、images、audio 路由兼容 OpenAI。支持的模型还可通过 Anthropic Messages 与 Gemini 路径访问。
