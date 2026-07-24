# Spec: 邀请分佣计划（被邀请用户充值 25% 佣金）

## 问题描述

当前邀请奖励只有一种：被邀请用户达到资格后，邀请人获得固定的 ¥100 额度奖励
（`common.InviteRewardCNYYuan`，经 `TrySettleDelayedInviteReward` 延迟发放，
受 `MaxValidInvites=10` 上限约束）。所有用户一视同仁，**不参与**被邀请人的充值分成。

期望：

1. **普通用户**保留原来的固定 100 额度邀请奖励，行为不变。
2. 新增「分佣计划」：**只有后台专门批准加入分佣计划的用户**，才可获得其邀请
   用户充值金额的 **25% 佣金**（按被邀请人每次成功充值的额度结算，持续、无上限）。

> 说明：分佣用户与 100 额度奖励**互斥**——被批准加入分佣计划后，不再发放固定
> 100 额度邀请奖励，改为按被邀请人充值额度的 25% 持续结算佣金。普通用户（未批准）
> 仍走原 100 额度路径。该互斥语义来自需求原文「保留**普通用户**原来的 100 额度奖励」
> 与「**只有**分佣计划用户才可获得佣金」的对照。

## 现状梳理（关键代码路径）

### 邀请关系与固定奖励
- `model/user.go` User 结构：`AffCode`、`InviterId`、`AffCount`、`RewardedInviteCount`、
  `AffQuota`（邀请佣金池，可转入现金）、`AffHistoryQuota`、`InviteRewardPending`。
- `common/constants.go`：`QuotaForInviter=0`、`QuotaForInvitee=0`（即时路径，默认关闭）、
  `DelayedInviteReward=true`、`InviteRewardCNYYuan=100.0`、`MaxValidInvites=10`。
- `model/user.go:finishInsert`：注册时若 `DelayedInviteReward`，仅写 `inviter_id`+
  `invite_reward_pending=true`，不立即发奖。
- `model/campaign.go:TrySettleDelayedInviteReward`：被邀请人达到资格（邮箱+令牌+首笔
  计费成功）后，给双方各发 ¥100 额度（promo），邀请人受 `MaxValidInvites` 上限。
- `model/user.go:inviteUser`：非延迟路径下按 `QuotaForInviter` 发放到 `AffQuota`。
- `service/text_quota.go:508`：首笔计费成功后调用 `TrySettleDelayedInviteReward`。
- `controller/user.go:TransferAffQuota` / `model/user.go:TransferAffQuotaToQuota`：
  用户将 `AffQuota` 手动转入可消费 `Quota`（最小单位 `QuotaPerUnit`）。

### 充值链路（佣金触发点）
`model/topup.go` 与 `controller/topup.go` 中所有把订单 `Status` 由 `Pending` 翻为
`Success` 并给用户加额度的入口，均需结算佣金：

1. `model/topup.go:Recharge`（Stripe）—— 事务内 `quota = Money * QuotaPerUnit`，
   `quota + ?`。
2. `model/topup.go:RechargeCreem`（Creem）—— `quota = Amount`，`quota + ?`。
3. `model/topup.go:RechargeWaffo`（Waffo）—— `quotaToAdd = Amount * QuotaPerUnit`，
   `quota + ?`。
4. `model/topup.go:RechargeWaffoPancake`（Waffo Pancake）—— 同上。
5. `model/topup.go:ManualCompleteTopUp`（管理员补单）—— `quotaToAdd`（经
   `QuotaFromDecimalChecked` 饱和），写 `quota = quotaAfter`。
6. `controller/topup.go` 易支付 webhook（`PaymentProviderEpay`）：`topUp.Update()` 翻
   状态后 `model.IncreaseUserQuota(...)`。

所有入口都先校验 `Status==Pending` 再翻 `Success`，天然幂等——佣金在同一翻状态路径
结算即可保证「每笔充值只发一次佣金」。

### 管理员编辑用户
- `controller/user.go:UpdateUser` → `model/user.go:EditWithTx`。
- `EditWithTx` 用**显式列白名单** `updates` map（`username/display_name/group/
  remark[/password]`）写库，新增字段必须显式加入该白名单。
- 前端 `web/default/src/features/users/components/users-mutate-drawer.tsx`：管理员
  编辑抽屉，`transformFormDataToPayload` 组装提交体。
- 邀请/分享相关已有「后台批准」先例：`controller/share.go` + `share-review-dialog.tsx`
  （分享奖励审批队列）。

### 系统选项持久化
- `model/option.go`：`OptionMap` 种子 + `updateOption` switch（如 `QuotaForInviter`）。
- `common/constants.go` 存放可配置全局变量。

## 设计方案

### 数据模型

1. **`User.CommissionApproved`（新字段，bool）**
   - `model/user.go`：`CommissionApproved bool `json:"commission_approved" gorm:"column:commission_approved"``
   - 不加 gorm `default:` 标签，遵循 `InviteRewardPending` 同款约定（避免跨库
     AutoMigrate 重复 ALTER）。Go 零值 `false` 即「未批准」。
   - GORM AutoMigrate 跨 SQLite/MySQL/PostgreSQL 以 `ADD COLUMN ... DEFAULT FALSE`
     落地，旧行得到 `false`。

2. **`common.AffCommissionRate`（新全局配置，float64，默认 0.25）**
   - `common/constants.go`：`var AffCommissionRate = 0.25`。
   - `model/option.go`：种子 `common.OptionMap["AffCommissionRate"] =
     strconv.FormatFloat(..., 'f', -1, 64)`；`updateOption` 新增 `case "AffCommissionRate"`
     用 `strconv.ParseFloat` 回写。
   - 取值范围守卫：`<0` 视为 `0`（不发），`>1` 视为 `1`（封顶 100%）；NaN/Inf 视为 `0`。

### 佣金结算（核心）

新增 `model/commission.go`：

```
// SettleRechargeCommission 结算一笔成功充值对应的邀请人佣金。
// 必须在充值订单状态由 Pending→Success 的同一事务内调用（或紧随其后的同一翻状态
// 代码块内），以复用充值的幂等性：每笔订单只翻一次状态 ⇒ 只结算一次佣金。
//
// inviteeId: 被邀请人(充值人) id；quotaAdded: 本次充值给被邀请人增加的额度。
// 返回 settled=true 表示实际发放了佣金。
func SettleRechargeCommission(tx *gorm.DB, inviteeId int, quotaAdded int) (settled bool, err error)
```

逻辑（全部在传入的 `tx` 内，保证与充值原子）：
1. `quotaAdded <= 0` 或 `inviteeId <= 0` → 直接返回 `false, nil`。
2. `lockForUpdate(tx).First(&invitee, inviteeId)`，读 `InviterId`。
3. `InviterId == 0` 或 `InviterId == inviteeId`（防御自邀）→ `false, nil`。
4. `lockForUpdate(tx).First(&inviter, invitee.InviterId)`，读 `CommissionApproved`。
5. `!inviter.CommissionApproved` → `false, nil`（普通用户走原 100 额度路径，不在此结算）。
6. 计算佣金：`rate := clampAffCommissionRate(common.AffCommissionRate)`；
   `commission, clamp := common.QuotaFromFloatChecked(float64(quotaAdded) * rate)`
   （走 `common/quota_math.go` 饱和取整，符合计费安全不变式；`clamp` 非 nil 时
   `common.SysError` 记录异常）。
   - `commission <= 0` → `false, nil`。
7. `inviter.AffQuota += commission`；`inviter.AffHistoryQuota += commission`；
   `tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
     "aff_quota": inviter.AffQuota, "aff_history": inviter.AffHistoryQuota})`。
8. 返回 `true, nil`。日志在事务提交后由调用方记录（与现有 `RecordTopupLog` 风格一致），
   内容形如「分佣收入 %s（来自用户 %d 充值）」。

> 佣金进入 `AffQuota`（邀请佣金池，可转入现金），与现有邀请额度资金池一致；
> 用户经既有 `TransferAffQuotaToQuota` 转入可消费 `Quota`。不直接加到 `Quota`，
> 避免改变「promo/cash/aff」三类余额语义。

### 接入各充值入口

在每个入口「翻状态 + 加额度」之后、事务提交之前调用 `SettleRechargeCommission`：

| 入口 | 改动 |
| --- | --- |
| `Recharge` (Stripe) | 事务内 `quota + ?` 后调 `SettleRechargeCommission(tx, topUp.UserId, int(quota))`；提交后按 `settled` 记日志 |
| `RechargeCreem` | 同上，`quotaAdded = int(quota)` |
| `RechargeWaffo` | 同上，`quotaToAdd` |
| `RechargeWaffoPancake` | 同上 |
| `ManualCompleteTopUp` | 事务内 `Update("quota", quotaAfter)` 后调；`result` 增 `CommissionSettled bool` 供审计 |
| 易支付 webhook (`controller/topup.go`) | `model.IncreaseUserQuota` 成功后，用**独立事务**调 `SettleRechargeCommission`（该入口本身不在单一事务里），并在 `topUp.Status==Success` 时跳过（幂等） |

> 易支付路径：webhook 在 `topUp.Status==Pending` 分支内先 `topUp.Update()` 翻状态、
> 再 `IncreaseUserQuota`。佣金结算放在 `IncreaseUserQuota` 成功之后，包在
> `model.DB.Transaction(...)` 里调用 `SettleRechargeCommission`；因外层已用
> `LockOrder/UnlockOrder` 串行化同 trade_no，且状态已翻 Success，重试 webhook 不会
> 重复进入此分支，幂等成立。

### 互斥：分佣用户不发 100 额度邀请奖励

- `model/campaign.go:TrySettleDelayedInviteReward`：在确定 `inviterId` 后、发放前，
  读取 `inviter.CommissionApproved`；为 `true` 时：
  - 仍清掉被邀请人的 `invite_reward_pending`（消费该状态，防反复触发）；
  - **仍给被邀请人发 ¥100 promo**（被邀请人作为普通新用户保留奖励，需求只说邀请人侧
    互斥；被邀请人侧不变更更安全，且不改变被邀请人体验）。
  - **跳过邀请人侧发放**（不发 ¥100、不增 `RewardedInviteCount`，仅 `AffCount++`）。
- `model/user.go:inviteUser`（非延迟路径）：开头读 `CommissionApproved`，为 `true` 时
  仅 `AffCount++` 并返回，不发 `QuotaForInviter`。
- `model/user.go:finishInsert` 即时路径同样受 `inviteUser` 改动覆盖。

> 备注：被邀请人侧是否也互斥？需求只要求「邀请人」拿佣金替代 100。被邀请人仍按原
> 逻辑拿 ¥100 promo，保持新用户体验。如需被邀请人侧也改，后续调整。

### 管理员批准入口（后台）

- **后端**：复用 `UpdateUser`（`controller/user.go`）+ `EditWithTx`。
  - `EditWithTx` 的 `updates` 白名单新增 `"commission_approved": newUser.CommissionApproved`。
  - `UpdateUser` 已有 `canManageTargetRole` 校验，仅管理员可改；无需额外权限。
  - 不开放给用户自助修改（`UpdateSelf`/`UpdateUserSetting` 不涉及此字段）。
- **前端**：`users-mutate-drawer.tsx`（仅 update 模式、仅管理员可见）增加一个
  `Switch`「分佣计划（获批邀请人可获被邀请人充值 25% 佣金）」，绑定
  `commission_approved`，纳入 `transformFormDataToPayload`。
- `users-columns.tsx` / 行操作：可选展示一个「分佣」徽标（`commission_approved` 为 true
  时），便于后台识别。

### 用户侧可见性

- `GetSelf`（`controller/user.go`）已返回 `aff_quota`/`aff_history_quota`/`inviter_id`；
  增加 `commission_approved` 字段输出。
- 钱包页（`web/default/src/features/wallet`）：在邀请/佣金卡片处展示
  「是否分佣计划成员」与当前佣金比例；`AffQuota` 转账入口已存在，无需新增。
- 佣金收入记录：用 `RecordLog(inviterId, LogTypeSystem, ...)` 写入用户日志，用户可在
  日志页查看（与既有邀请奖励日志同类型）。

### 系统设置（佣金比例）

- `web/default/src/features/system-settings` 新增一个数值输入「邀请充值佣金比例
  （0~1，默认 0.25）」，提交 `AffCommissionRate` 到 `/api/option`。
- 与 `QuotaForInviter`/`QuotaForInvitee` 同一设置区。

## 计费安全不变式（遵循 AGENTS.md）

- 佣金倍率 `AffCommissionRate` 为用户可控（管理员配置）的计费乘子，须在
  `clampAffCommissionRate` 内夹到 `[0,1]` 并拒绝 NaN/Inf 后再参与乘法。
- `quotaAdded * rate` 经 `common.QuotaFromFloatChecked` 饱和取整（int32 上限）；
  `clamp` 非 nil 时 `common.SysError` 记录，单请求不可能触达上限。
- 不使用裸 `int(float64(...))` 转换；统一走 `common/quota_math.go`。
- 佣金永远非负：`commission <= 0` 时跳过发放；不存在「负佣金=给邀请人扣额」路径。
- 充值金额本身已被各入口校验（`quotaToAdd <= 0` 拒绝、`MaxQuota` 上限校验等）。

## 不在范围内

- 不做分佣提现到法币（仅入 `AffQuota` → 转入 `Quota` 消费）。
- 不做分级佣金 / 多级下线（仅一级邀请人）。
- 不做历史充值回溯补发佣金（仅结算「邀请人已获批」之后的充值）。
- 不做被邀请人侧互斥（被邀请人仍拿 ¥100 promo）。
- 不做独立的分佣审批队列 UI（用编辑抽屉里的开关，足够直接）。
- 不改 `MaxValidInvites` 上限语义（仅作用于 100 额度路径，分佣不受其约束）。
- 不改 `web/classic` 旧前端（仅 `web/default`）。

## 风险

- **幂等**：佣金结算与翻状态同事务/同分支，复用充值幂等；易支付路径靠
  `LockOrder` + 状态校验。需测试 webhook 重试不双发。
- **跨库迁移**：`commission_approved` 布尔列，三库 AutoMigrate 均 `ADD COLUMN ...
  DEFAULT FALSE`，无 `ALTER COLUMN` 风险；旧行得 `false`。
- **并发**：邀请人行用 `lockForUpdate` 加锁，与 `inviteUser`/
  `TransferAffQuotaToQuota` 同锁竞争，无死锁（顺序一致：先 invitee 后 inviter）。
- **配置回退**：`AffCommissionRate` 解析失败/NaN 时夹为 0（不发），不会因配置错误
  超发。
- **互斥误判**：`TrySettleDelayedInviteReward` 中读 `CommissionApproved` 必须在
  `lockForUpdate(inviter)` 之后，确保用的是事务内最新值。

## 受保护信息

- 不触碰 new-api / QuantumNous 相关品牌、版权、模块路径、Go import path 等受保护信息。
