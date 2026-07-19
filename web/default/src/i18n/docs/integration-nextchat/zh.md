NextChat（ChatGPT-Next-Web 及兼容分支）可通过 OpenAI 兼容设置连接 NovaPuraAI。配置好 Base URL、API Key 与默认模型后，即可像往常一样使用 Web UI。

## 前置条件

- 已运行 NextChat（自托管或本地）
- NovaPuraAI API Key
- 部署源站，例如 `https://www.novapuraai.com`

## 需要填写的设置

在 NextChat **设置**（文案因分支/版本而异）：

| 字段 | 推荐值 |
| --- | --- |
| Endpoint / API base | `https://www.novapuraai.com/v1` |
| API key | `sk-xxxxxxxx` |
| Model | 来自 `GET /v1/models` 的 ID |

若界面只保存源站并自行追加 `/v1`，则填写 `https://www.novapuraai.com`，避免重复 `/v1`。不确定时，打开浏览器网络面板，确认最终路径为 `/v1/chat/completions`。

## 环境变量风格部署

许多 NextChat Docker 镜像支持：

```bash
OPENAI_API_KEY=sk-xxxxxxxx
BASE_URL=https://www.novapuraai.com/v1
# 部分镜像使用 OPENAI_API_BASE / OPENAI_BASE_URL — 以镜像文档为准
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
| 聊天 404 | Base URL 错误（缺 `/v1` 或重复） | 对照网络面板路径修正 |
| 401 | 未传密钥或密钥错误 | 粘贴 NovaPuraAI 密钥，而非 OpenAI 平台密钥 |
| 模型列表为空 | 前端无法调用 `/v1/models` | 检查 CORS/代理与密钥权限 |
| 余额错误 | 额度不足 | 在 NovaPuraAI 控制台充值 |

## 安全说明

- 若 NextChat 支持服务端代理模式，优先隐藏浏览器中的密钥。
- 公开演示请使用低额度密钥，并尽量限制可用模型。

## 相关文档

- [Base URL 与端点](/docs/base-url)
- [鉴权](/docs/authentication)
- [常见问题](/docs/faq)
