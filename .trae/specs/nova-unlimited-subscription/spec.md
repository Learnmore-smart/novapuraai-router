# Spec: Nova Unlimited 订阅套餐 + Plans 落地页 + 主页模型列表更新

## 目标

新增一个「Nova Unlimited」月付自动续费订阅套餐：指定模型不限量使用、套餐内模型
不扣账户余额、套餐外模型继续按余额计费。配套 Plans 落地页、Stripe Checkout 购买
流程、优惠码（Stripe 原生 Coupon）、订阅管理页、Webhook 处理、affiliate 分佣接入，
以及主页模型列表更新（移除 GLM 5.2、补齐 DeepSeek V4 Flash）。

代码复用：本项目已有完整的订阅 / Stripe / affiliate / webhook 基建（见「现状」）。
本 spec 只描述**新增 / 修改**部分，不重复已实现的能力。

---

## 现状（已有基建，复用）

| 能力 | 位置 |
|---|---|
| `SubscriptionPlan` / `UserSubscription` 模型 + CRUD | `model/subscription.go:146-281` |
| 订阅 Stripe 支付（含 Checkout 创建） | `controller/subscription_payment_stripe.go` |
| Stripe webhook 签名验证 + 幂等表 | `controller/stripe_topup.go:377`、`model/stripe_webhook_event.go` |
| Stripe 多环境加密凭证（test / production） | `model/stripe_credential.go`、`setting/payment_stripe.go` |
| 资金源抽象（`FundingSource` / `WalletFunding` / `SubscriptionFunding`） | `service/funding_source.go` |
| 计费会话工厂（订阅 vs 钱包路由） | `service/billing_session.go:342` (`NewBillingSession`) |
| `BillingPreference`（`subscription_first` 等四档） | `common/str.go:121` |
| Affiliate 现金分佣（25% / 14 天冻结 / 提现） | `model/commission.go`、`controller/commission.go` |
| `User.StripeCustomer` 字段 | `model/user.go:75` |
| 现有订阅前端功能 | `web/default/src/features/subscriptions/` |
| 主页模型列表常量 | `web/default/src/features/home/constants.ts` |
| Navbar 动态链接 hook | `web/default/src/hooks/use-top-nav-links.ts` |
| TanStack Router 文件路由 | `web/default/src/routes/` |

---

## 关键设计决策（采用推荐方案）

### 1. 套餐模型：扩展现有 `SubscriptionPlan`

`SubscriptionPlan` 当前是配额制（`TotalAmount` / `AmountUsed`）。Nova Unlimited 需要
「指定模型不限量、$0 扣费」。在 `SubscriptionPlan` 上新增两个字段，引入
`model_coverage` 计费模式，复用所有已有支付 / webhook / 分佣管道：

```go
// model/subscription.go (SubscriptionPlan 新增字段)
BillingMode    string `json:"billing_mode" gorm:"type:varchar(16);default:'quota'"`
// 'quota' = 现有配额制；'model_coverage' = 套餐内模型 $0 计费、不消耗配额

AllowedModels  string `json:"allowed_models" gorm:"type:text;default:''"`
// 逗号分隔的模型 ID 列表（与 channel.Models 同格式）；'model_coverage' 模式下生效
// 后端唯一来源；前端不可决定某模型是否属于套餐
```

`UserSubscription` 不需要结构变更——`AmountTotal` / `AmountUsed` 在
`model_coverage` 模式下保持 0，仅作为订阅状态载体（`Status` / `EndTime` /
`StripeSubscriptionId` 等仍需使用，见下文）。

### 2. 优惠码：Stripe 原生 Coupon + Promotion Code

需求（百分比优惠、首月 / N 月 / 永久、后端验证、应用到 Stripe Checkout）与 Stripe
原生 Coupon 完全对应（`duration: once | repeating | forever`）。不建本地 promo 表：

- 管理员在 Stripe Dashboard 创建 Coupon + Promotion Code。
- 前端输入优惠码 → 后端调用 `promotioncode.List` 验证（active、未过期、未达上限）→
  返回 `coupon.percent_off` / `coupon.duration` / `coupon.duration_in_months` 给前端展示。
- 创建 Checkout Session 时把 `discounts[].coupon` 设为该 coupon ID。
- 退款 / 拒付时 Stripe 自动反向处理 coupon，无需本地撤销逻辑。

### 3. 币种选择：用户可切换 + 自动猜测

- 后端在 `dto.UserSetting` 增 `BillingCurrency string`（`'CNY' | 'USD' | ''`，空 = 自动）。
- Plans 页与 Subscribe Modal 显示币种切换器（CNY ¥99 / USD $19）。
- 匿名访客：前端用 `navigator.language` 猜测（`zh-*` → CNY，否则 USD）。
- 已登录用户：读 `UserSetting.BillingCurrency`，为空则用前端猜测值并写入。
- 后端按用户选择返回对应 `StripePriceId`（CNY / USD 两个 price，由后端配置）。

### 4. 分佣费率：复用 `common.AffCommissionRate`（25%）

订阅首付款 + 月度续费都按实付金额（coupon 后）走 `model.SettleRechargeCommission`
同款链路。退款 / 拒付触发 `model.RevertCommissionForTopUp` 等价逻辑（新增订阅版）。

---

## 数据模型变更

### `model/subscription.go` — `SubscriptionPlan` 新增字段

```go
BillingMode   string `json:"billing_mode" gorm:"type:varchar(16);default:'quota'"`
AllowedModels string `json:"allowed_models" gorm:"type:text;default:''"`
```

`NormalizeDefaults()` 兜底空值 → `'quota'`。

### `model/subscription.go` — `UserSubscription` 新增字段

```go
StripeSubscriptionId string `json:"stripe_subscription_id" gorm:"type:varchar(64);default:'';index"`
StripeCustomerId     string `json:"stripe_customer_id" gorm:"type:varchar(64);default:'';index"`
CancelAtPeriodEnd    bool   `json:"cancel_at_period_end"`
// Status 已存在（active/expired/cancelled）；扩展取值见「订阅状态映射」
```

`StripeSubscriptionId` 用于 webhook 幂等与「Manage subscription」跳转 Stripe Portal。

### `model/subscription.go` — `SubscriptionOrder` 新增字段

```go
StripeSessionId       string `json:"stripe_session_id" gorm:"type:varchar(64);default:'';index"`
PromotionCodeId       string `json:"promotion_code_id" gorm:"type:varchar(64);default:''"`
CouponId              string `json:"coupon_id" gorm:"type:varchar(64);default:''"`
OriginalAmountCents   int64  `json:"original_amount_cents" gorm:"type:bigint;default:0"`
DiscountAmountCents   int64  `json:"discount_amount_cents" gorm:"type:bigint;default:0"`
PaidAmountCents       int64  `json:"paid_amount_cents" gorm:"type:bigint;default:0"`
PaidCurrency          string `json:"paid_currency" gorm:"type:varchar(8);default:''"`
```

### `dto.UserSetting` — 新增字段

```go
BillingCurrency string `json:"billing_currency"`
```

### 迁移

跨三库（SQLite / MySQL / PostgreSQL）安全迁移，遵循 AGENTS.md：
- 在 `model/main.go:migrateDB()` 的 `AutoMigrate(...)` 列表中已包含
  `&SubscriptionPlan{}`、`&UserSubscription{}`、`&SubscriptionOrder{}`，新增字段会
  自动 `ALTER TABLE ... ADD COLUMN`（SQLite 兼容）。
- **不**使用 `gorm:"default:true"` 类布尔默认；`CancelAtPeriodEnd` 默认 false 由
  Go 零值保证，不写 gorm default tag。

---

## 后端：计费注入（套餐内模型 $0）

### 注入点

`service/billing_session.go:342` (`NewBillingSession`)，在
`pref := common.NormalizeBillingPreference(...)` 之后、`tryWallet` / `trySubscription`
之前，新增分支：

```go
if subId, planId, planTitle, ok, err := model.UserActiveSubscriptionCoversModel(
    relayInfo.UserId, relayInfo.OriginModelName,
); err == nil && ok {
    return &BillingSession{
        funding:         NewModelCoverageFunding(subId, planId, planTitle),
        preConsumedQuota: 0,
        trusted:         true,
        settled:         false,
        refunded:        false,
    }, nil
}
```

### 新增 `ModelCoverageFunding`

`service/funding_source.go` 新增：

```go
type ModelCoverageFunding struct {
    subscriptionId int
    planId         int
    planTitle      string
}
// Source() = "subscription_covered"
// PreConsume(int) error  → nil（no-op）
// Settle(int) error      → nil（no-op，不扣配额）
// Refund() error         → nil
```

### 新增 model 层查询

`model/subscription.go` 紧邻 `HasActiveUserSubscription`（line 840）新增：

```go
func UserActiveSubscriptionCoversModel(userId int, modelName string) (
    subId int, planId int, planTitle string, ok bool, err error,
)
```

- 查 `user_subscriptions` 中 `status='active' AND end_time > now`。
- 对每条加载其 `SubscriptionPlan`，仅当 `plan.BillingMode == 'model_coverage'` 时
  解析 `plan.AllowedModels`（逗号分隔）。
- 匹配 `modelName` 时使用 `ratio_setting.FormatMatchingModelName` 归一化（与定价
  查询一致），同时支持精确匹配。
- 命中即返回。失败 / 无命中 → `ok=false`。

### 异步任务路径

`service/task_billing.go:90` (`taskAdjustFunding`) 新增分支：
若 `task.PrivateData.BillingSource == "subscription_covered"`，直接返回 nil，不调整
任何资金源。`task.PrivateData` 由 `relayInfo.BillingSource` 在任务创建时快照。

### 使用记录（即使 $0 也记录）

`PostTextConsumeQuota` / `PostAudioConsumeQuota` / `PostWssConsumeQuota` /
`LogTaskConsumption` 在调用 `RecordConsumeLog` 时：
- `Quota` 字段保持 `summary.Quota`（实际计算值），但 `Billing.Settle` 是 no-op，
  所以**不会真扣**。
- `other.billing_source = "subscription_covered"`、`other.subscription_id`、
  `other.subscription_plan_id`、`other.subscription_plan_title` 由现有
  `appendBillingInfo`（`service/log_info_generate.go:159`）写入。
- `model.UpdateUserUsedQuota` / `UpdateChannelUsedQuota` 当且仅当
  `summary.Quota > 0` 才调用（现有行为，已是 `> 0` 守卫）。

### 别名映射

`relayInfo.OriginModelName` 是客户端模型名（channel `model_mapping` 应用前），与
`SubscriptionPlan.AllowedModels` 比对时使用同一命名空间。`FormatMatchingModelName`
归一化已覆盖 thinking-budget / gizmo 变体。

---

## 后端：Stripe Checkout 路由

新增路由（在 `router/api-router.go` 既有 `subscriptionRoute` 组下扩展）：

| 方法 | 路径 | 鉴权 | 处理 |
|---|---|---|---|
| GET | `/api/subscription/plans/public` | 公开 | 返回启用的套餐（仅公开字段，过滤掉内部 ID） |
| GET | `/api/subscription/self/full` | UserAuth | 返回当前订阅 + 状态 + 续费日 + coupon + Stripe Portal 可用性 |
| POST | `/api/subscription/stripe/checkout` | UserAuth + CriticalRateLimit | 创建 Checkout Session（带 coupon） |
| POST | `/api/subscription/stripe/validate-promo` | UserAuth + CriticalRateLimit | 验证优惠码，返回折扣详情 |
| GET | `/api/subscription/stripe/portal` | UserAuth | 创建 Stripe Customer Portal Session，返回 URL |
| POST | `/api/stripe/subscription/webhook` | 公开（签名验证） | 订阅专属 webhook |
| GET | `/api/subscription/success` | UserAuth | 支付成功页（仅展示，**不激活订阅**——激活由 webhook 完成） |
| GET | `/api/subscription/canceled` | UserAuth | 支付取消页 |

### 防重复订阅

`SubscriptionRequestStripeCheckout`（新增控制器）：
1. 调 `model.HasActiveUserSubscription(userId)`，已有有效订阅 → 返回
   `{"error": "already_subscribed", "manage_url": "..."}`，前端引导到 Manage。
2. 创建 `SubscriptionOrder`（status=`pending`）→ 创建 Stripe Session →
   把 `session.Id` 写回 order。
3. 前端按钮点击后立即 loading，禁用按钮，防止重复点击。
4. 已有 `pending` 订单且未过期（5 分钟内）→ 复用同一 Session URL。

### Checkout Session 参数

```go
params := &stripe.CheckoutSessionParams{
    Mode:                  stripe.String(string(stripe.CheckoutSessionModeSubscription)),
    ClientReferenceID:     stripe.String(referenceId),
    Customer:              stripe.String(user.StripeCustomer),  // 若已存在
    CustomerEmail:         stripe.String(user.Email),            // 若新建
    LineItems:             []*stripe.CheckoutSessionLineItemParams{{
        Price:    stripe.String(plan.StripePriceIdByCurrency(currency)),
        Quantity: stripe.Int64(1),
    }},
    Discounts:             []*stripe.CheckoutSessionDiscountParams{{
        Coupon: stripe.String(couponId),  // 仅当用户输入并通过 validate-promo
    }},
    SuccessURL:            stripe.String(paymentReturnPath("/subscription/success?order={REFERENCE}")),
    CancelURL:             stripe.String(paymentReturnPath("/subscription/canceled?order={REFERENCE}")),
    SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
        Metadata: map[string]string{
            "user_id":    strconv.Itoa(userId),
            "plan_id":    strconv.Itoa(plan.Id),
            "trade_no":   referenceId,
        },
    },
}
```

`SubscriptionPlan` 需要支持双币种 Stripe Price：在 `SubscriptionPlan` 上把
`StripePriceId` 字段语义扩展为 JSON `{"USD":"price_...","CNY":"price_..."}`，或新增
`StripePriceIdCNY` 字段。**采用 JSON 方案**（向后兼容旧字符串：解析失败按 USD 处理）。

---

## 后端：Webhook 处理

新增 `controller/subscription_stripe_webhook.go`，路由
`POST /api/stripe/subscription/webhook`。复用 `setting.StripeWebhookSecret` 与
`stripe-go/v85/webhook.ConstructEventWithOptions`。

幂等：复用 `model.TryInsertStripeWebhookEvent`（已有表，已注册 AutoMigrate）。

处理事件：

| Stripe 事件 | 处理 |
|---|---|
| `checkout.session.completed`（subscription mode） | 记录 `StripeSubscriptionId` / `StripeCustomerId`；不激活（等 `invoice.paid`） |
| `invoice.paid`（首次 + 月度续费） | 激活 / 续期 `UserSubscription`；落 `SubscriptionOrder.PaidAmountCents` / `PaidCurrency`；触发 affiliate 分佣 |
| `invoice.payment_failed` | 标记 `UserSubscription.Status = 'past_due'` |
| `customer.subscription.updated` | 同步 `cancel_at_period_end`、`current_period_end`、`status` |
| `customer.subscription.deleted` | 标记 `Status = 'expired'` / `'canceled'`；触发 affiliate 佣金撤销（若在冻结期内） |
| `charge.refunded` | 调 `RevertCommissionForSubscription`（新增）按比例撤销佣金 |

### 订阅状态映射

| Stripe `subscription.status` | `UserSubscription.Status` | 前端按钮 |
|---|---|---|
| `active`（`cancel_at_period_end=false`） | `active` | Current plan / Manage subscription |
| `active`（`cancel_at_period_end=true`） | `canceling` | Reactivate / Manage |
| `past_due` | `past_due` | Update payment method / Manage |
| `unpaid` / `incomplete` | `payment_failed` | Update payment method |
| `canceled` | `canceled` | Subscribe now |
| `expired`（end_of_period 到期且未续费） | `expired` | Subscribe now |

---

## 后端：Affiliate 接入

### 首次付款 + 续费

`invoice.paid` 处理时：
1. 从 `SubscriptionOrder`（按 `trade_no` = `invoice.metadata.trade_no`）拿到
   `PaidAmountCents` / `PaidCurrency` / `UserId`。
2. 调 `model.SettleSubscriptionCommission(userId, orderId, paidAmountCents, paidCurrency)`
   （新增，紧邻 `model/commission.go:131` `SettleRechargeCommission`）：
   - 查 `user.InviterId`，无 → 返回。
   - 查 `user.CommissionApproved`，false → 走旧 ¥100 额度路径（不进现金分佣）。
   - true → 按 `common.AffCommissionRate`（25%）算佣金美分，落 `Commission` 表
     （`TopUpId` 字段复用为订阅订单号；幂等键 `(topup_id, inviter_id)`）。
   - 进 `PendingCommissionCents`，14 天后由现有 `ReleaseMaturedCommissions` 解冻。

### 退款 / 拒付

`charge.refunded` 时新增 `RevertCommissionForSubscription(orderId)`：
- 找该订单对应的 `Commission` 行，若 `Status='pending'` → 改 `reverted`，从
  `PendingCommissionCents` 扣回。
- 若 `Status='available'` / 已解冻 → 从 `CommissionBalanceCents` 扣回（可能为负，
  允许，下次佣金优先补回）。落 `RevertedAt` / `RevertReason='subscription_refund'`。

---

## 前端：Plans 落地页 + Navbar

### Navbar 新增链接

`web/default/src/hooks/use-top-nav-links.ts`：在 `Model Square` 后追加
`links.push({ title: 'Plans', href: '/plans' })`。同一数组驱动桌面与移动端，自动覆盖。

i18n key `Plans` 加入所有 `web/default/src/i18n/locales/*.json`。

### `/plans` 路由（公开）

新建 `web/default/src/routes/plans/index.tsx`：
```tsx
export const Route = createFileRoute('/plans/')({ component: Plans })
```

特性模块 `web/default/src/features/plans/`：
- `index.tsx` — 页面组件
- `api.ts` — `getPublicPlans()`、`validatePromo(code, planId, currency)`、
  `createCheckout(planId, currency, promoCode?)`、`getSubscriptionStatus()`、
  `createPortalSession()`
- `hooks/use-plans.ts`、`use-subscription-status.ts`、`use-promo.ts`、`use-checkout.ts`
- `components/plan-card.tsx`、`subscribe-modal.tsx`、`promo-confirm-dialog.tsx`、
  `currency-switcher.tsx`、`subscription-status-badge.tsx`

页面内容：
- Hero：Nova Unlimited 标题、副标题、月费（按币种显示 ¥99 / $19）、自动续费 / 随时取消说明。
- 权益列表：套餐内模型不限量、不扣余额、套餐外按余额计费、随时取消。
- Subscribe now 按钮（已登录未订阅 → 弹 Modal；已订阅 → 显示 Current plan + Manage）。
- 币种切换器（顶部）。
- FAQ：自动续费、取消、套餐内模型列表（后端返回）。

刷新直接访问：TanStack Router 文件路由天然支持；后端 `/api/subscription/plans/public`
公开访问；SPA fallback 由 `router/web-router.go` 已处理。

### Subscribe Modal（Base UI Dialog）

字段：
- 套餐名称（Nova Unlimited）
- 月费（按当前币种）
- 自动续费说明
- 随时取消说明
- 优惠码输入框 + Apply 按钮
- 原价 / 优惠金额 / 本次支付金额 / 下次续费金额（用 `validate-promo` 返回值填充）
- 前往 Stripe Checkout 按钮（点击 → loading → 跳转 `pay_link`）

不显示：月份数量、3/6/12 月选项、预付选项（确认 Modal 字段固定为月付）。

### Promo Confirm Dialog

`Apply` 成功后弹出：
- 原价：¥99
- 优惠：25%
- 本次支付：¥74.25
- 下个月起：¥99/月
- 优惠范围：仅首月 / 持续 N 月 / 永久（来自 `coupon.duration` + `duration_in_months`）

无效 / 过期 / 达上限 → 错误 toast，不弹确认。

### `/account/subscription` 路由（受保护）

新建 `web/default/src/routes/_authenticated/account/subscription/index.tsx`：
```tsx
export const Route = createFileRoute('/_authenticated/account/subscription/')({
  component: AccountSubscription,
})
```

注意：项目当前**没有** `/account` 父路由；按现有 `_authenticated/profile/`、
`_authenticated/wallet/` 平级模式，新建 `_authenticated/account/subscription/`，父段
`account` 是无 layout 的路径段（TanStack Router 支持嵌套无 layout 路由）。

页面内容：
- 当前套餐名称
- 订阅状态 badge（Active / Canceling / Canceled / Past due / Payment failed / Expired）
- 月费 + 币种
- 下次续费日期
- 是否已取消续费（`cancel_at_period_end`）
- 当前优惠（若 coupon 仍生效）
- Manage subscription（→ Stripe Portal）
- View invoices（→ Stripe Portal invoices）
- Update payment method（→ Stripe Portal）
- Cancel subscription（→ Stripe Portal 或调 Stripe API `subscription.update(cancel_at_period_end=true)`）

---

## 前端：主页模型列表更新

### `web/default/src/features/home/constants.ts`

```ts
export const LANDING_MODEL_ROWS = [
  { name: 'DeepSeek V4 Pro', note: 'Pay per token' },
  { name: 'DeepSeek V4 Flash', note: 'Pay per token' },
  { name: 'Nemotron 3 Ultra', note: 'Pay per token' },
  { name: 'Laguna XS 2.1', note: 'Pay per token' },
] as const
```

移除 `GLM 5.2`，新增 `DeepSeek V4 Flash`，保留 `DeepSeek V4 Pro` / `Nemotron 3 Ultra` /
`Laguna XS 2.1`。

### `web/default/src/features/home/constants.test.ts`

当前测试已过期（断言 `Kimi K2.6`，与实际常量不符）。重写为：
```ts
test('advertises current flagship models instead of unavailable provider families', () => {
  assert.deepEqual(LANDING_MODEL_ROWS, [
    { name: 'DeepSeek V4 Pro', note: 'Pay per token' },
    { name: 'DeepSeek V4 Flash', note: 'Pay per token' },
    { name: 'Nemotron 3 Ultra', note: 'Pay per token' },
    { name: 'Laguna XS 2.1', note: 'Pay per token' },
  ])
  const advertisedNames = LANDING_MODEL_ROWS.map((m) => m.name).join(' ')
  assert.doesNotMatch(advertisedNames, /GPT|Claude|Gemini|GLM/i)
})
```

### i18n

`web/default/src/i18n/locales/*.json`（en/zh/zh-TW/fr/ru/ja/vi）：
- 新增 key `DeepSeek V4 Flash`（其他三个已存在）。
- 移除 `GLM 5.2` key（可选清理）。
- 新增 `Plans`、`Subscribe now`、`Current plan`、`Manage subscription`、
  `Update payment method`、`Reactivate`、`Cancel subscription`、`View invoices`、
  `Next renewal`、`Promo code`、`Apply`、`Original price`、`Discount`、`Due today`、
  `Next renewal amount`、`Auto-renews monthly`、`Cancel anytime` 等键。

运行 `bun run i18n:sync` 自动补齐缺失键。

---

## Stripe 配置

管理员在 Stripe Dashboard / 本项目 admin 后台完成：

1. **Product**: `Nova Unlimited`
2. **Prices**:
   - `price_...usd`: US$19.00 / month, recurring, USD
   - `price_...cny`: ¥99.00 / month, recurring, CNY
3. **Coupon / Promotion Code**（可选，按需创建）。
4. **Webhook endpoint**: `https://<host>/api/stripe/subscription/webhook`，
   事件至少订阅：`checkout.session.completed`、`invoice.paid`、
   `invoice.payment_failed`、`customer.subscription.updated`、
   `customer.subscription.deleted`、`charge.refunded`。
5. **Customer Portal**: 在 Stripe Dashboard 启用，配置允许 cancel / update payment /
   view invoices。
6. **Test / Live 凭证**: 通过现有 `PUT /api/option/stripe/:environment/credentials`
   写入（已加密落库）。

`SubscriptionPlan.StripePriceId` 存 JSON：
```json
{"USD":"price_xxx","CNY":"price_yyy"}
```

新增 `SubscriptionPlan.StripePriceIdByCurrency(currency string) string` 解析方法，
兼容旧单字符串值（视为 USD）。

---

## 路由刷新 / 直接访问保障

- 所有新增前端路由文件 → TanStack Router 重新生成 `routeTree.gen.ts`。
- 后端 `/api/subscription/plans/public`、`/api/stripe/subscription/webhook` 公开。
- SPA fallback `router/web-router.go:22-37` 已将未匹配 GET 请求回退到 `index.html`，
  保证 `/plans`、`/account/subscription`、`/subscription/success` 直接访问与刷新不报错。
- 后端新增 API 路由在 `router/api-router.go` 注册，重启即生效。

---

## 测试

### 后端

- `service/billing_session_test.go`：`model_coverage` 模式下 `NewBillingSession`
  返回 `ModelCoverageFunding`，`Settle` 不扣 quota。
- `model/subscription_test.go`：`UserActiveSubscriptionCoversModel` 命中 / 未命中 /
  多订阅 / 别名归一化。
- `controller/subscription_stripe_webhook_test.go`：`invoice.paid` 激活 + 分佣 +
  幂等（重复事件不重复激活）。
- 遵循 AGENTS.md：`testify/require` 用于 setup + fatal，`testify/assert` 用于非致命。
- 不加纯覆盖率 / 随机 fuzz测试。

### 前端

- `web/default/src/features/home/constants.test.ts` 重写（见上）。
- Plans 页面 / Subscribe Modal / Promo Confirm Dialog 通过 `bun run build` 类型检查。

---

## 实施顺序（建议分阶段）

1. **Phase 1（隔离小改动）**: 主页模型列表更新 + 测试 + i18n。
2. **Phase 2（数据模型 + 计费注入）**: `SubscriptionPlan` / `UserSubscription` /
   `SubscriptionOrder` 新增字段 + `ModelCoverageFunding` + `NewBillingSession` 注入 +
   `UserActiveSubscriptionCoversModel` + 异步任务路径守卫。
3. **Phase 3（后端支付路由）**: `/api/subscription/stripe/checkout` /
   `validate-promo` / `portal` / `self/full` / `plans/public` + `SubscriptionPlan`
   双币种 Price 解析 + 防重复订阅。
4. **Phase 4（Webhook）**: `controller/subscription_stripe_webhook.go` + 事件处理 +
   状态映射 + 幂等。
5. **Phase 5（Affiliate）**: `SettleSubscriptionCommission` /
   `RevertCommissionForSubscription` + webhook 集成。
6. **Phase 6（前端 Plans 页）**: Navbar 链接 + `/plans` 路由 + 特性模块 + Subscribe
   Modal + Promo Confirm Dialog + Currency Switcher + Subscription Status Badge。
7. **Phase 7（前端订阅管理页）**: `/account/subscription` + Manage/Cancel/Reactivate
   按钮（跳 Stripe Portal）。
8. **Phase 8（Stripe 配置文档）**: 在 admin 后台增加 Nova Unlimited plan 配置说明
   （Product / Price / Webhook / Portal 启用步骤）。

每个 Phase 内自测通过后再进入下一个。

---

## 不在本 spec 范围内

- 旧版 quota 制订阅的迁移或下线（保持现有 `/subscriptions` 与
  `subscription_payment_{creem,waffo_pancake,epay}.go` 不变）。
- 套餐模型列表的前端管理 UI（后端字段已就绪，admin UI 后续单独迭代）。
- 多套餐比较页（当前只有一个 Nova Unlimited）。
- 一次性预付 / 多月套餐选项（已明确不做）。
- 套餐内模型的限速（不限量 = 不限速，按现有 channel 限速走）。
