# Tasks

- [ ] Task 1: 后端数据模型与全局配置
  - [ ] SubTask 1.1: `model/user.go` User 结构新增 `CommissionApproved bool` 字段，gorm tag 仅 `column:commission_approved`（无 default，对齐 `InviteRewardPending`）
  - [ ] SubTask 1.2: `common/constants.go` 新增 `var AffCommissionRate = 0.25`（注释说明：被邀请人充值额度的佣金比例，0~1）
  - [ ] SubTask 1.3: `model/option.go` 种子 `common.OptionMap["AffCommissionRate"] = strconv.FormatFloat(common.AffCommissionRate, 'f', -1, 64)`；`updateOption` 新增 `case "AffCommissionRate"`：`strconv.ParseFloat` 后回写 `common.AffCommissionRate`
  - [ ] SubTask 1.4: `go build ./...` 通过；GORM AutoMigrate 在三库以 `ADD COLUMN ... DEFAULT FALSE` 落地 `commission_approved`

- [ ] Task 2: 佣金结算核心（model/commission.go）
  - [ ] SubTask 2.1: 新建 `model/commission.go`，实现 `clampAffCommissionRate(rate float64) float64`：NaN/Inf/`<0`→0，`>1`→1
  - [ ] SubTask 2.2: 实现 `SettleRechargeCommission(tx *gorm.DB, inviteeId int, quotaAdded int) (settled bool, err error)`：按 spec 逻辑（读 invitee.InviterId→自邀防御→锁 inviter 读 CommissionApproved→未批准返回 false→`common.QuotaFromFloatChecked(float64(quotaAdded)*rate)` 饱和取整→增 `AffQuota`/`AffHistoryQuota`→clamp 非 nil 时 `common.SysError`）
  - [ ] SubTask 2.3: `commission <= 0` 时返回 `false, nil`；全部写操作在传入 `tx` 内
  - [ ] SubTask 2.4: `go build ./...` 通过

- [ ] Task 3: 接入各充值入口
  - [ ] SubTask 3.1: `model/topup.go:Recharge`（Stripe）：事务内 `quota + ?` 后调 `SettleRechargeCommission(tx, topUp.UserId, int(quota))`；提交后 `settled` 为 true 时 `RecordLog(inviterId, LogTypeSystem, "分佣收入 ...")`（需返回 inviterId 或在函数内记日志）
  - [ ] SubTask 3.2: `RechargeCreem` 同 3.1，`quotaAdded = int(quota)`
  - [ ] SubTask 3.3: `RechargeWaffo` 同 3.1，`quotaToAdd`
  - [ ] SubTask 3.4: `RechargeWaffoPancake` 同 3.1
  - [ ] SubTask 3.5: `ManualCompleteTopUp`：事务内 `Update("quota", quotaAfter)` 后调；`ManualTopUpResult` 增 `CommissionSettled bool` 字段；提交后按结果记日志
  - [ ] SubTask 3.6: `controller/topup.go` 易支付 webhook：`model.IncreaseUserQuota` 成功后，包 `model.DB.Transaction` 调 `SettleRechargeCommission`（仅当刚翻状态成功时进入）；记日志
  - [ ] SubTask 3.7: `go build ./...` 通过

- [ ] Task 4: 互斥——分佣用户不发 100 额度邀请奖励
  - [ ] SubTask 4.1: `model/campaign.go:TrySettleDelayedInviteReward`：在 `lockForUpdate(tx).First(&inviter, ...)` 后读 `inviter.CommissionApproved`；为 true 时仍清被邀请人 `invite_reward_pending`、仍发被邀请人 ¥100 promo，但跳过邀请人 ¥100 发放与 `RewardedInviteCount++`，仅 `AffCount++`
  - [ ] SubTask 4.2: `model/user.go:inviteUser`（非延迟即时路径）：开头锁 inviter 后读 `CommissionApproved`，为 true 时仅 `AffCount++` 返回，不发 `QuotaForInviter`
  - [ ] SubTask 4.3: `go build ./...` 通过

- [ ] Task 5: 管理员批准入口（后端）
  - [ ] SubTask 5.1: `model/user.go:EditWithTx` 的 `updates` map 新增 `"commission_approved": newUser.CommissionApproved`
  - [ ] SubTask 5.2: 确认 `controller/user.go:UpdateUser` 已有 `canManageTargetRole` 校验，无需额外权限；确认 `UpdateSelf`/`UpdateUserSetting` 不写该字段（白名单已隔离）
  - [ ] SubTask 5.3: `controller/user.go:GetSelf` 返回体增加 `"commission_approved": user.CommissionApproved`
  - [ ] SubTask 5.4: `go build ./...` 通过

- [ ] Task 6: 后端测试
  - [ ] SubTask 6.1: `model/commission_test.go` 新增 `TestSettleRechargeCommission`：分佣获批 inviter + 充值 → 佣金 = round(quotaAdded*0.25) 入 `AffQuota`/`AffHistoryQuota`；未获批 inviter → 不发；无 inviter → 不发；自邀 → 不发
  - [ ] SubTask 6.2: 新增幂等用例：对同一充值重复调 `SettleRechargeCommission`（模拟翻状态后的二次进入）不应双发（由调用方状态门控保证，测试覆盖「quotaAdded<=0 跳过」与「未获批跳过」分支）
  - [ ] SubTask 6.3: 新增饱和用例：超大 `quotaAdded`（接近 int32 上限）*0.25 经 `QuotaFromFloatChecked` 饱和且 `clamp` 非 nil，不溢出为负
  - [ ] SubTask 6.4: `model/campaign_test.go` 新增/扩展：分佣获批 inviter 的 `TrySettleDelayedInviteReward` 不发邀请人 ¥100、不增 `RewardedInviteCount`，但被邀请人仍得 ¥100 promo
  - [ ] SubTask 6.5: `go test ./model/... -run 'Commission|DelayedInvite|Settle'` 通过

- [ ] Task 7: 前端——管理员编辑抽屉批准开关
  - [ ] SubTask 7.1: `web/default/src/features/users/types.ts` `userSchema` 增 `commission_approved: z.boolean().optional()`；`UserFormData` 增 `commission_approved?: boolean`
  - [ ] SubTask 7.2: `users-mutate-drawer.tsx`：仅 `isUpdate` 且当前用户为管理员时，渲染 `Switch`「分佣计划成员」，绑定 `commission_approved`，描述「获批后，该用户邀请的人充值时，其可获 25% 佣金（替代固定 100 额度邀请奖励）」
  - [ ] SubTask 7.3: `transformFormDataToPayload`：update 模式下携带 `commission_approved`
  - [ ] SubTask 7.4: `users-columns.tsx`：`commission_approved` 为 true 时展示「分佣」徽标（可选，便于识别）

- [ ] Task 8: 前端——用户钱包可见性与系统设置
  - [ ] SubTask 8.1: 钱包邀请/佣金卡片展示 `commission_approved` 状态与当前 `AffCommissionRate`（需从 `/api/option` 或 `GetSelf` 透出比例；若不便，仅显示「分佣计划成员」）
  - [ ] SubTask 8.2: `web/default/src/features/system-settings` 在 `QuotaForInviter`/`QuotaForInvitee` 同区新增数值输入「邀请充值佣金比例（0~1）」提交 `AffCommissionRate`
  - [ ] SubTask 8.3: `web/default/src/i18n/locales/en.json`、`zh.json` 新增开关/描述/徽标/设置项文案（英文为 key）；在 `web/default` 下 `bun run i18n:sync` 同步 fr/ru/ja/vi
  - [ ] SubTask 8.4: `cd web/default && bun run typecheck` 通过
  - [ ] SubTask 8.5: `cd web/default && bun run build` 通过

- [ ] Task 9: 验证与回归
  - [ ] SubTask 9.1: `go build ./...` 通过
  - [ ] SubTask 9.2: `go test ./model/...` 通过（含既有 `campaign_test.go`/`redemption` 等不受影响）
  - [ ] SubSub 9.3: 端到端（手动）：管理员在编辑抽屉给用户 A 开启分佣 → A 邀请 B → B 充值 X → A 的 `AffQuota` 增加 round(X*0.25)，B 的 `Quota` 增加 X；A 不再得固定 ¥100 邀请奖励，B 仍得 ¥100 promo
  - [ ] SubTask 9.4: 端到端（手动）：未开启分佣的普通邀请人 C 邀请 D → D 充值 → C 不得佣金，C 仍走原 ¥100 邀请奖励路径（受 `MaxValidInvites` 上限）
  - [ ] SubTask 9.5: 端到端（手动）：易支付 webhook 重试不双发佣金

# Task Dependencies

- Task 1 → Task 2（结算依赖 `AffCommissionRate` 与 `CommissionApproved` 字段）
- Task 2 → Task 3（各入口调用 `SettleRechargeCommission`）
- Task 1 → Task 4（互斥依赖 `CommissionApproved` 字段）
- Task 1 → Task 5（`EditWithTx` 白名单与 `GetSelf` 依赖字段）
- Task 2、Task 4 → Task 6（测试覆盖结算与互斥）
- Task 5 → Task 7（前端开关依赖后端字段与 `UpdateUser` 支持）
- Task 1、Task 5 → Task 8（钱包展示依赖 `GetSelf` 透出；系统设置依赖 option 持久化）
- Task 8 依赖 Task 7 文案确定（i18n 同步）
- Task 9 依赖全部完成
