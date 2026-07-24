# Tasks

- [ ] Task 1: 后端数据模型与全局配置
  - [ ] SubTask 1.1: `model/user.go` User 新增 `CommissionApproved bool`（gorm `column:commission_approved`，无 default）、`PendingCommissionCents int64`（`column:pending_commission_cents`）、`CommissionBalanceCents int64`（`column:commission_balance_cents`）、`CommissionTotalCents int64`（`column:commission_total_cents`）
  - [ ] SubTask 1.2: `model/topup.go` TopUp 新增 `PaidAmountCents int64`（`column:paid_amount_cents`）、`PaidCurrency string`（`column:paid_currency;size:8`），用于落库支付成功时确认的实付金额
  - [ ] SubTask 1.3: `common/constants.go` 新增 `var AffCommissionRate = 0.25`、`var MinWithdrawalCents int64 = 1000`（$10）、`var CommissionFreezeDays = 14`
  - [ ] SubTask 1.4: `model/option.go` 种子 + `updateOption` 新增三个 case：`AffCommissionRate`（ParseFloat）、`MinWithdrawalCents`（ParseInt）、`CommissionFreezeDays`（ParseInt）
  - [ ] SubTask 1.5: `model/main.go` AutoMigrate 注册 `Commission{}`、`Withdrawal{}`；User/TopUp 新列跨三库以 `ADD COLUMN ... DEFAULT 0/FALSE/''` 落地
  - [ ] SubTask 1.6: `go build ./...` 通过

- [ ] Task 2: 佣金结算核心 + 解冻 job（model/commission.go）
  - [ ] SubTask 2.1: 新建 `model/commission.go`，定义 `Commission` 结构（含 `UniqueIndex:[]string{"topup_id","inviter_id"}`）与状态常量 `CommissionStatusPending/Available/Reverted`
  - [ ] SubTask 2.2: 实现 `clampAffCommissionRate(rate float64) float64`：NaN/Inf/`<0`→0，`>1`→1
  - [ ] SubTask 2.3: 实现 `SettleRechargeCommission(tx, inviteeId int, topUpId string, paidAmountCents int64, paidCurrency string) (settled bool, err error)`：按 spec 11 步逻辑（paidAmountCents<=0 跳过 → 读 invitee.InviterId → 自邀防御 → 锁 inviter 读 CommissionApproved → 未批准跳过 → 幂等查 Commission 表 → `QuotaFromFloatChecked(float64(paidAmountCents)*rate)` 饱和取整 → 算 availableAt → 写 Commission pending 行 → 增 inviter `PendingCommissionCents`/`CommissionTotalCents` → clamp 非 nil 时 SysError）。**佣金进 Pending（冻结），不进 Balance**
  - [ ] SubTask 2.4: 实现 `ReleaseMaturedCommissions() (released int, err error)`：分批 100 条查 `status=pending AND available_at<=now`，按 inviter 聚合 sum，`lockForUpdate(user)` → `PendingCommissionCents -= sum`、`CommissionBalanceCents += sum`（绝对值写）→ 批量更新 Commission 行 `status=available`；`PendingCommissionCents` 变负时 SysError + 跳过
  - [ ] SubTask 2.5: `commissionCents <= 0` 时返回 `false, nil`；全部写操作在传入 `tx` 内
  - [ ] SubTask 2.6: `go build ./...` 通过

- [ ] Task 3: 各充值入口捕获实付 + 调用结算
  - [ ] SubTask 3.1: Stripe `Recharge`：从 checkout session 读 `amount_received`（整数美分）落库 `topUp.PaidAmountCents`/`PaidCurrency="USD"`；事务内加额度后调 `SettleRechargeCommission(tx, topUp.UserId, topUp.Id, topUp.PaidAmountCents, "USD")`；提交后记日志
  - [ ] SubTask 3.2: Creem `RechargeCreem`：确认回调实付字段，落库 `PaidAmountCents`/`PaidCurrency`；若 CNY 按 `EffectiveUSDCNYRate` 折 USD 美分；调结算 + 记日志
  - [ ] SubTask 3.3: Waffo `RechargeWaffo`：同 3.2 模式
  - [ ] SubTask 3.4: WaffoPancake `RechargeWaffoPancake`：同 3.2 模式
  - [ ] SubTask 3.5: 易支付 webhook（`controller/topup.go`）：回调 `amount`（CNY 元）按 `round(amount/EffectiveUSDCNYRate*100)` 折 USD 美分，落库 `PaidAmountCents`/`PaidCurrency="CNY"`；`IncreaseUserQuota` 成功后包独立事务调 `SettleRechargeCommission`；记日志
  - [ ] SubTask 3.6: `ManualCompleteTopUp`：按 `topUp.PaymentProvider` 分支取对应实付字段；`ManualTopUpResult` 增 `CommissionSettled bool`；调结算 + 记日志
  - [ ] SubTask 3.7: `go build ./...` 通过

- [ ] Task 4: 互斥——分佣用户不发 ¥100 额度邀请奖励
  - [ ] SubTask 4.1: `model/campaign.go:TrySettleDelayedInviteReward`：`lockForUpdate(inviter)` 后读 `inviter.CommissionApproved`；为 true 时仍清 `invite_reward_pending`、仍发被邀请人 ¥100 promo，但跳过邀请人 ¥100 与 `RewardedInviteCount++`，仅 `AffCount++`
  - [ ] SubTask 4.2: `model/user.go:inviteUser`（即时路径）：获批 inviter 仅 `AffCount++`，不发 `QuotaForInviter`
  - [ ] SubTask 4.3: `go build ./...` 通过

- [ ] Task 5: 管理员批准入口（后端）
  - [ ] SubTask 5.1: `model/user.go:EditWithTx` `updates` 白名单新增 `commission_approved`
  - [ ] SubTask 5.2: 确认 `UpdateUser` 既有 `canManageTargetRole` 校验；`UpdateSelf`/`UpdateUserSetting` 不触及该字段
  - [ ] SubTask 5.3: `controller/user.go:GetSelf` 返回 `commission_approved`/`pending_commission_cents`/`commission_balance_cents`/`commission_total_cents`
  - [ ] SubTask 5.4: `go build ./...` 通过

- [ ] Task 6: 提现链路（model/withdrawal.go + controller）
  - [ ] SubTask 6.1: 新建 `model/withdrawal.go`，定义 `Withdrawal` 结构（含预留字段 `PayoutChannel`/`PayoutTxId`）+ 状态常量（`WithdrawalStatusPending/Approved/Rejected/Paid/Failed`）+ 渠道常量（`PayoutChannelManual/StripeConnect`）
  - [ ] SubTask 6.2: `model/withdrawal.go` 实现 `CreateWithdrawal(tx, userId, amountCents, method, accountInfo) (*Withdrawal, error)`：`lockForUpdate(user)` → 校验 `commission_approved` + `CommissionBalanceCents >= amountCents >= MinWithdrawalCents`（**只用已解冻余额**）→ 扣减 `CommissionBalanceCents`（绝对值写）→ 按 `EffectiveUSDCNYRate` 算 `AmountDisplay` → 写 `Withdrawal(Status=pending, PayoutChannel=manual)`
  - [ ] SubTask 6.3: 实现 `ReviewWithdrawal(tx, id, reviewerId, approve bool, reason string) error`：状态机校验（仅 pending 可审核）；拒绝时 `lockForUpdate(user)` 退还 `CommissionBalanceCents += amountCents`
  - [ ] SubTask 6.4: 实现 `MarkWithdrawalPaid(tx, id, payoutTxId string) error`（仅 approved→paid，终态，记 `PayoutTxId`/`PaidAt`）与 `MarkWithdrawalFailed(tx, id, reason string) error`（approved→failed，退还佣金）
  - [ ] SubTask 6.5: `controller/withdrawal.go` 用户侧：`POST /api/user/withdrawal`（发起）、`GET /api/user/withdrawals`（自己的记录）
  - [ ] SubTask 6.6: `controller/withdrawal.go` 管理员侧：`GET /api/withdrawals`（队列，支持 status 过滤）、`POST /api/withdrawal/:id/approve|reject|mark-paid|mark-failed`
  - [ ] SubTask 6.7: `router/api-router.go` 注册路由（用户侧需 auth，管理员侧需 admin auth）
  - [ ] SubTask 6.8: 注册定时任务每小时调 `ReleaseMaturedCommissions()`（参考 `model/main.go` 既有定时任务模式）
  - [ ] SubTask 6.9: `go build ./...` 通过

- [ ] Task 7: 后端测试
  - [ ] SubTask 7.1: `model/commission_test.go` `TestSettleRechargeCommission`：获批 inviter + 实付 10000 美分 → 佣金 2500 美分入 `PendingCommissionCents`（**不是 BalanceCents**）/`CommissionTotalCents`，写 Commission pending 行（`AvailableAt=now+14d`）；未获批不发；无 inviter 不发；自邀不发
  - [ ] SubTask 7.2: 基数正确性：模拟「付 $100 + 赠送 quota」场景，只传 `paidAmountCents=10000`，佣金按 10000 算不按 quotaAdded 算
  - [ ] SubTask 7.3: 幂等：同 `topUpId+inviterId` 重复调 `SettleRechargeCommission` 第二次返回 `settled=false`，余额不双增（幂等检查 + UniqueIndex）
  - [ ] SubTask 7.4: 饱和：超大 `paidAmountCents` 使乘积近 int32 上限，`QuotaFromFloatChecked` 饱和不溢出为负，clamp 非 nil
  - [ ] SubTask 7.5: `TestReleaseMaturedCommissions`：构造 pending 行 `AvailableAt` 已过/未过，调用后已过的转 available 且 `PendingCommissionCents` 减少 `CommissionBalanceCents` 增加；未过的不变；`PendingCommissionCents` 不会变负
  - [ ] SubTask 7.6: `model/campaign_test.go`：获批 inviter 的 `TrySettleDelayedInviteReward` 不发邀请人 ¥100、不增 `RewardedInviteCount`，被邀请人仍得 promo
  - [ ] SubTask 7.7: `model/withdrawal_test.go`：发起提现扣 `CommissionBalanceCents`（不扣 Pending）；余额不足/<MinWithdrawalCents/非获批拒绝；冻结中佣金不可提；reject 退还；mark-failed 退还；paid 终态不可逆；并发不超发（两个 goroutine 同余额同时申请，只有一个成功）
  - [ ] SubTask 7.8: `go test ./model/... -run 'Commission|DelayedInvite|Settle|Withdrawal|ReleaseMatured'` 通过

- [ ] Task 8: 前端——管理员编辑抽屉批准开关 + 提现审核页
  - [ ] SubTask 8.1: `web/default/src/features/users/types.ts` `userSchema` 增 `commission_approved: z.boolean().optional()`；`UserFormData` 增 `commission_approved?: boolean`
  - [ ] SubTask 8.2: `users-mutate-drawer.tsx`：仅 `isUpdate` + 管理员时渲染 `Switch`「分佣计划成员」，描述「获批后获 25% 现金佣金（冻结 14 天后可提现，替代固定 ¥100 API 额度）」；`transformFormDataToPayload` 携带
  - [ ] SubTask 8.3: `users-columns.tsx`：`commission_approved` 为 true 时展示「分佣」徽标
  - [ ] SubTask 8.4: 新建 `web/default/src/features/withdrawals/`：管理员提现审核页（列表 + 状态过滤 + approve/reject/mark-paid（含 PayoutTxId 输入）/mark-failed 操作 + 退款原因输入）
  - [ ] SubTask 8.5: 注册路由与导航菜单项（管理员区）

- [ ] Task 9: 前端——用户钱包现金佣金卡片 + 系统设置
  - [ ] SubTask 9.1: 钱包页新增「现金佣金」卡片：分别显示 `pending_commission_cents`（冻结中，含预计解冻时间）/`commission_balance_cents`（可提现）/`commission_total_cents`（累计）（按 `EffectiveUSDCNYRate` 折 CNY 展示），**明确标注「这是可提现现金佣金，不是 API 使用额度」**；仅 `commission_approved` 用户显示提现按钮
  - [ ] SubTask 9.2: 提现申请表单（金额、收款方式 alipay/wechat/bank、账号+户名）+ 提现记录列表（状态、金额、时间、拒绝原因、PayoutTxId）
  - [ ] SubTask 9.3: `web/default/src/features/system-settings` 新增「邀请充值佣金比例（0~1）」、「最小提现金额（USD）」、「佣金冻结天数」输入
  - [ ] SubTask 9.4: `web/default/src/i18n/locales/en.json`/`zh.json` 新增全部文案（开关/描述/徽标/卡片/冻结/提现表单/审核页/设置项），英文为 key；`bun run i18n:sync` 同步 fr/ru/ja/vi
  - [ ] SubTask 9.5: `cd web/default && bun run typecheck` 通过
  - [ ] SubTask 9.6: `cd web/default && bun run build` 通过

- [ ] Task 10: 验证与回归
  - [ ] SubTask 10.1: `go build ./...` 通过
  - [ ] SubTask 10.2: `go test ./model/...` 通过（既有 `campaign_test.go`/`redemption` 等不受影响）
  - [ ] SubTask 10.3: 端到端（手动）：管理员给 A 开启分佣 → A 邀请 B → B 充值 $100（实付）→ A 的 `PendingCommissionCents` 增 2500（$25），`CommissionBalanceCents` 不变；A 不再得 ¥100 邀请奖励，B 仍得 ¥100 promo
  - [ ] SubTask 10.4: 端到端（手动）：手动调 `ReleaseMaturedCommissions`（或调小 `CommissionFreezeDays` 后等 job）→ A 的 `PendingCommissionCents` 减 2500、`CommissionBalanceCents` 增 2500，Commission 行转 available
  - [ ] SubTask 10.5: 端到端（手动）：A 提现 $15（≥MinWithdrawalCents，且 ≤BalanceCents）→ 余额扣 1500 → 管理员审核通过 → 标记已付款（填 PayoutTxId）→ 状态 paid 终态；A 余额显示更新
  - [ ] SubTask 10.6: 端到端（手动）：管理员拒绝提现 → 余额退还；标记打款失败 → 余额退还
  - [ ] SubTask 10.7: 端到端（手动）：未获批用户 C 邀请 D → D 充值 → C 不得佣金，C 仍走原 ¥100 邀请奖励路径
  - [ ] SubTask 10.8: 端到端（手动）：易支付 webhook 重试不双发佣金；同订单重复调用不双发
  - [ ] SubTask 10.9: 端到端（手动）：冻结中佣金不可提（提现申请校验 `CommissionBalanceCents` 而非 Pending+Balance）

# Task Dependencies

- Task 1 → Task 2（结算依赖字段与配置）
- Task 2 → Task 3（各入口调用结算）
- Task 1 → Task 4（互斥依赖 `CommissionApproved`）
- Task 1 → Task 5（`EditWithTx` 白名单与 `GetSelf` 依赖字段）
- Task 1、Task 2 → Task 6（提现依赖 `CommissionBalanceCents`/`MinWithdrawalCents`；解冻 job 与 Task 2 同文件）
- Task 2、Task 4 → Task 7（测试覆盖结算、解冻与互斥）
- Task 6 → Task 7.7（提现测试依赖提现链路）
- Task 5 → Task 8.1-8.3（前端开关依赖后端字段）
- Task 6 → Task 8.4-8.5（提现审核页依赖后端 API）
- Task 5、Task 6 → Task 9（钱包卡片依赖 `GetSelf` 透出；提现表单依赖 API）
- Task 9.4 依赖 Task 8 文案确定（i18n 同步）
- Task 10 依赖全部完成
