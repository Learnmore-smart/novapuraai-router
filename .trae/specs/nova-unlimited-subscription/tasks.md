# Tasks

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax for tracking. Follow phase order; each phase self-tests before the next begins. Spec: `.trae/specs/nova-unlimited-subscription/spec.md`.

**Goal:** 实现 Nova Unlimited 月付订阅套餐（套餐内模型 $0 计费）+ Plans 落地页 + Stripe Checkout + 优惠码 + 订阅管理页 + Webhook + affiliate 分佣，并更新主页模型列表。

**Architecture:** 复用现有 `SubscriptionPlan` / Stripe / affiliate / webhook 基建。在 `SubscriptionPlan` 新增 `BillingMode` + `AllowedModels` 字段，在 `NewBillingSession` 注入 `ModelCoverageFunding`（$0 扣费 no-op）。优惠码用 Stripe 原生 Coupon。前端新增 `/plans` 与 `/account/subscription` 路由。

**Tech Stack:** Go 1.22 + Gin + GORM v2（SQLite/MySQL/PostgreSQL）；React 19 + TanStack Router + TanStack Query + Base UI + Tailwind；Stripe Go SDK v85；i18next。

---

## Phase 1: 主页模型列表更新（隔离小改动）

- [ ] Task 1: 主页常量 + 测试 + i18n
  - [ ] SubTask 1.1: `web/default/src/features/home/constants.ts` 修改 `LANDING_MODEL_ROWS`：移除 `{ name: 'GLM 5.2', note: 'Pay per token' }`，新增 `{ name: 'DeepSeek V4 Flash', note: 'Pay per token' }`（置于 DeepSeek V4 Pro 之后）。最终顺序：DeepSeek V4 Pro / DeepSeek V4 Flash / Nemotron 3 Ultra / Laguna XS 2.1
  - [ ] SubTask 1.2: `web/default/src/features/home/constants.test.ts` 重写 `deepEqual` 断言匹配新数组；`doesNotMatch` 正则改为 `/GPT|Claude|Gemini|GLM/i`（加入 GLM）
  - [ ] SubTask 1.3: `web/default/src/i18n/locales/en.json` / `zh.json` / `zh-TW.json` / `fr.json` / `ru.json` / `ja.json` / `vi.json` 新增 key `DeepSeek V4 Flash`（值 = 模型名本身，与其他模型 key 同模式）；可选删除 `GLM 5.2` key
  - [ ] SubTask 1.4: `cd web/default && bun run i18n:sync` 同步，确认无缺失 key
  - [ ] SubTask 1.5: `cd web/default && bun test features/home/constants.test.ts` 通过
  - [ ] SubTask 1.6: `cd web/default && bun run typecheck` 通过
  - [ ] SubTask 1.7: commit `feat(home): replace GLM 5.2 with DeepSeek V4 Flash in landing model list`

---

## Phase 2: 数据模型 + 计费注入（套餐内模型 $0）

- [ ] Task 2: `SubscriptionPlan` / `UserSubscription` / `SubscriptionOrder` 新增字段
  - [ ] SubTask 2.1: `model/subscription.go` `SubscriptionPlan` struct（line 146-190）新增字段：
    - `BillingMode string \`json:"billing_mode" gorm:"type:varchar(16);default:'quota'"\``（`'quota'` = 现有配额制；`'model_coverage'` = 套餐内模型 $0 计费）
    - `AllowedModels string \`json:"allowed_models" gorm:"type:text;default:''"\``（逗号分隔模型 ID 列表）
  - [ ] SubTask 2.2: `model/subscription.go` `SubscriptionPlan.NormalizeDefaults()`（line 204-211）末尾追加：`if p.BillingMode == "" { p.BillingMode = "quota" }`
  - [ ] SubTask 2.3: `model/subscription.go` `UserSubscription` struct（line 253-281）新增字段：
    - `StripeSubscriptionId string \`json:"stripe_subscription_id" gorm:"type:varchar(64);default:'';index"\``
    - `StripeCustomerId string \`json:"stripe_customer_id" gorm:"type:varchar(64);default:'';index"\``
    - `CancelAtPeriodEnd bool \`json:"cancel_at_period_end"\``（不写 gorm default tag，Go 零值 false）
  - [ ] SubTask 2.4: `model/subscription.go` `SubscriptionOrder` struct（line 214-228）新增字段：
    - `StripeSessionId string \`json:"stripe_session_id" gorm:"type:varchar(64);default:'';index"\``
    - `PromotionCodeId string \`json:"promotion_code_id" gorm:"type:varchar(64);default:''"\``
    - `CouponId string \`json:"coupon_id" gorm:"type:varchar(64);default:''"\``
    - `OriginalAmountCents int64 \`json:"original_amount_cents" gorm:"type:bigint;default:0"\``
    - `DiscountAmountCents int64 \`json:"discount_amount_cents" gorm:"type:bigint;default:0"\``
    - `PaidAmountCents int64 \`json:"paid_amount_cents" gorm:"type:bigint;default:0"\``
    - `PaidCurrency string \`json:"paid_currency" gorm:"type:varchar(8);default:''"\``
  - [ ] SubTask 2.5: 新增 `SubscriptionPlan` 状态常量（紧邻 struct 定义后）：
    ```go
    const (
        SubscriptionBillingModeQuota         = "quota"
        SubscriptionBillingModeModelCoverage = "model_coverage"
    )
    ```
  - [ ] SubTask 2.6: 新增 `SubscriptionPlanStatus` 扩展常量（紧邻既有 `SubscriptionStatusActive` 等常量）：
    ```go
    const (
        SubscriptionStatusCanceling     = "canceling"
        SubscriptionStatusPastDue       = "past_due"
        SubscriptionStatusPaymentFailed = "payment_failed"
    )
    ```
  - [ ] SubTask 2.7: `go build ./...` 通过（GORM AutoMigrate 会自动 ALTER TABLE ADD COLUMN，三库兼容）

- [ ] Task 3: `UserSetting.BillingCurrency` + `SubscriptionPlan.StripePriceIdByCurrency`
  - [ ] SubTask 3.1: `dto/user_setting.go`（找到 `UserSetting` struct）新增 `BillingCurrency string \`json:"billing_currency"\``
  - [ ] SubTask 3.2: `model/subscription.go` 新增方法：
    ```go
    // StripePriceIdByCurrency 解析 StripePriceId 字段。
    // 兼容三种格式：
    //   1. JSON {"USD":"price_xxx","CNY":"price_yyy"}
    //   2. 旧单字符串 "price_xxx"（视为 USD）
    //   3. 空字符串
    // currency 为空或非 USD/CNY 时默认 USD。
    func (p *SubscriptionPlan) StripePriceIdByCurrency(currency string) string {
        if p.StripePriceId == "" {
            return ""
        }
        cur := strings.ToUpper(strings.TrimSpace(currency))
        if cur != "USD" && cur != "CNY" {
            cur = "USD"
        }
        // 尝试 JSON 解析
        var priceMap map[string]string
        if err := common.UnmarshalJsonStr(p.StripePriceId, &priceMap); err == nil {
            if id, ok := priceMap[cur]; ok {
                return id
            }
            // 回退到 USD
            if id, ok := priceMap["USD"]; ok {
                return id
            }
            return ""
        }
        // 旧单字符串格式：仅当请求 USD 时返回
        if cur == "USD" {
            return p.StripePriceId
        }
        return ""
    }
    ```
  - [ ] SubTask 3.3: `go build ./...` 通过

- [ ] Task 4: `UserActiveSubscriptionCoversModel` model 层查询
  - [ ] SubTask 4.1: `model/subscription.go` 紧邻 `HasActiveUserSubscription`（line 840）新增函数：
    ```go
    // UserActiveSubscriptionCoversModel 检查用户是否有覆盖指定模型的有效订阅。
    // 仅当 plan.BillingMode == 'model_coverage' 且 modelName 在 plan.AllowedModels 中时返回 ok=true。
    // modelName 通过 ratio_setting.FormatMatchingModelName 归一化后匹配。
    func UserActiveSubscriptionCoversModel(userId int, modelName string) (subId int, planId int, planTitle string, ok bool, err error) {
        if userId <= 0 || modelName == "" {
            return 0, 0, "", false, nil
        }
        now := common.GetTimestamp()
        var subs []UserSubscription
        if err := DB.Where("user_id = ? AND status = ? AND end_time > ?",
            userId, SubscriptionStatusActive, now).Find(&subs).Error; err != nil {
            return 0, 0, "", false, err
        }
        normalized := ratio_setting.FormatMatchingModelName(modelName)
        for _, sub := range subs {
            plan, pErr := GetSubscriptionPlanById(sub.PlanId)
            if pErr != nil || plan == nil {
                continue
            }
            if plan.BillingMode != SubscriptionBillingModeModelCoverage {
                continue
            }
            if planModelCovers(plan.AllowedModels, modelName, normalized) {
                return sub.Id, plan.Id, plan.Title, true, nil
            }
        }
        return 0, 0, "", false, nil
    }

    // planModelCovers 检查 allowedModels（逗号分隔）是否包含 modelName。
    // 同时匹配原始名和归一化名。
    func planModelCovers(allowedModels, modelName, normalized string) bool {
        if allowedModels == "" || modelName == "" {
            return false
        }
        for _, raw := range strings.Split(allowedModels, ",") {
            m := strings.TrimSpace(raw)
            if m == "" {
                continue
            }
            if m == modelName {
                return true
            }
            if ratio_setting.FormatMatchingModelName(m) == normalized {
                return true
            }
        }
        return false
    }
    ```
  - [ ] SubTask 4.2: `model/subscription.go` import 增加 `github.com/QuantumNous/new-api/setting/ratio_setting`（若未导入）
  - [ ] SubTask 4.3: `go build ./...` 通过

- [ ] Task 5: `ModelCoverageFunding` 资金源实现
  - [ ] SubTask 5.1: `service/funding_source.go` 新增 `ModelCoverageFunding` struct 与方法：
    ```go
    // ModelCoverageFunding 套餐内模型覆盖资金源。
    // PreConsume/Settle/Refund 全部 no-op，不扣任何配额。
    // Source() 返回 "subscription_covered"。
    type ModelCoverageFunding struct {
        subscriptionId int
        planId         int
        planTitle      string
    }

    func NewModelCoverageFunding(subscriptionId, planId int, planTitle string) *ModelCoverageFunding {
        return &ModelCoverageFunding{subscriptionId: subscriptionId, planId: planId, planTitle: planTitle}
    }

    func (m *ModelCoverageFunding) Source() string { return BillingSourceSubscriptionCovered }
    func (m *ModelCoverageFunding) PreConsume(amount int) error { return nil }
    func (m *ModelCoverageFunding) Settle(delta int) error { return nil }
    func (m *ModelCoverageFunding) Refund() error { return nil }
    ```
  - [ ] SubTask 5.2: `service/billing_session.go`（或 `service/funding_source.go` 顶部常量区）新增 `BillingSourceSubscriptionCovered = "subscription_covered"` 常量（紧邻既有 `BillingSourceWallet` / `BillingSourceSubscription`）
  - [ ] SubTask 5.3: `go build ./...` 通过

- [ ] Task 6: `NewBillingSession` 注入 model_coverage 分支
  - [ ] SubTask 6.1: `service/billing_session.go:342` `NewBillingSession` 在 `pref := common.NormalizeBillingPreference(...)`（line 347）之后、`tryWallet` 闭包定义之前，插入：
    ```go
    // 套餐内模型覆盖：若用户有有效订阅且模型属于该订阅套餐，则 $0 计费
    if subId, planId, planTitle, covered, coverErr := model.UserActiveSubscriptionCoversModel(
        relayInfo.UserId, relayInfo.OriginModelName,
    ); coverErr == nil && covered {
        session := &BillingSession{
            relayInfo:        relayInfo,
            funding:          NewModelCoverageFunding(subId, planId, planTitle),
            preConsumedQuota: 0,
            trusted:          true,
        }
        syncSubscriptionCoveredInfo(relayInfo, subId, planId, planTitle)
        return session, nil
    }
    ```
  - [ ] SubTask 6.2: `service/billing_session.go` 新增辅助函数 `syncSubscriptionCoveredInfo`：
    ```go
    // syncSubscriptionCoveredInfo 把套餐覆盖信息写入 relayInfo，供日志展示。
    func syncSubscriptionCoveredInfo(relayInfo *relaycommon.RelayInfo, subId, planId int, planTitle string) {
        if relayInfo == nil {
            return
        }
        relayInfo.BillingSource = BillingSourceSubscriptionCovered
        relayInfo.SubscriptionId = subId
        relayInfo.SubscriptionPlanId = planId
        relayInfo.SubscriptionPlanTitle = planTitle
        // 覆盖模式下不消耗配额
        relayInfo.SubscriptionAmountTotal = 0
        relayInfo.SubscriptionAmountUsedAfterPreConsume = 0
    }
    ```
  - [ ] SubTask 6.3: 验证 `relaycommon.RelayInfo` 已有 `BillingSource` / `SubscriptionId` / `SubscriptionPlanId` / `SubscriptionPlanTitle` / `SubscriptionAmountTotal` / `SubscriptionAmountUsedAfterPreConsume` 字段（由 `service/billing_session.go:318 syncRelayInfo` 使用）。若字段名不同，按实际字段名调整。
  - [ ] SubTask 6.4: `go build ./...` 通过

- [ ] Task 7: 异步任务路径守卫
  - [ ] SubTask 7.1: `service/task_billing.go:90` `taskAdjustFunding` 函数顶部新增分支：
    ```go
    if task.PrivateData.BillingSource == BillingSourceSubscriptionCovered {
        // 套餐覆盖模型：不调整任何资金源，记录使用即可
        return nil
    }
    ```
  - [ ] SubTask 7.2: 验证 `task.PrivateData.BillingSource` 字段存在（由 `relayInfo.BillingSource` 在任务创建时快照）。若字段路径不同，按实际调整。
  - [ ] SubTask 7.3: `go build ./...` 通过

- [ ] Task 8: 日志记录套餐覆盖信息
  - [ ] SubTask 8.1: `service/log_info_generate.go:159` `appendBillingInfo` 函数末尾追加：当 `relayInfo.BillingSource == BillingSourceSubscriptionCovered` 时，写入 `other["billing_source"] = "subscription_covered"`、`other["subscription_id"]`、`other["subscription_plan_id"]`、`other["subscription_plan_title"]`、`other["wallet_quota_deducted"] = 0`（既有 `appendBillingInfo` 已写其他字段，此处补齐 covered 分支）
  - [ ] SubTask 8.2: 确认 `model.RecordConsumeLog` 在 `Quota == 0` 时仍写日志（`model/log.go:343 RecordConsumeLog` 仅由 `common.LogConsumeEnabled` 守卫，与 quota 值无关）—— 已验证，无需修改
  - [ ] SubTask 8.3: `go build ./...` 通过

- [ ] Task 9: Phase 2 后端测试
  - [ ] SubTask 9.1: 新建 `model/subscription_covers_test.go`：
    - `TestUserActiveSubscriptionCoversModel_Hit`：构造 user + plan（BillingMode=model_coverage, AllowedModels="gpt-4,claude-3-opus"）+ active subscription，调用 `UserActiveSubscriptionCoversModel(userId, "gpt-4")` → ok=true
    - `TestUserActiveSubscriptionCoversModel_Miss`：同上但传 "gpt-3.5" → ok=false
    - `TestUserActiveSubscriptionCoversModel_QuotaPlanIgnored`：plan.BillingMode='quota' 时即使模型在 AllowedModels 也 ok=false
    - `TestUserActiveSubscriptionCoversModel_ExpiredSubIgnored`：subscription.end_time < now → ok=false
    - `TestUserActiveSubscriptionCoversModel_NormalizedMatch`：AllowedModels="gemini-2.5-flash-thinking-001"，传 "gemini-2.5-flash-thinking-002" → 归一化后 ok=true
  - [ ] SubTask 9.2: 新建 `service/billing_session_model_coverage_test.go`：
    - `TestNewBillingSession_ModelCoverage`：mock `model.UserActiveSubscriptionCoversModel` 返回 covered → `NewBillingSession` 返回 `BillingSession`，`funding.Source() == "subscription_covered"`，`PreConsume(100)` 不扣 quota，`Settle(1000)` 不扣 quota
  - [ ] SubTask 9.3: 测试遵循 AGENTS.md：`testify/require` 用于 setup + fatal，`testify/assert` 用于非致命；显式初始化 DB / user / plan / subscription
  - [ ] SubTask 9.4: `go test ./model/... -run 'UserActiveSubscriptionCoversModel'` 通过
  - [ ] SubTask 9.5: `go test ./service/... -run 'ModelCoverage'` 通过

---

## Phase 3: 后端 Stripe Checkout 路由

- [ ] Task 10: 公开套餐列表 + 用户订阅详情路由
  - [ ] SubTask 10.1: `controller/subscription.go` 新增 `GetSubscriptionPlansPublic`：
    ```go
    // GetSubscriptionPlansPublic 返回启用的套餐（公开字段，匿名可访问）。
    // 过滤掉内部 ID、AllowedModels（前端不能决定模型归属）等敏感字段。
    func GetSubscriptionPlansPublic(c *gin.Context) {
        plans, err := model.GetEnabledSubscriptionPlans()
        if err != nil {
            common.ApiError(c, err)
            return
        }
        type publicPlan struct {
            Id            int     `json:"id"`
            Title         string  `json:"title"`
            Subtitle      string  `json:"subtitle"`
            PriceAmount   float64 `json:"price_amount"`
            Currency      string  `json:"currency"`
            DurationUnit  string  `json:"duration_unit"`
            DurationValue int     `json:"duration_value"`
            BillingMode   string  `json:"billing_mode"`
        }
        var result []publicPlan
        for _, p := range plans {
            result = append(result, publicPlan{
                Id: p.Id, Title: p.Title, Subtitle: p.Subtitle,
                PriceAmount: p.PriceAmount, Currency: p.Currency,
                DurationUnit: p.DurationUnit, DurationValue: p.DurationValue,
                BillingMode: p.BillingMode,
            })
        }
        common.ApiSuccess(c, result)
    }
    ```
  - [ ] SubTask 10.2: `controller/subscription.go` 新增 `GetSubscriptionSelfFull`：返回当前用户订阅 + 状态 + 续费日 + cancel_at_period_end + plan 信息 + Stripe Portal 可用性。从 `UserSubscription` + `SubscriptionPlan` join 返回。
  - [ ] SubTask 10.3: `router/api-router.go` 在既有 `subscriptionRoute`（line 200）组外注册公开路由：
    ```go
    // 公开套餐列表（匿名可访问）
    apiRouter.GET("/subscription/plans/public", controller.GetSubscriptionPlansPublic)
    ```
  - [ ] SubTask 10.4: `router/api-router.go` 在 `subscriptionRoute` 组内（line 211 之前）注册：
    ```go
    subscriptionRoute.GET("/self/full", controller.GetSubscriptionSelfFull)
    ```
  - [ ] SubTask 10.5: `go build ./...` 通过

- [ ] Task 11: 优惠码验证路由
  - [ ] SubTask 11.1: 新建 `controller/subscription_stripe_promo.go`：
    ```go
    type ValidatePromoRequest struct {
        Code     string `json:"code"`
        PlanId   int    `json:"plan_id"`
        Currency string `json:"currency"` // USD / CNY
    }

    type ValidatePromoResponse struct {
        Valid              bool    `json:"valid"`
        CouponId           string  `json:"coupon_id"`
        PromotionCodeId    string  `json:"promotion_code_id"`
        PercentOff         float64 `json:"percent_off"`
        Duration           string  `json:"duration"`           // once / repeating / forever
        DurationInMonths   int     `json:"duration_in_months"`
        OriginalAmountCents int64  `json:"original_amount_cents"`
        DiscountAmountCents int64  `json:"discount_amount_cents"`
        DueTodayCents       int64  `json:"due_today_cents"`
        NextRenewalCents    int64  `json:"next_renewal_cents"`
        Currency           string  `json:"currency"`
        DurationLabel      string  `json:"duration_label"`      // "首月" / "N 个月" / "永久"
    }
    ```
  - [ ] SubTask 11.2: 实现 `ValidatePromo` 控制器：
    - 校验 `setting.StripeApiSecret` 已配置
    - `stripe.Key = setting.StripeApiSecret`
    - `promotioncode.List(&stripe.PromotionCodeListParams{Code: stripe.String(req.Code), Active: stripe.Bool(true)})` 查找 active promotion code
    - 未找到 → `common.ApiErrorMsg(c, "优惠码无效或已过期")`
    - 找到 → 读 `coupon`（通过 `promotioncode.Coupon` 展开）：`percent_off` / `duration` / `duration_in_months`
    - 校验 `coupon.Valid`、`coupon.Redemptions` 未达 `coupon.MaxRedemptions`、`coupon RedeemBy` 未过期
    - 按 `plan.PriceAmount` + `currency` 计算 `OriginalAmountCents`（如 ¥99 → 9900 cents；$19 → 1900 cents）
    - `DiscountAmountCents = OriginalAmountCents * percent_off / 100`
    - `DueTodayCents = OriginalAmountCents - DiscountAmountCents`
    - `NextRenewalCents = OriginalAmountCents`（若 duration=forever 则 = DueTodayCents）
    - `DurationLabel`：once → "仅首月"；repeating → fmt.Sprintf("持续 %d 个月", DurationInMonths)；forever → "永久"
    - 返回 `ValidatePromoResponse`
  - [ ] SubTask 11.3: `router/api-router.go` 在 `subscriptionRoute` 组内注册：
    ```go
    subscriptionRoute.POST("/stripe/validate-promo", middleware.CriticalRateLimit(), controller.ValidatePromo)
    ```
  - [ ] SubTask 11.4: `go build ./...` 通过

- [ ] Task 12: Checkout Session 创建路由（防重复订阅）
  - [ ] SubTask 12.1: `controller/subscription_payment_stripe.go` 新增 `SubscriptionRequestStripeCheckout`：
    ```go
    type SubscriptionStripeCheckoutRequest struct {
        PlanId         int    `json:"plan_id"`
        Currency       string `json:"currency"`        // USD / CNY
        PromotionCode  string `json:"promotion_code"`  // 可选，用户输入的码（不是 coupon ID）
    }
    ```
  - [ ] SubTask 12.2: 实现逻辑：
    1. 校验 plan 存在 + enabled + StripePriceIdByCurrency(currency) 非空
    2. 校验 `setting.StripeApiSecret` / `setting.StripeWebhookSecret` 已配置
    3. **防重复订阅**：`model.HasActiveUserSubscription(userId)` → 已有 → 返回 `{"error": "already_subscribed", "manage_url": "/account/subscription"}`（HTTP 200，business error）
    4. **防重复点击**：查 `SubscriptionOrder` 中 user_id + plan_id + status='pending' + create_time > now-300s → 复用其 `StripeSessionId` 重建 session URL（Stripe Checkout Session URL 可通过 `session.Get` 重新获取）→ 返回 `pay_link`
    5. 若传了 `PromotionCode` → 调 `ValidatePromo` 内部逻辑拿 `couponId` + `promotionCodeId`
    6. 创建 `SubscriptionOrder`（status=pending, PaymentMethod=stripe, PaymentProvider=stripe, Money=plan.PriceAmount, OriginalAmountCents/DiscountAmountCents/PaidCurrency 预填）
    7. 调 `genStripeSubscriptionCheckoutLink`（新函数）创建 Checkout Session
    8. 把 `session.Id` 写回 `order.StripeSessionId`
    9. 返回 `{"pay_link": session.URL}`
  - [ ] SubTask 12.3: 新增 `genStripeSubscriptionCheckoutLink`：
    ```go
    func genStripeSubscriptionCheckoutLink(referenceId, customerId, email, priceId, couponId, successURL, cancelURL string, userId, planId int) (*stripe.CheckoutSession, error) {
        stripe.Key = setting.StripeApiSecret
        params := &stripe.CheckoutSessionParams{
            Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
            ClientReferenceID: stripe.String(referenceId),
            LineItems: []*stripe.CheckoutSessionLineItemParams{{
                Price:    stripe.String(priceId),
                Quantity: stripe.Int64(1),
            }},
            SuccessURL: stripe.String(successURL),
            CancelURL:  stripe.String(cancelURL),
            SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
                Metadata: map[string]string{
                    "user_id":  strconv.Itoa(userId),
                    "plan_id":  strconv.Itoa(planId),
                    "trade_no": referenceId,
                },
            },
        }
        if customerId != "" {
            params.Customer = stripe.String(customerId)
        } else if email != "" {
            params.CustomerEmail = stripe.String(email)
        }
        if couponId != "" {
            params.Discounts = []*stripe.CheckoutSessionDiscountParams{{
                Coupon: stripe.String(couponId),
            }}
        }
        return session.New(params)
    }
    ```
  - [ ] SubTask 12.4: `router/api-router.go` 在 `subscriptionRoute` 组内注册：
    ```go
    subscriptionRoute.POST("/stripe/checkout", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripeCheckout)
    ```
  - [ ] SubTask 12.5: `go build ./...` 通过

- [ ] Task 13: Stripe Customer Portal 路由
  - [ ] SubTask 13.1: `controller/subscription_payment_stripe.go` 新增 `SubscriptionStripePortal`：
    ```go
    func SubscriptionStripePortal(c *gin.Context) {
        userId := c.GetInt("id")
        user, err := model.GetUserById(userId, false)
        if err != nil || user == nil || user.StripeCustomer == "" {
            common.ApiErrorMsg(c, "未找到 Stripe 客户")
            return
        }
        stripe.Key = setting.StripeApiSecret
        params := &stripe.BillingPortalSessionParams{
            Customer:  stripe.String(user.StripeCustomer),
            ReturnURL: stripe.String(paymentReturnPath("/account/subscription")),
        }
        ps, err := billingportalSession.New(params)
        if err != nil {
            common.ApiError(c, err)
            return
        }
        common.ApiSuccess(c, gin.H{"url": ps.URL})
    }
    ```
  - [ ] SubTask 13.2: import `github.com/stripe/stripe-go/v85/billingportal/session`（注意 alias 避免与 checkout/session 冲突：`billingportalSession "github.com/stripe/stripe-go/v85/billingportal/session"`)
  - [ ] SubTask 13.3: `router/api-router.go` 注册：
    ```go
    subscriptionRoute.GET("/stripe/portal", controller.SubscriptionStripePortal)
    ```
  - [ ] SubTask 13.4: `go build ./...` 通过

---

## Phase 4: Webhook 处理

- [ ] Task 14: 订阅 Stripe Webhook 控制器
  - [ ] SubTask 14.1: 新建 `controller/subscription_stripe_webhook.go`：
    ```go
    package controller

    import (
        "context"
        "io"
        "net/http"

        "github.com/QuantumNous/new-api/setting"
        "github.com/gin-gonic/gin"
        "github.com/stripe/stripe-go/v85"
        "github.com/stripe/stripe-go/v85/webhook"
    )

    func SubscriptionStripeWebhook(c *gin.Context) {
        if setting.StripeWebhookSecret == "" {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        payload, err := io.ReadAll(c.Request.Body)
        if err != nil {
            c.AbortWithStatus(http.StatusBadRequest)
            return
        }
        signature := c.GetHeader("Stripe-Signature")
        event, err := webhook.ConstructEventWithOptions(payload, signature, setting.StripeWebhookSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
        if err != nil {
            c.AbortWithStatus(http.StatusBadRequest)
            return
        }
        // 幂等：复用现有 stripe_webhook_events 表
        inserted, err := model.TryInsertStripeWebhookEvent(event.ID, event.Type)
        if err != nil {
            c.Status(http.StatusOK) // 已处理过，幂等返回 200
            return
        }
        if !inserted {
            c.Status(http.StatusOK)
            return
        }
        ctx := c.Request.Context()
        service.ProcessSubscriptionStripeEvent(ctx, event)
        c.Status(http.StatusOK)
    }
    ```
  - [ ] SubTask 14.2: `router/api-router.go` 在 webhook 公开区（line 57-90 附近）注册：
    ```go
    apiRouter.POST("/stripe/subscription/webhook", anonymousRequestBodyLimit, controller.SubscriptionStripeWebhook)
    ```
  - [ ] SubTask 14.3: `go build ./...` 通过

- [ ] Task 15: Webhook 事件处理 service
  - [ ] SubTask 15.1: 新建 `service/subscription_stripe_webhook.go`：
    ```go
    package service

    import (
        "context"
        "github.com/stripe/stripe-go/v85"
    )

    func ProcessSubscriptionStripeEvent(ctx context.Context, event stripe.Event) {
        switch event.Type {
        case "checkout.session.completed":
            handleSubCheckoutSessionCompleted(ctx, event)
        case "invoice.paid":
            handleSubInvoicePaid(ctx, event)
        case "invoice.payment_failed":
            handleSubInvoicePaymentFailed(ctx, event)
        case "customer.subscription.updated":
            handleSubCustomerSubscriptionUpdated(ctx, event)
        case "customer.subscription.deleted":
            handleSubCustomerSubscriptionDeleted(ctx, event)
        case "charge.refunded":
            handleSubChargeRefunded(ctx, event)
        }
    }
    ```
  - [ ] SubTask 15.2: 实现 `handleSubCheckoutSessionCompleted`：
    - 解析 `stripe.CheckoutSession` from event.Data.Raw
    - 仅处理 `session.Mode == "subscription"`
    - 从 `session.Subscription`（subscription ID）调 `subscription.Get` 拿 subscription
    - 从 `session.ClientReferenceID` 或 `subscription.Metadata["trade_no"]` 查 `SubscriptionOrder`
    - 写回 `order.StripeSessionId = session.ID`、`order.StripeSubscriptionId = subscription.ID`、`order.StripeCustomerId = session.Customer`
    - 若用户 `User.StripeCustomer` 为空，写入 `session.Customer`（同 `model/topup.go:154` 模式）
    - **不激活订阅**（等 `invoice.paid`）
  - [ ] SubTask 15.3: 实现 `handleSubInvoicePaid`：
    - 解析 `stripe.Invoice`
    - 从 `invoice.Subscription` 拿 subscription ID
    - 查 `UserSubscription` by `StripeSubscriptionId`
    - 若不存在（首次付款）：
      - 从 `invoice.Metadata["trade_no"]` 或 `Lines.Data[0].Metadata["trade_no"]` 查 `SubscriptionOrder`
      - 从 `order.PlanId` 加载 `SubscriptionPlan`
      - 用 `calcPlanEndTime(now, plan)` 算 `EndTime`
      - 创建 `UserSubscription{UserId, PlanId, Status='active', StartTime=now, EndTime, StripeSubscriptionId, StripeCustomerId, AmountTotal=0, AmountUsed=0, CancelAtPeriodEnd=false, Source='order'}`
      - 落 `order.PaidAmountCents = invoice.AmountPaid`、`order.PaidCurrency = invoice.Currency`
      - 触发 affiliate：`model.SettleSubscriptionCommission(tx, order.UserId, order.TradeNo, order.PaidAmountCents, order.PaidCurrency)`（Phase 5 实现）
    - 若存在（续费）：
      - 更新 `EndTime = subscription.CurrentPeriodEnd`
      - 若 `Status != 'active'` → 恢复为 `'active'`
      - 落 `order.PaidAmountCents`（新订单行，或更新原订单）
      - 触发 affiliate 续费分佣
  - [ ] SubTask 15.4: 实现 `handleSubInvoicePaymentFailed`：
    - 查 `UserSubscription` by `StripeSubscriptionId`（从 `invoice.Subscription`）
    - 更新 `Status = 'past_due'`
  - [ ] SubTask 15.5: 实现 `handleSubCustomerSubscriptionUpdated`：
    - 解析 `stripe.Subscription`
    - 查 `UserSubscription` by `StripeSubscriptionId`
    - 同步 `CancelAtPeriodEnd = subscription.CancelAtPeriodEnd`
    - 同步 `EndTime = subscription.CurrentPeriodEnd`
    - 状态映射（见 spec 表格）：active+cancel_at_period_end=true → 'canceling'；active+false → 'active'；past_due → 'past_due'；unpaid/incomplete → 'payment_failed'
  - [ ] SubTask 15.6: 实现 `handleSubCustomerSubscriptionDeleted`：
    - 查 `UserSubscription` by `StripeSubscriptionId`
    - 更新 `Status = 'canceled'`、`EndTime = now`（使用既有 `SubscriptionStatusCancelled = "cancelled"` 拼写，与现有常量一致）
    - 若订阅有对应 `SubscriptionOrder` 且 Commission 在冻结期 → 触发 `model.RevertCommissionForSubscription(orderId)`（Phase 5 实现）
  - [ ] SubTask 15.7: 实现 `handleSubChargeRefunded`：
    - 解析 `stripe.Charge`
    - 从 `charge.Invoice` 拿 invoice ID → 查 invoice → 拿 subscription ID
    - 查对应 `SubscriptionOrder`（按 `StripeSubscriptionId` 或 `trade_no`）
    - 调 `model.RevertCommissionForSubscription(orderId)`（Phase 5 实现）
  - [ ] SubTask 15.8: `go build ./...` 通过

- [ ] Task 16: Webhook 测试
  - [ ] SubTask 16.1: 新建 `controller/subscription_stripe_webhook_test.go`：
    - `TestSubscriptionStripeWebhook_NoSecret`：`setting.StripeWebhookSecret = ""` → HTTP 403
    - `TestSubscriptionStripeWebhook_InvalidSignature`：错误签名 → HTTP 400
    - `TestSubscriptionStripeWebhook_DuplicateEventId`：同一 event.ID 两次调用 → 第二次仍返回 200，handler 不重复执行
  - [ ] SubTask 16.2: 新建 `service/subscription_stripe_webhook_test.go`：
    - `TestHandleSubInvoicePaid_FirstPayment`：构造 mock invoice event → 创建 UserSubscription active + 触发分佣
    - `TestHandleSubInvoicePaid_Renewal`：已有 UserSubscription → 更新 EndTime + 触发续费分佣
    - `TestHandleSubCustomerSubscriptionUpdated_CancelAtPeriodEnd`：mock subscription event with cancel_at_period_end=true → UserSubscription.Status='canceling'
  - [ ] SubTask 16.3: `go test ./controller/... -run 'SubscriptionStripeWebhook'` 通过
  - [ ] SubTask 16.4: `go test ./service/... -run 'HandleSub'` 通过

---

## Phase 5: Affiliate 接入

- [ ] Task 17: `SettleSubscriptionCommission` model 函数
  - [ ] SubTask 17.1: `model/commission.go` 紧邻 `SettleRechargeCommission`（line 131）新增：
    ```go
    // SettleSubscriptionCommission 结算订阅付款的 affiliate 现金佣金。
    // 复用 SettleRechargeCommission 同款逻辑，TopUpId 字段复用为订阅订单号。
    // 幂等键 (topup_id, inviter_id) 保证同订单不重复结算。
    func SettleSubscriptionCommission(tx *gorm.DB, inviteeId int, orderId string, paidAmountCents int64, paidCurrency string) (settled bool, err error) {
        return settleCommissionInternal(tx, inviteeId, orderId, paidAmountCents, paidCurrency, "subscription_payment")
    }
    ```
  - [ ] SubTask 17.2: 若 `SettleRechargeCommission` 内部逻辑可直接复用，提取 `settleCommissionInternal(tx, inviteeId, sourceId, paidAmountCents, paidCurrency, source string)` 私有函数，让 `SettleRechargeCommission` 与 `SettleSubscriptionCommission` 都委托给它。`source` 仅作日志区分。
  - [ ] SubTask 17.3: `go build ./...` 通过

- [ ] Task 18: `RevertCommissionForSubscription` model 函数
  - [ ] SubTask 18.1: `model/commission.go` 新增：
    ```go
    // RevertCommissionForSubscription 撤销订阅订单对应的 affiliate 佣金。
    // 若佣金在 pending（冻结中）→ 从 PendingCommissionCents 扣回，改 status=reverted。
    // 若已 available（解冻）→ 从 CommissionBalanceCents 扣回（可能为负，下次佣金优先补回）。
    func RevertCommissionForSubscription(orderId string) error {
        // 找该订单对应的 Commission 行（TopUpId = orderId）
        var comms []Commission
        if err := DB.Where("topup_id = ? AND status IN ?", orderId, []string{CommissionStatusPending, CommissionStatusAvailable}).Find(&comms).Error; err != nil {
            return err
        }
        for _, c := range comms {
            if err := revertSingleCommission(DB, c); err != nil {
                return err
            }
        }
        return nil
    }
    ```
  - [ ] SubTask 18.2: 实现 `revertSingleCommission`：按 status 分支扣减 inviter 的 `PendingCommissionCents` 或 `CommissionBalanceCents`，落 `RevertedAt` / `RevertReason='subscription_refund'`，改 status='reverted'
  - [ ] SubTask 18.3: `go build ./...` 通过

- [ ] Task 19: Webhook 集成 affiliate 调用
  - [ ] SubTask 19.1: `service/subscription_stripe_webhook.go` `handleSubInvoicePaid` 在创建/续期 `UserSubscription` 后，调 `model.SettleSubscriptionCommission(tx, order.UserId, order.TradeNo, order.PaidAmountCents, order.PaidCurrency)`（在同一个 GORM 事务内）
  - [ ] SubTask 19.2: `service/subscription_stripe_webhook.go` `handleSubCustomerSubscriptionDeleted` 与 `handleSubChargeRefunded` 调 `model.RevertCommissionForSubscription(orderId)`
  - [ ] SubTask 19.3: `go build ./...` 通过

- [ ] Task 20: Affiliate 测试
  - [ ] SubTask 20.1: `model/commission_test.go` 新增：
    - `TestSettleSubscriptionCommission_ApprovedInviter`：获批 inviter + 订阅付款 9900 美分（¥99）→ 佣金 2475 美分入 PendingCommissionCents
    - `TestSettleSubscriptionCommission_Idempotent`：同 orderId 重复调用 → 第二次 settled=false
    - `TestRevertCommissionForSubscription_Pending`：佣金 pending → revert → PendingCommissionCents 扣回，status=reverted
    - `TestRevertCommissionForSubscription_Available`：佣金 available → revert → CommissionBalanceCents 扣回（允许为负）
  - [ ] SubTask 20.2: `go test ./model/... -run 'SettleSubscription|RevertCommissionForSubscription'` 通过

---

## Phase 6: 前端 Plans 落地页

- [ ] Task 21: Navbar 链接 + i18n
  - [ ] SubTask 21.1: `web/default/src/hooks/use-top-nav-links.ts` 在 `Model Square` push 之后、`Rankings` 之前插入：
    ```ts
    links.push({ title: 'Plans', href: '/plans' })
    ```
  - [ ] SubTask 21.2: `web/default/src/i18n/locales/en.json` 新增 key（值=英文原文）：
    `Plans` / `Subscribe now` / `Current plan` / `Manage subscription` / `Update payment method` / `Reactivate` / `Cancel subscription` / `View invoices` / `Next renewal` / `Promo code` / `Apply` / `Original price` / `Discount` / `Due today` / `Next renewal amount` / `Auto-renews monthly` / `Cancel anytime` / `Already subscribed` / `Manage your subscription` / `Unlimited use of covered models` / `Covered models` / `Plan benefits`
  - [ ] SubTask 21.3: `zh.json` 对应翻译：`套餐` / `立即订阅` / `当前套餐` / `管理订阅` / `更新支付方式` / `重新激活` / `取消订阅` / `查看账单` / `下次续费` / `优惠码` / `应用` / `原价` / `优惠` / `本次支付` / `下次续费金额` / `每月自动续费` / `随时取消` / `已有订阅` / `管理你的订阅` / `套餐内模型不限量` / `套餐内模型` / `套餐权益`
  - [ ] SubTask 21.4: 其他 locale（zh-TW / fr / ru / ja / vi）按英文 key 补齐
  - [ ] SubTask 21.5: `cd web/default && bun run i18n:sync` 同步

- [ ] Task 22: Plans 特性模块 + 路由
  - [ ] SubTask 22.1: 新建 `web/default/src/features/plans/api.ts`：
    ```ts
    import { api } from '@/lib/api'

    export interface PublicPlan {
      id: number
      title: string
      subtitle: string
      price_amount: number
      currency: string
      duration_unit: string
      duration_value: number
      billing_mode: string
    }

    export interface SubscriptionStatus {
      has_active_subscription: boolean
      status: string
      plan_title: string
      price_amount: number
      currency: string
      end_time: number
      cancel_at_period_end: boolean
    }

    export interface PromoValidation {
      valid: boolean
      coupon_id: string
      promotion_code_id: string
      percent_off: number
      duration: string
      duration_in_months: number
      original_amount_cents: number
      discount_amount_cents: number
      due_today_cents: number
      next_renewal_cents: number
      currency: string
      duration_label: string
    }

    export interface CheckoutResponse {
      pay_link: string
    }

    export async function getPublicPlans() {
      const res = await api.get('/api/subscription/plans/public')
      return res.data as PublicPlan[]
    }

    export async function getSubscriptionStatus() {
      const res = await api.get('/api/subscription/self/full')
      return res.data as SubscriptionStatus
    }

    export async function validatePromo(code: string, planId: number, currency: string) {
      const res = await api.post('/api/subscription/stripe/validate-promo', { code, plan_id: planId, currency })
      return res.data as PromoValidation
    }

    export async function createCheckout(planId: number, currency: string, promoCode?: string) {
      const res = await api.post('/api/subscription/stripe/checkout', { plan_id: planId, currency, promotion_code: promoCode })
      return res.data as CheckoutResponse
    }

    export async function createPortalSession() {
      const res = await api.get('/api/subscription/stripe/portal')
      return res.data as { url: string }
    }
    ```
  - [ ] SubTask 22.2: 新建 `web/default/src/features/plans/hooks/use-plans.ts`：`useQuery({ queryKey: ['plans-public'], queryFn: getPublicPlans })`
  - [ ] SubTask 22.3: 新建 `web/default/src/features/plans/hooks/use-subscription-status.ts`：`useQuery({ queryKey: ['subscription-status'], queryFn: getSubscriptionStatus, enabled: !!auth.user })`
  - [ ] SubTask 22.4: 新建 `web/default/src/features/plans/hooks/use-promo.ts`：`useMutation` 调 `validatePromo`，返回 `{ validate, data, isPending, error }`
  - [ ] SubTask 22.5: 新建 `web/default/src/features/plans/hooks/use-checkout.ts`：`useMutation` 调 `createCheckout`，成功后 `window.location.href = data.pay_link`；`already_subscribed` 错误时跳 `/account/subscription`
  - [ ] SubTask 22.6: 新建 `web/default/src/features/plans/components/currency-switcher.tsx`：CNY/USD 切换器，状态存 localStorage + 写入 user setting
  - [ ] SubTask 22.7: 新建 `web/default/src/features/plans/components/subscription-status-badge.tsx`：根据 status 显示 badge（Active/Canceling/Canceled/Past due/Payment failed/Expired）
  - [ ] SubTask 22.8: 新建 `web/default/src/features/plans/components/plan-card.tsx`：展示 Nova Unlimited 套餐卡（标题、副标题、月费按币种、权益列表、Subscribe now / Current plan 按钮）
  - [ ] SubTask 22.9: 新建 `web/default/src/features/plans/components/promo-confirm-dialog.tsx`：`AlertDialog` 显示原价/优惠/本次支付/下次续费/优惠范围，确认后回填 Subscribe Modal
  - [ ] SubTask 22.10: 新建 `web/default/src/features/plans/components/subscribe-modal.tsx`：`Dialog` 含套餐名/月费/自动续费/随时取消说明/优惠码输入+Apply/原价/优惠/本次支付/下次续费/前往 Stripe Checkout 按钮。**不显示**月份数量、3/6/12 月选项、预付选项。按钮点击后 loading + 禁用。
  - [ ] SubTask 22.11: 新建 `web/default/src/features/plans/index.tsx`：页面组件，组合 `CurrencySwitcher` + `PlanCard` + `SubscribeModal` + `PromoConfirmDialog`。匿名访客也可看（已登录未订阅 → 弹 Modal；已订阅 → 显示 Current plan + Manage）。
  - [ ] SubTask 22.12: 新建 `web/default/src/routes/plans/index.tsx`：
    ```tsx
    import { createFileRoute } from '@tanstack/react-router'
    import { Plans } from '@/features/plans'
    export const Route = createFileRoute('/plans/')({ component: Plans })
    ```
  - [ ] SubTask 22.13: 运行 TanStack Router codegen（`cd web/default && bun run dev` 一次自动生成 `routeTree.gen.ts`，或 `bun run gen:routes` 若有）
  - [ ] SubTask 22.14: `cd web/default && bun run typecheck` 通过
  - [ ] SubTask 22.15: `cd web/default && bun run build` 通过
  - [ ] SubTask 22.16: 手动验证：访问 `/plans` 直接刷新不报错；移动端 navbar 显示 Plans；点击 Subscribe now 弹 Modal

---

## Phase 7: 前端订阅管理页

- [ ] Task 23: `/account/subscription` 路由 + 页面
  - [ ] SubTask 23.1: 新建 `web/default/src/features/account-subscription/index.tsx`：展示当前套餐 / 订阅状态 badge / 月费+币种 / 下次续费日期 / 是否已取消续费 / 当前优惠 / 操作按钮（Manage subscription / View invoices / Update payment method / Cancel subscription）
  - [ ] SubTask 23.1.1: Manage / View invoices / Update payment method 按钮 → 调 `createPortalSession()` 跳转返回的 URL（Stripe Customer Portal 配置不同入口）
  - [ ] SubTask 23.1.2: Cancel subscription → 调 `createPortalSession()`（让用户在 Stripe Portal 内取消）或调后端 cancel 接口（若新增）。**采用 Portal 方案**：复用 `createPortalSession()`，Stripe Portal 配置允许 cancel。
  - [ ] SubTask 23.1.3: Reactivate → 调 `createPortalSession()`（Stripe Portal 内可恢复 cancel_at_period_end 订阅）
  - [ ] SubTask 23.2: 复用 `web/default/src/features/plans/api.ts` 的 `getSubscriptionStatus` 与 `createPortalSession`，或单独建 `web/default/src/features/account-subscription/api.ts` 复用同一份 api（推荐复用 plans/api.ts 避免重复）
  - [ ] SubTask 23.3: 新建 `web/default/src/routes/_authenticated/account/subscription/index.tsx`：
    ```tsx
    import { createFileRoute } from '@tanstack/react-router'
    import { AccountSubscription } from '@/features/account-subscription'
    export const Route = createFileRoute('/_authenticated/account/subscription/')({ component: AccountSubscription })
    ```
  - [ ] SubTask 23.4: 运行 TanStack Router codegen
  - [ ] SubTask 23.5: `cd web/default && bun run typecheck` 通过
  - [ ] SubTask 23.6: `cd web/default && bun run build` 通过
  - [ ] SubTask 23.7: 手动验证：访问 `/account/subscription` 直接刷新不报错；未登录跳 `/sign-in`

---

## Phase 8: Stripe 配置文档 + 端到端验证

- [ ] Task 24: 管理员配置文档
  - [ ] SubTask 24.1: 在 `web/default/src/features/system-settings/integrations/` 现有 `payment-settings-section.tsx` 旁新增 `nova-unlimited-subscription-setup-guide.tsx`（只读说明组件）：列出 Stripe Dashboard 配置步骤（Product / Price USD / Price CNY / Coupon / Webhook endpoint / Customer Portal 启用），并在 admin 后台 system-settings 显示
  - [ ] SubTask 24.2: 文档说明 `SubscriptionPlan.StripePriceId` 字段需填 JSON `{"USD":"price_xxx","CNY":"price_yyy"}`；`BillingMode='model_coverage'`；`AllowedModels` 填逗号分隔模型 ID
  - [ ] SubTask 24.3: `cd web/default && bun run typecheck` 通过

- [ ] Task 25: 端到端验证
  - [ ] SubTask 25.1: `go build ./...` 通过
  - [ ] SubTask 25.2: `go test ./model/... ./service/... ./controller/...` 全通过
  - [ ] SubTask 25.3: `cd web/default && bun run typecheck && bun run build` 通过
  - [ ] SubTask 25.4: 手动 E2E（Stripe Test Mode）：
    1. 后台创建 SubscriptionPlan：title=Nova Unlimited, BillingMode=model_coverage, AllowedModels="deepseek-v4-pro,deepseek-v4-flash", StripePriceId='{"USD":"price_test_usd","CNY":"price_test_cny"}', PriceAmount=19, Currency=USD, DurationUnit=month, DurationValue=1
    2. 用户 A 访问 `/plans` → 看到 Nova Unlimited 卡片
    3. 点击 Subscribe now → Modal 弹出 → 输入测试优惠码 → Apply → Promo Confirm Dialog 显示折扣详情
    4. 点前往 Stripe Checkout → 跳转 Stripe Test Checkout → 用 4242 测试卡支付
    5. Webhook 触发 `invoice.paid` → UserSubscription 创建 active + Affiliate 分佣（若有 inviter）
    6. 用户 A 调用套餐内模型（deepseek-v4-pro）→ 计费 $0，使用记录有 log（billing_source=subscription_covered）
    7. 用户 A 调用套餐外模型 → 正常扣余额
    8. 访问 `/account/subscription` → 看到 Active 状态 + Manage 按钮 → 跳 Stripe Portal 可取消
    9. 在 Stripe Portal 取消 → Webhook `customer.subscription.updated` → Status 变 canceling
    10. 重复订阅尝试 → 返回 already_subscribed 错误
  - [ ] SubTask 25.5: commit 全部改动

---

## 自检清单（实施完成后逐项确认）

- [ ] 主页 LANDING_MODEL_ROWS 不含 GLM 5.2，含 DeepSeek V4 Pro + DeepSeek V4 Flash
- [ ] Navbar 桌面端 + 移动端都显示 Plans 链接
- [ ] `/plans` 直接访问 + 刷新不报错
- [ ] `/account/subscription` 直接访问 + 刷新不报错
- [ ] 未登录访问 `/account/subscription` 跳 sign-in
- [ ] Subscribe Modal 不显示月份数量 / 3/6/12 月选项 / 预付选项
- [ ] 优惠码 Apply 后弹确认 Modal 显示原价/优惠/本次支付/下次续费/优惠范围
- [ ] 无效/过期/达上限优惠码显示错误
- [ ] 已有有效订阅用户点购买返回 already_subscribed
- [ ] 购买按钮点击后 loading + 禁用
- [ ] 套餐内模型调用扣费为 0，使用记录仍写 log
- [ ] 套餐外模型调用正常扣余额
- [ ] 无有效订阅用户走原余额计费
- [ ] 模型别名归一化后匹配套餐模型列表
- [ ] Webhook 防重复处理（幂等）
- [ ] 订阅状态 6 种（Active/Canceling/Canceled/Past due/Payment failed/Expired）正确显示
- [ ] Affiliate 首次付款 + 续费都分佣，按实付金额（coupon 后）
- [ ] 退款/拒付撤销佣金
- [ ] 14 天冻结规则保留
- [ ] Stripe Test/Live 配置分开（复用现有 stripe_credentials 加密存储）
- [ ] 前端不提交/修改付款金额（仅传 plan_id + currency + promotion_code）
- [ ] 套餐模型列表由后端控制（前端不决定模型归属）
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 通过
- [ ] `cd web/default && bun run typecheck && bun run build` 通过
