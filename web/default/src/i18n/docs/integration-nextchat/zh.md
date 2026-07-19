NextChat（ChatGPT-Next-Web 及兼容分支）可通过 OpenAI 兼容设置连接 NovaPuraAI。一次配置 Base URL、API 密钥与默认模型后，即可按往常使用 Web UI。

## 前置条件

- 正在运行的 NextChat（自托管或本地）
- NovaPuraAI API 密钥
- 部署源站例如 `https://www.novapuraai.com`

## 需要配置的设置

在 NextChat **设置** 中（文案可能因分支/版本而异）：

| 字段 | 推荐值 |
| --- | --- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key | `sk-xxxxxxxx` |
| Model | 来自 `GET /v1/models` 的 ID |

若界面只保存源站并自行追加 `/v1`，则使用不含重复 `/v1` 的 `https://www.novapuraai.com`。不确定时，打开浏览器网络工具，确认最终路径为 `/v1/chat/completions`。

## 环境变量风格部署

许多 NextChat Docker 镜像接受：

```bash
OPENAI_API_KEY=sk-xxxxxxxx
BASE_URL=https://www.novapuraai.com/v1
# some images use OPENAI_API_BASE / OPENAI_BASE_URL — check your image docs
```

修改环境变量后请重启容器。

## 冒烟测试

```bash
curl "https://www.novapuraai.com/v1/chat/completions" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

## 常见问题

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 对话返回 404 | Base URL 错误（缺少或重复了 `/v1`） | 对照网络面板中的最终路径 |
| 401 | 未传入密钥或密钥错误 | 粘贴 NovaPuraAI 密钥，而非 OpenAI 平台密钥 |
| 模型列表为空 | 前端无法调用 `/v1/models` | 检查 CORS/代理与密钥权限 |
| 余额错误 | 无额度 | 在 NovaPuraAI 控制台充值 |

## 安全说明

- 若 NextChat 构建支持服务端代理模式，优先用其隐藏浏览器中的密钥。
- 公开演示请使用低额度密钥并严格限制模型白名单。

## 相关文档

- [Base URL 与端点](/docs/base-url)
- [身份验证](/docs/authentication)
- [常见问题](/docs/faq)
