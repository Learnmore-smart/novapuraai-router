# Spec: 邀请分佣计划（可提现现金佣金 + 14 天冻结）

## 需求

明确区分两类邀请奖励：

1. **普通用户**（未获批）：邀请奖励是 **¥100 API 额度**（promo），只能站内消费，
   **不能提现**。行为与现状一致。
2. **获批分佣用户**（后台批准）：不拿固定 ¥100 额度，改拿被邀请用户**实际付款金额**
   的 **25% 现金佣金**，**可以提现**。持续、无上限、不追溯历史。

两种奖励**互斥**：获批后不再走 ¥100 额度路径，改走现金佣金路径。

> 佣金**不进 `AffQuota`**（那只能转 API 余额消费，不能提现）。需单独建立「可提现佣金
> 余额 + 冻结余额 + 佣金流水 + 提现申请 + 管理员审核」五套结构。

### 两个关键设计点（用户确认补充）

1. **佣金冻结 14 天**：佣金结算时先进入 `PendingCommissionCents`（冻结中），14 天后
   由解冻 job 转入 `CommissionBalanceCents`（可提现）。降低退款/拒付风险——若被邀请人
   在冻结期内退款，可回收对应佣金。冻结期 `CommissionFreezeDays`（默认 14，后台可配）。
2. **提现预留付款字段**：当前用人工审核 + 手动打款；`Withdrawal` 表预留 `PayoutChannel`
   （manual/stripe_connect）与 `PayoutTxId`（Stripe payout id / 手动打款凭证号）字段，
   后期接 Stripe Connect 自动打款时无需改表结构。

## 现状梳理（关键代码路径）

### 邀请关系与固定奖励
- `model/user.go` User：`AffCode`、`InviterId`、`AffCount`、`RewardedInviteCount`、
  `AffQuota`、`AffHistoryQuota`、`InviteRewardPending`。
- `common/constants.go`：`DelayedInviteReward=true`、`InviteRewardCNYYuan=100.0`、
  `MaxValidInvites=10`。
- `model/campaign.go:TrySettleDelayedInviteReward`：被邀请人达资格后给双方各发 ¥100
  promo，邀请人受 `MaxValidInvites` 上限。
- `model/user.go:inviteUser`：非延迟路径按 `QuotaForInviter` 发放到 `AffQuota`。

### 充值链路与「实际付款金额」
`model/topup.go` TopUp 结构关键字段：
- `Money float64`（Stripe）：**经分组倍率换算后的 USD 数量**，不是原始付款——
  不可直接当佣金基数。
- `Amount int64`（其它渠道）：USD 整数数量。
- `PaymentProvider`：`stripe/creem/waffo/waffo_pancake/epay/balance`。

> **必须读取支付成功时确认的实际付款金额**，不能假设 `topUp.Money`/`topUp.Amount`
> 就是实付。Stripe 在 webhook/checkout session 里有 `amount_received`（整数美分），
> Creem/Waffo/Epay 在各自回调里有实际付款字段。需在各 provider 回调里捕获并落库
> 到 TopUp 的新字段 `PaidAmountCents`/`PaidCurrency`。

### 汇率机制
- `setting.EffectiveUSDCNYRate(operation_setting.USDExchangeRate)`：BoC 牌价 + 后台
  配置（默认 7.3）兜底。
- `common.CNYYuanToQuota`/`QuotaToCNYYuan`：CNY↔USD↔quota 换算已封装。
- 系统内部记账本位币为 USD（`QuotaPerUnit`）。

### 现有项目无提现链路
全仓库无 withdraw/cashout/payout 业务代码（仅 stripe skill 文档与「promo 不可提现」
免责声明）。提现从零搭建。

### 管理员编辑用户
- `controller/user.go:UpdateUser` → `model/user.go:EditWithTx`（显式列白名单 `updates`）。
- 前端 `web/default/src/features/users/components/users-mutate-drawer.tsx`。

## 设计方案

### 数据模型

#### 1. `User` 扩展字段
- `CommissionApproved bool`（gorm `column:commission_approved`，无 default，对齐
  `InviteRewardPending`，跨三库 `ADD COLUMN ... DEFAULT FALSE`）。
- `PendingCommissionCents int64`（gorm `column:pending_commission_cents`）：**冻结中**
  佣金（已结算但未到 14 天解冻期），USD 美分 int64。
- `CommissionBalanceCents int64`（gorm `column:commission_balance_cents`）：**已解冻
  可提现**现金佣金余额，USD 美分 int64。
- `CommissionTotalCents int64`（gorm `column:commission_total_cents`）：累计已赚佣金
  （含冻结+已解冻，不含已提现），便于展示与对账。

#### 2. `model/commission.go` — `Commission` 佣金流水表
```
type Commission struct {
    ID              int64   `gorm:"primaryKey"`
    InviterId       int     `gorm:"column:inviter_id;index"`
    InviteeId       int     `gorm:"column:invitee_id;index"`
    TopUpId         string  `gorm:"column:topup_id;index"`         // 关联订单，幂等键
    PaidAmountCents int64   `gorm:"column:paid_amount_cents"`      // 实付（USD 美分）
    PaidCurrency    string  `gorm:"column:paid_currency;size:8"`   // 原始币种 ISO
    Rate            float64 `gorm:"column:rate"`                   // 结算时费率快照
    CommissionCents int64   `gorm:"column:commission_cents"`      // 佣金（USD 美分）
    Status          string  `gorm:"column:status;size:16;index"`  // pending/available/reverted
    AvailableAt     int64   `gorm:"column:available_at;index"`    // 解冻时间戳（created+14d）
    RevertedAt      int64   `gorm:"column:reverted_at"`           // 回收时间戳（退款触发）
    RevertReason    string  `gorm:"column:revert_reason;type:text"`
    CreatedAt       int64   `gorm:"column:created_at"`
}
// UniqueIndex(topup_id, inviter_id) 防同订单重复结算
```

状态机：
- `pending`（冻结中，已入 `PendingCommissionCents`，未到 `AvailableAt`）
- `available`（已解冻，已转 `CommissionBalanceCents`，可提现）
- `reverted`（被邀请人退款，佣金从 `PendingCommissionCents` 扣回，不可提现）

#### 3. `model/withdrawal.go` — `Withdrawal` 提现申请表
```
type Withdrawal struct {
    ID            int64   `gorm:"primaryKey"`
    UserId        int     `gorm:"column:user_id;index"`
    AmountCents   int64   `gorm:"column:amount_cents"`           // 提现金额（USD 美分）
    Currency      string  `gorm:"column:currency;size:8"`        // 展示/打款币种（默认 CNY）
    AmountDisplay float64 `gorm:"column:amount_display"`         // 按申请时汇率折算的展示金额
    Method        string  `gorm:"column:method;size:32"`         // 收款方式（alipay/wechat/bank）
    AccountInfo   string  `gorm:"column:account_info;type:text"` // 收款账户（JSON：账号+户名）
    Status        string  `gorm:"column:status;size:16;index"`   // pending/approved/rejected/paid/failed
    RejectReason  string  `gorm:"column:reject_reason;type:text"`
    ReviewedBy    int     `gorm:"column:reviewed_by"`
    ReviewedAt    int64   `gorm:"column:reviewed_at"`
    // ↓ 预留字段（人工打款阶段用，后期接 Stripe Connect 不改表）
    PayoutChannel string  `gorm:"column:payout_channel;size:32"` // manual/stripe_connect
    PayoutTxId    string  `gorm:"column:payout_tx_id;size:128"`  // Stripe payout id / 手动凭证号
    PaidAt        int64   `gorm:"column:paid_at"`
    CreatedAt     int64   `gorm:"column:created_at"`
}
```

#### 4. 全局配置（`common/constants.go` + `model/option.go`）
- `var AffCommissionRate = 0.25`（佣金比例，0~1，后台可配）
- `var MinWithdrawalCents int64 = 1000`（最小提现 $10）
- `var CommissionFreezeDays = 14`（冻结天数，后台可配）
- `option.go` 种子 + `updateOption` 三个 case；守卫：rate `<0`→0、`>1`→1、NaN/Inf→0；
  freezeDays `<0`→0（立即解冻）。

### 佣金结算核心（`model/commission.go`）

```
// SettleRechargeCommission 结算一笔成功充值对应的邀请人现金佣金。
// 必须在充值订单状态由 Pending→Success 的同一事务内调用，复用充值幂等。
// 佣金进入 PendingCommissionCents（冻结），由解冻 job 在 AvailableAt 后转可提现。
// paidAmountCents: 支付成功时确认的实际付款金额（USD 美分，已折算）。
// paidCurrency: 原始付款币种（ISO，用于流水留存）。
// 返回 settled=true 表示实际发放了佣金。
func SettleRechargeCommission(tx *gorm.DB, inviteeId int, topUpId string,
    paidAmountCents int64, paidCurrency string) (settled bool, err error)
```

逻辑（全部在传入 `tx` 内）：
1. `paidAmountCents <= 0` 或 `inviteeId <= 0` → `false, nil`。
2. `lockForUpdate(tx).First(&invitee, inviteeId)`，读 `InviterId`。
3. `InviterId == 0` 或 `InviterId == inviteeId`（自邀防御）→ `false, nil`。
4. `lockForUpdate(tx).First(&inviter, invitee.InviterId)`，读 `CommissionApproved`。
5. `!inviter.CommissionApproved` → `false, nil`。
6. **幂等检查**：`tx.Where("topup_id = ? AND inviter_id = ?", topUpId, inviter.Id).
   First(&existing)`；若存在 → `false, nil`（已结算）。
7. `rate := clampAffCommissionRate(common.AffCommissionRate)`；
   `commissionCents, clamp := common.QuotaFromFloatChecked(float64(paidAmountCents) * rate)`
   （对美分做比例运算，结果仍是美分；走 `QuotaFromFloatChecked` 饱和取整）。
   - `commissionCents <= 0` → `false, nil`。
   - `clamp` 非 nil → `common.SysError` 记录异常。
8. `availableAt := now + CommissionFreezeDays * 86400`（freezeDays=0 时 `availableAt=now`）。
9. 写 `Commission` 流水（`Status="pending"`，`AvailableAt=availableAt`，含 `Rate` 快照、
   `PaidCurrency`）。
10. `inviter.PendingCommissionCents += commissionCents`；
    `inviter.CommissionTotalCents += commissionCents`；
    `tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
      "pending_commission_cents": inviter.PendingCommissionCents,
      "commission_total_cents": inviter.CommissionTotalCents})`。
11. 返回 `true, nil`。提交后由调用方记 `RecordLog(inviterId, LogTypeSystem, "现金佣金收入 ...（冻结中，X 天后可提现）")`。

> **为什么用 int64 美分**：避免 float64 金额精度误差。1 USD = 100 美分，int64 范围远超
> 任何现实金额。`paidAmountCents * rate` 走 `QuotaFromFloatChecked` 饱和取整。

### 佣金解冻 job（`model/commission.go`）

```
// ReleaseMaturedCommissions 把已到 AvailableAt 且仍 pending 的佣金转为 available，
// 并把对应金额从 PendingCommissionCents 转到 CommissionBalanceCents。
// 建议由后台定时任务（如每小时）调用；也可在用户访问钱包页时按需触发（限频）。
func ReleaseMaturedCommissions() (released int, err error)
```

逻辑（单事务）：
1. `tx.Where("status = ? AND available_at <= ?", "pending", now).Find(&list)`（分批，
   避免大事务；每批 100 条）。
2. 按 `InviterId` 聚合 `sum(commissionCents)`。
3. 对每个 inviter：`lockForUpdate(user)` → `PendingCommissionCents -= sum`、
   `CommissionBalanceCents += sum`（绝对值写）→ 批量 `Update` Commission 行 `status="available"`。
4. 余额非负：`PendingCommissionCents -= sum` 后若 `<0` → `SysError` + 跳过该用户（理论上
   不应发生，因 pending 行与 Pending 字段一一对应）。

### 佣金回收（退款触发，可选 MVP+）

> MVP 不做退款回收（仅冻结 + 自动解冻）。但 `Commission.Status="reverted"` +
> `RevertedAt`/`RevertReason` 字段已预留，后期接退款 webhook 时按 `TopUpId` 找 pending
> 行回收即可。**MVP 实现：若被邀请人退款，佣金仍按 14 天后解冻**（接受这一风险，因
> 14 天冻结期已覆盖大部分退款窗口）。

### 各充值入口：捕获实际付款 + 调用结算

**第一步：各 provider 回调里捕获并落库实际付款**（TopUp 新增 `PaidAmountCents`/
`PaidCurrency` 字段，在翻状态成功时一并写入）：

| Provider | 实际付款来源 | 折算 |
| --- | --- | --- |
| Stripe | checkout session `amount_received`（整数美分，USD） | 直接用，币种 USD |
| Creem | 回调 `amount` × 100（如以 USD 计） | 直接用；若 CNY 按 `EffectiveUSDCNYRate` 折 USD 美分 |
| Waffo / WaffoPancake | 回调实际付款字段 | 同上折算 |
| 易支付 Epay | 回调 `amount`（CNY 元） | `paidAmountCents = round(amount / EffectiveUSDCNYRate × 100)` |
| ManualCompleteTopUp | 管理员补单按 `PaymentProvider` 分支取对应字段 | 同上 |

**第二步**：在翻状态成功 + 加额度 + 写 `PaidAmountCents` 之后、事务提交前调
`SettleRechargeCommission(tx, topUp.UserId, topUp.Id, topUp.PaidAmountCents, topUp.PaidCurrency)`。

> 易支付 webhook 不在单一事务里：`IncreaseUserQuota` 成功后，包 `model.DB.Transaction`
> 调 `SettleRechargeCommission`（状态已翻 Success + 幂等检查保证不双发）。

### 互斥：分佣用户不发 ¥100 额度邀请奖励

- `model/campaign.go:TrySettleDelayedInviteReward`：`lockForUpdate(inviter)` 后读
  `inviter.CommissionApproved`；为 true 时：
  - 仍清被邀请人 `invite_reward_pending`；
  - 仍给**被邀请人**发 ¥100 promo（被邀请人作为普通新用户保留奖励）；
  - **跳过邀请人侧 ¥100 发放**与 `RewardedInviteCount++`，仅 `AffCount++`。
- `model/user.go:inviteUser`（即时路径）：获批 inviter 仅 `AffCount++`，不发 `QuotaForInviter`。

### 提现链路

#### 用户侧
- `POST /api/user/withdrawal`：发起提现申请。
  - 校验 `commission_approved`（仅获批用户可申请）。
  - 校验 `CommissionBalanceCents >= amountCents >= MinWithdrawalCents`（**只用已解冻
    余额，冻结中佣金不可提**）。
  - 校验 `amountCents > 0`、收款信息非空。
  - 事务：`lockForUpdate(user)` → 扣减 `CommissionBalanceCents` → 写 `Withdrawal`
    （`Status="pending"`，`PayoutChannel="manual"`，按当时 `EffectiveUSDCNYRate` 算
    `AmountDisplay`）。
- `GET /api/user/withdrawals`：查看自己的提现记录。
- 前端钱包页新增「现金佣金」卡片：分别显示 `PendingCommissionCents`（冻结中，含预计
  解冻时间）与 `CommissionBalanceCents`（可提现）、`CommissionTotalCents`（累计）；
  **明确标注「这是可提现现金佣金，不是 API 使用额度」**；仅 `commission_approved`
  用户显示提现按钮。

#### 管理员侧
- `GET /api/withdrawals/?status=pending`：提现审核队列。
- `POST /api/withdrawal/:id/approve`：审核通过（`Status="approved"`，记 `ReviewedBy`/`ReviewedAt`）。
- `POST /api/withdrawal/:id/reject`：拒绝（`Status="rejected"`，**退还佣金到
  `CommissionBalanceCents`**，写 `RejectReason`）。
- `POST /api/withdrawal/:id/mark-paid`：标记已打款（`Status="paid"`，`PaidAt`，可选
  填 `PayoutTxId` 凭证号；`PayoutChannel="manual"`）。
- `POST /api/withdrawal/:id/mark-failed`：标记打款失败（`Status="failed"`，**退还佣金**）。
- 前端新增「提现审核」管理页（`web/default/src/features/withdrawals`）。

> **状态机**：pending → approved → paid（成功，终态）/ failed（退款）；
> pending → rejected（退款）。approved 之后打款失败也退款。已 paid 终态不可逆。
> `PayoutChannel`/`PayoutTxId` 预留给后期 Stripe Connect：接通后 mark-paid 由
> Stripe webhook 自动写入 `PayoutChannel="stripe_connect"` + `PayoutTxId=payout_id`。

### 管理员批准入口
- 复用 `UpdateUser` + `EditWithTx`，`updates` 白名单加 `commission_approved`。
- 前端编辑抽屉加 Switch「分佣计划成员」，描述：
  > 获批后，该用户邀请的人充值时，其可获 25% 现金佣金（冻结 14 天后可提现，替代固定
  > ¥100 API 额度）。
- `GetSelf` 透出 `commission_approved`/`pending_commission_cents`/
  `commission_balance_cents`/`commission_total_cents`。

### 系统设置
- `web/default/src/features/system-settings` 新增：
  - 「邀请充值佣金比例（0~1，默认 0.25）」→ `AffCommissionRate`。
  - 「最小提现金额（USD，默认 10）」→ `MinWithdrawalCents`。
  - 「佣金冻结天数（默认 14）」→ `CommissionFreezeDays`。

### 定时任务
- 注册后台 job 每小时调 `ReleaseMaturedCommissions()`（参考既有 `model/main.go` 的
  定时任务注册模式，如 `RecordLog` 的定时清理）。

## 计费安全不变式（AGENTS.md）

- 佣金倍率经 `clampAffCommissionRate` 夹 `[0,1]`，拒 NaN/Inf。
- **佣金基数 = 实付金额 `paidAmountCents`（int64 美分），不是 `quotaAdded`**（后者含
  赠送/分组加成会多算）；也不是 `topUp.Money`（已分组倍率换算）。
- `paidAmountCents * rate` 走 `common.QuotaFromFloatChecked` 饱和取整，不裸 `int()`。
- 佣金永远非负；`commissionCents <= 0` 跳过。
- **幂等三重保护**：① 翻状态门控（每订单只翻一次）② `Commission` 表
   `UniqueIndex(topup_id, inviter_id)` ③ `SettleRechargeCommission` 内查询存在性。
- **冻结与解冻一致**：结算时 `PendingCommissionCents += c` 且写 `pending` 行；
  解冻时 `PendingCommissionCents -= c` 且 `CommissionBalanceCents += c` 且行转 `available`。
  三个操作同事务，保证冻结余额与 pending 行一一对应。
- **提现不超发**：扣减余额用 `lockForUpdate(user)`；只能扣 `CommissionBalanceCents`
  （已解冻），不能扣 `PendingCommissionCents`（冻结中）。
- **余额非负**：扣减前校验 `>= amount`；GORM `Updates` 用绝对值写，不用增量表达式
  （防并发负余额）。
- 拒绝/失败退还佣金，余额只增不减（除用户发起的提现扣减）。
- int64 美分范围远超现实金额，但 `QuotaFromFloatChecked` 仍按 int32 上限饱和——
  `clamp` 非 nil 时 `SysError` + 跳过发放，不会因饱和截断为负。

## 不在范围内

- 不接 Stripe Connect 自动打款（仅人工审核 + 手动打款 + 标记已付款；表结构已预留
  `PayoutChannel`/`PayoutTxId` 字段，后期接通无需改表）。
- 不做分级佣金 / 多级下线（仅一级邀请人）。
- 不追溯历史充值补发佣金（仅结算邀请人获批之后的充值）。
- 不做被邀请人退款自动回收佣金（MVP 接受 14 天冻结期覆盖退款窗口的风险；
  `Commission.Status="reverted"` 字段已预留供后期接入）。
- 不做被邀请人侧互斥（被邀请人仍拿 ¥100 promo）。
- 不改 `MaxValidInvites` 上限语义（仅作用于 ¥100 额度路径）。
- 不改 `web/classic` 旧前端（仅 `web/default`）。
- 佣金以 USD 美分记账，提现展示金额按申请时汇率折算 CNY；实际打款币种由管理员人工处理。

## 风险

- **实付金额捕获**：各 provider 回调字段不同，需逐个确认 `amount_received` 等字段
  存在且可信；缺失时 `paidAmountCents=0` → 不发佣金（安全失败）。
- **幂等**：三重保护，但易支付 webhook 重试需测试不双发。
- **冻结与解冻对账**：`PendingCommissionCents` 必须等于该用户所有 `status=pending`
  行的 `commissionCents` 之和。解冻 job 必须同事务更新两者。需测试并发解冻不超转。
- **退款回收缺失**：MVP 不做退款回收，若被邀请人在 14 天冻结期内退款，佣金仍会解冻。
  接受此风险（14 天通常覆盖退款窗口）；后期接退款 webhook 时按 `TopUpId` 找 pending
  行回收。
- **并发**：提现扣余额与佣金入账都用 `lockForUpdate(user)`，顺序一致（先 user 后
  commission/withdrawal），无死锁。
- **退款一致性**：拒绝/失败必须退还余额；用 `lockForUpdate` + 绝对值写，防并发。
- **跨库迁移**：2 个新表 + User 4 个新列 + TopUp 2 个新列。新表跨三库 AutoMigrate；
  User/TopUp 新列 `ADD COLUMN ... DEFAULT 0`（int64）/`DEFAULT FALSE`（bool）/
  `DEFAULT ''`（string），旧行得零值。
- **金额展示**：前端需把 int64 美分按 `EffectiveUSDCNYRate` 折算成 CNY 展示，并标注
  币种与冻结状态，避免用户误读。

## 受保护信息

- 不触碰 new-api / QuantumNous 相关品牌、版权、模块路径、Go import path 等受保护信息。
