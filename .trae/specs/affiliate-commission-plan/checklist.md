# Checklist

## 设计前提（用户跳过澄清，记录默认决策）
- [x] 分佣用户与 100 额度邀请奖励**互斥**（获批后不再发邀请人 ¥100，改拿充值 25% 佣金）
- [x] 佣金进入既有 `AffQuota`（邀请佣金池），用户经 `TransferAffQuotaToQuota` 转入可消费 `Quota`
- [x] 佣金比例 25% 为**后台可配置**（`AffCommissionRate`，默认 0.25，0~1）
- [x] 佣金适用于**全部充值渠道**（Stripe/Creem/Waffo/WaffoPancake/易支付/管理员补单）
- [x] 被邀请人侧不互斥（仍拿 ¥100 promo）
- [x] 不回溯历史充值（仅结算邀请人获批之后的充值）

## 后端：数据模型与配置
- [ ] `User.CommissionApproved` 字段（gorm 仅 `column:commission_approved`，无 default）
- [ ] `common.AffCommissionRate = 0.25`
- [ ] `model/option.go` 种子 + `updateOption` `case "AffCommissionRate"`（ParseFloat）
- [ ] `commission_approved` 跨三库 AutoMigrate 以 `ADD COLUMN ... DEFAULT FALSE` 落地
- [ ] `go build ./...` 通过

## 后端：佣金结算核心
- [ ] `model/commission.go` 新建
- [ ] `clampAffCommissionRate`：NaN/Inf/<0→0，>1→1
- [ ] `SettleRechargeCommission(tx, inviteeId, quotaAdded)`：读 InviterId→自邀防御→锁 inviter 读 CommissionApproved→未批准返回 false→`QuotaFromFloatChecked` 饱和取整→增 AffQuota/AffHistoryQuota
- [ ] `commission <= 0` 跳过；全部写操作在传入 `tx` 内
- [ ] clamp 非 nil 时 `common.SysError` 记录

## 后端：接入充值入口
- [ ] `Recharge`（Stripe）事务内调用 + 提交后记日志
- [ ] `RechargeCreem` 同上
- [ ] `RechargeWaffo` 同上
- [ ] `RechargeWaffoPancake` 同上
- [ ] `ManualCompleteTopUp` 事务内调用 + `ManualTopUpResult.CommissionSettled`
- [ ] 易支付 webhook：`IncreaseUserQuota` 后包独立事务调用 + 记日志
- [ ] `go build ./...` 通过

## 后端：互斥逻辑
- [ ] `TrySettleDelayedInviteReward`：获批 inviter 跳过邀请人 ¥100 与 `RewardedInviteCount++`，仅 `AffCount++`；被邀请人仍发 ¥100 promo；仍清 `invite_reward_pending`
- [ ] `inviteUser`（即时路径）：获批 inviter 仅 `AffCount++`，不发 `QuotaForInviter`
- [ ] `CommissionApproved` 读取在 `lockForUpdate(inviter)` 之后（事务内最新值）

## 后端：管理员批准入口
- [ ] `EditWithTx` `updates` 白名单新增 `commission_approved`
- [ ] `UpdateUser` 复用既有 `canManageTargetRole` 权限校验
- [ ] `UpdateSelf`/`UpdateUserSetting` 不触及 `commission_approved`
- [ ] `GetSelf` 返回 `commission_approved`
- [ ] `go build ./...` 通过

## 后端：测试
- [ ] `TestSettleRechargeCommission`：获批发佣金 / 未获批不发 / 无 inviter 不发 / 自邀不发
- [ ] 幂等：翻状态后二次进入不双发（状态门控 + `quotaAdded<=0`/未获批分支）
- [ ] 饱和：近 int32 上限 `quotaAdded`*0.25 经 `QuotaFromFloatChecked` 不溢出为负，clamp 非 nil
- [ ] `TrySettleDelayedInviteReward` 获批 inviter 不发邀请人 ¥100、被邀请人仍得 promo
- [ ] `go test ./model/... -run 'Commission|DelayedInvite|Settle'` 通过

## 计费安全不变式（AGENTS.md）
- [ ] 佣金倍率经 `clampAffCommissionRate` 夹到 [0,1]，拒绝 NaN/Inf
- [ ] `quotaAdded * rate` 走 `common.QuotaFromFloatChecked`（不裸 `int(...)`）
- [ ] 佣金永远非负；无负佣金路径
- [ ] 充值金额本身经各入口既有校验（`quotaToAdd<=0` 拒绝、`MaxQuota` 上限）

## 前端（web/default）
- [ ] `userSchema`/`UserFormData` 增 `commission_approved`
- [ ] 编辑抽屉（仅 update + 管理员）增 `Switch` + 描述
- [ ] `transformFormDataToPayload` update 模式携带 `commission_approved`
- [ ] `users-columns.tsx` 增「分佣」徽标（可选）
- [ ] 钱包卡片展示分佣成员状态与佣金比例
- [ ] 系统设置增「邀请充值佣金比例（0~1）」输入
- [ ] en.json/zh.json 新增文案；`bun run i18n:sync` 同步 fr/ru/ja/vi
- [ ] `bun run typecheck` 通过
- [ ] `bun run build` 通过

## 回归与边界
- [ ] 普通用户（未获批）100 额度邀请奖励行为不变（含 `MaxValidInvites` 上限）
- [ ] 获批用户充值佣金持续、无上限（不受 `MaxValidInvites` 约束）
- [ ] 管理员撤销批准后，后续充值不再发佣金；已发佣金不回收
- [ ] 自邀防御（`InviterId == inviteeId` 不发）
- [ ] 易支付 webhook 重试不双发佣金
- [ ] 跨库（SQLite/MySQL/PostgreSQL）迁移与查询正常
- [ ] 不改 `web/classic` 旧前端
- [ ] 不做提现/分级佣金/历史回溯（在范围外）

## 受保护信息
- [ ] 未触碰 new-api / QuantumNous 相关品牌、版权、模块路径、Go import path 等受保护信息
