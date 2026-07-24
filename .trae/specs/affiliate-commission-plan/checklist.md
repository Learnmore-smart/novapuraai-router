# Checklist

## 设计前提（用户确认的决策）
- [x] **普通用户**：¥100 API 额度（promo），站内消费，**不能提现**
- [x] **获批分佣用户**：实付金额 25% **现金佣金**，**可以提现**
- [x] 两种奖励**互斥**（获批后不走 ¥100 路径）
- [x] **佣金不进 AffQuota**（那不能提现）；新建可提现佣金余额 + 冻结余额 + 流水 + 提现申请 + 审核
- [x] **金额用 int64 美分**（避免浮点误差），USD 本位币记账
- [x] **佣金基数 = 实付金额**（读取支付成功时确认的实际付款，不假设 `topUp.Money`/`Amount`）
- [x] **佣金冻结 14 天**：先进 `PendingCommissionCents`，解冻 job 后转 `CommissionBalanceCents`（可提现）；降低退款/拒付风险
- [x] 佣金比例 25% 后台可配（`AffCommissionRate`，默认 0.25，0~1）
- [x] 最小提现 $10（`MinWithdrawalCents=1000`，后台可配）
- [x] 冻结天数后台可配（`CommissionFreezeDays=14`）
- [x] 提现：人工审核 + 手动打款 + 标记已付款（不接 Stripe Connect）
- [x] **Withdrawal 预留 `PayoutChannel`/`PayoutTxId` 字段**，后期接 Stripe Connect 不改表
- [x] 不追溯历史充值
- [x] 被邀请人侧不互斥（仍拿 ¥100 promo）
- [x] MVP 不做退款回收（`reverted` 状态字段已预留供后期接入）

## 后端：数据模型与配置
- [ ] User 新增 `CommissionApproved`/`PendingCommissionCents`/`CommissionBalanceCents`/`CommissionTotalCents`（无 gorm default）
- [ ] TopUp 新增 `PaidAmountCents`/`PaidCurrency`（落库实付）
- [ ] `common.AffCommissionRate = 0.25`、`common.MinWithdrawalCents = 1000`、`common.CommissionFreezeDays = 14`
- [ ] `model/option.go` 种子 + `updateOption` 三个 case
- [ ] AutoMigrate 注册 `Commission`/`Withdrawal`；User/TopUp 新列跨三库 `ADD COLUMN ... DEFAULT 0/FALSE/''`
- [ ] `go build ./...` 通过

## 后端：佣金结算核心 + 解冻 job
- [ ] `model/commission.go` 新建
- [ ] `Commission` 结构 + `UniqueIndex(topup_id, inviter_id)` + 状态常量 + `AvailableAt`/`RevertedAt`/`RevertReason`
- [ ] `clampAffCommissionRate`：NaN/Inf/<0→0，>1→1
- [ ] `SettleRechargeCommission(tx, inviteeId, topUpId, paidAmountCents int64, paidCurrency string)`：11 步逻辑
- [ ] **佣金进 `PendingCommissionCents`（冻结），不进 `CommissionBalanceCents`**
- [ ] **幂等三重保护**：翻状态门控 + UniqueIndex + 函数内查存在性
- [ ] `paidAmountCents <= 0` 或 `commissionCents <= 0` 跳过；clamp 非 nil 时 `common.SysError`
- [ ] 全部写操作在传入 `tx` 内
- [ ] `ReleaseMaturedCommissions()`：分批 100 条查 `status=pending AND available_at<=now`，按 inviter 聚合，`lockForUpdate(user)` 转账（绝对值写），批量更新行 status=available
- [ ] 解冻时 `PendingCommissionCents` 不会变负（变负时 SysError + 跳过）

## 后端：各充值入口捕获实付 + 调用结算
- [ ] Stripe：读 checkout session `amount_received` 落库 + 调结算
- [ ] Creem：确认回调实付字段 + 调结算
- [ ] Waffo / WaffoPancake：同上
- [ ] 易支付 webhook：CNY 元按 `EffectiveUSDCNYRate` 折 USD 美分 + 独立事务调结算
- [ ] ManualCompleteTopUp：按 `PaymentProvider` 分支取实付 + `ManualTopUpResult.CommissionSettled`
- [ ] `go build ./...` 通过

## 后端：互斥逻辑
- [ ] `TrySettleDelayedInviteReward`：获批 inviter 跳过 ¥100 与 `RewardedInviteCount++`，仅 `AffCount++`；被邀请人仍发 promo；仍清 `invite_reward_pending`
- [ ] `inviteUser`（即时路径）：获批 inviter 仅 `AffCount++`，不发 `QuotaForInviter`
- [ ] `CommissionApproved` 读取在 `lockForUpdate(inviter)` 之后

## 后端：管理员批准入口
- [ ] `EditWithTx` `updates` 白名单新增 `commission_approved`
- [ ] `UpdateUser` 复用既有 `canManageTargetRole`；`UpdateSelf`/`UpdateUserSetting` 不触及
- [ ] `GetSelf` 返回 `commission_approved`/`pending_commission_cents`/`commission_balance_cents`/`commission_total_cents`
- [ ] `go build ./...` 通过

## 后端：提现链路 + 定时任务
- [ ] `model/withdrawal.go` `Withdrawal` 结构（含预留 `PayoutChannel`/`PayoutTxId`）+ 状态常量 + 渠道常量
- [ ] `CreateWithdrawal`：`lockForUpdate(user)` + 校验（approved/已解冻余额/MinWithdrawalCents）+ 扣减 `CommissionBalanceCents`（绝对值写）+ 算 `AmountDisplay` + 写 pending（`PayoutChannel=manual`）
- [ ] `ReviewWithdrawal`：状态机（仅 pending 可审核）；拒绝时 `lockForUpdate(user)` 退还 `CommissionBalanceCents`
- [ ] `MarkWithdrawalPaid`（approved→paid 终态，记 `PayoutTxId`/`PaidAt`）、`MarkWithdrawalFailed`（approved→failed 退还）
- [ ] `controller/withdrawal.go` 用户侧：发起 + 列表
- [ ] `controller/withdrawal.go` 管理员侧：队列 + approve/reject/mark-paid（含 PayoutTxId）/mark-failed
- [ ] `router/api-router.go` 注册路由（用户 auth / 管理员 admin auth）
- [ ] 注册定时任务每小时调 `ReleaseMaturedCommissions()`
- [ ] `go build ./...` 通过

## 后端：测试
- [ ] `TestSettleRechargeCommission`：获批发（入 Pending 不入 Balance）/ 未获批不发 / 无 inviter 不发 / 自邀不发
- [ ] 基数正确性：付 $100+赠送场景，只按实付 10000 美分算佣金
- [ ] 幂等：同 `topUpId+inviterId` 重复调不双发（余额不双增）
- [ ] 饱和：超大 paidAmountCents 经 `QuotaFromFloatChecked` 不溢出为负，clamp 非 nil
- [ ] `TestReleaseMaturedCommissions`：已过解冻期转 available + 转账；未过不变；Pending 不变负
- [ ] `TrySettleDelayedInviteReward` 获批 inviter 不发 ¥100、被邀请人仍得 promo
- [ ] `withdrawal_test.go`：扣 Balance（不扣 Pending）/ 余额不足/<MinWithdrawalCents/非获批拒绝 / 冻结中佣金不可提 / reject 退还 / mark-failed 退还 / paid 终态不可逆 / 并发不超发
- [ ] `go test ./model/... -run 'Commission|DelayedInvite|Settle|Withdrawal|ReleaseMatured'` 通过

## 计费安全不变式（AGENTS.md）
- [ ] 佣金倍率经 `clampAffCommissionRate` 夹 [0,1]，拒 NaN/Inf
- [ ] **佣金基数为 paidAmountCents（实付），不是 quotaAdded / topUp.Money**
- [ ] `paidAmountCents * rate` 走 `common.QuotaFromFloatChecked`（不裸 `int()`）
- [ ] 佣金非负；无负佣金路径
- [ ] 幂等三重保护
- [ ] **冻结与解冻一致**：结算 `Pending+=c` + 写 pending 行；解冻 `Pending-=c` + `Balance+=c` + 行转 available，同事务
- [ ] 提现只扣 `CommissionBalanceCents`（已解冻），不扣 `PendingCommissionCents`（冻结中）
- [ ] 提现扣余额用 `lockForUpdate` + 绝对值写（不用增量表达式，防并发负余额）
- [ ] 拒绝/失败退还佣金，余额只增不减（除用户发起的提现扣减）

## 前端（web/default）
- [ ] `userSchema`/`UserFormData` 增 `commission_approved`
- [ ] 编辑抽屉（仅 update + 管理员）增 `Switch` + 描述（明确「可提现现金佣金，冻结 14 天，替代 ¥100 API 额度」）
- [ ] `transformFormDataToPayload` 携带 `commission_approved`
- [ ] `users-columns.tsx` 增「分佣」徽标
- [ ] 新建 `features/withdrawals/` 管理员审核页（列表/过滤/approve/reject/mark-paid（含 PayoutTxId 输入）/mark-failed/退款原因）
- [ ] 钱包页「现金佣金」卡片：分别显示冻结中（含预计解冻时间）/可提现/累计，**明确标注「可提现现金佣金，不是 API 额度」**，仅获批用户显示提现按钮
- [ ] 提现申请表单（金额/收款方式/账号+户名）+ 提现记录列表（含 PayoutTxId）
- [ ] 系统设置增「佣金比例（0~1）」、「最小提现金额（USD）」、「佣金冻结天数」
- [ ] en.json/zh.json 新增全部文案；`bun run i18n:sync` 同步 fr/ru/ja/vi
- [ ] `bun run typecheck` 通过
- [ ] `bun run build` 通过

## 回归与边界
- [ ] 普通用户（未获批）¥100 额度邀请奖励行为不变（含 `MaxValidInvites` 上限）
- [ ] 获批用户充值佣金持续、无上限（不受 `MaxValidInvites` 约束）
- [ ] 获批用户佣金先进冻结池，14 天后转可提现
- [ ] 冻结中佣金不可提现（提现只看 `CommissionBalanceCents`）
- [ ] 管理员撤销批准后，后续充值不再发佣金；已发佣金不回收
- [ ] 自邀防御（`InviterId == inviteeId` 不发）
- [ ] 易支付 webhook 重试不双发；同订单重复调用不双发
- [ ] 提现状态机：pending→approved→paid（终态）/ failed（退款）；pending→rejected（退款）；approved 之后打款失败也退款
- [ ] 并发提现不超发（同余额两个 goroutine 同时申请，只有一个成功）
- [ ] 解冻 job 并发不超转（`PendingCommissionCents` 不变负）
- [ ] 跨库（SQLite/MySQL/PostgreSQL）迁移与查询正常
- [ ] 不改 `web/classic` 旧前端
- [ ] 不接 Stripe Connect / 不做分级佣金 / 不追溯历史 / 不做退款回收（在范围外，字段已预留）

## 受保护信息
- [ ] 未触碰 new-api / QuantumNous 相关品牌、版权、模块路径、Go import path 等受保护信息
