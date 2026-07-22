# Spec: 修复「Launch-2026」优惠码兑换失败

## 问题描述

管理员对外宣传了一个可读优惠码 `Launch-2026`，用户输入后兑换，前端提示
「兑换失败，请稍后再试」（后端 i18n key `redeem.failed`）。期望：用户输入
`Launch-2026` 后成功获得对应额度。

## 根因定位

兑换链路：前端 `redeemTopupCode({ key: code })` → `POST /api/user/topup` →
`controller.TopUp` → `model.Redeem(req.Key, id)`。

- `controller/user.go` 的 `TopUp`：当 `model.Redeem` 返回任意错误时，统一返回
  `i18n.MsgRedeemFailed`（`redeem.failed` = 「兑换失败，请稍后重试」），不暴露
  细分原因（安全设计，见 controller/user.go:1316 注释）。
- `model/redemption.go` 的 `Redeem`：用 `WHERE \`key\` = ?` 查 `redemptions` 表，
  查不到则返回「无效的兑换码」→ 被上层吞并为 `ErrRedeemFailed`。
- `controller/redemption.go` 的 `AddRedemption`：**始终**用 `common.GetUUID()`
  生成 32 位 hex 作为 key（controller/redemption.go:159 `key := common.GetUUID()`），
  即使 JSON body 带了 `key` 字段也会被 `cleanRedemption.Key = key` 覆盖忽略。
- 前端管理员表单 `redemptions-mutate-drawer.tsx` 根本没有「Key/兑换码」输入框，
  `transformFormDataToPayload` 也不发送 `key`。

结论：系统**没有任何途径**（API 或 UI）创建一个人类可读的自定义 key（如
`Launch-2026`）。`redemptions` 表里不存在 `key='Launch-2026'` 的行，`Redeem`
查不到 → 返回「兑换失败」。`Redeem` 本身对「已存在的合法码」逻辑正确（见
`model/redemption_test.go` 全部通过），不是兑换流程的功能性 bug。

另外 `Redemption.Key` 列声明为 `char(32)`（定长）。即便手工塞入短码
`Launch-2026`（11 字符），在 PostgreSQL 上会被空格填充到 32 位，污染
`MaskedValueDisplay` 的复制/掩码（`redemptions-columns.tsx:145` 的
`slice(0,8)+'*'*16+slice(-8)` 假定 32 位），因此需要改为变长 `varchar`。

## 修复方案

允许管理员在「创建兑换码」时**可选**地指定一个自定义可读 key。留空则维持现状
（自动生成 UUID）。

### 后端

1. **`model/redemption.go`**
   - `Redemption.Key` 列类型 `char(32)` → `varchar(64)`（变长，支持可读码且不填充
     空格；GORM AutoMigrate 在 MySQL/PostgreSQL 上 `MODIFY/ALTER COLUMN`，SQLite
     为 TEXT 无影响。首次迁移后类型匹配，不会重复 ALTER）。
   - 新增 `IsRedemptionKeyTaken(key string) (bool, error)`：用 `Unscoped()` 查询
     所有行（含软删除），避免与软删除码的 unique index 冲突。属数据访问层，放 model。

2. **`controller/redemption.go`（`AddRedemption`）**
   - 绑定 JSON 后 `strings.TrimSpace(redemption.Key)`：
     - 非空时：要求 `redemption.Count == 1`，否则返回 `MsgRedemptionKeyRequiresSingle`；
       校验字符集 `^[A-Za-z0-9_-]+$` 与长度 3–64，否则 `MsgRedemptionKeyInvalid`；
       `IsRedemptionKeyTaken` 为真则 `MsgRedemptionKeyExists`；该次循环用自定义 key
       代替 `common.GetUUID()`。
     - 为空时：维持原逻辑（每次循环生成 UUID）。
   - 校验放在现有 `Count`/`Amount` 校验附近，insert 失败仍走现有
     `MsgRedemptionCreateFailed` 分支。

3. **i18n**（`i18n/keys.go` + `i18n/locales/{en,zh-CN,zh-TW}.yaml`）
   - 新增三条 key：`redemption.key_invalid`、`redemption.key_exists`、
     `redemption.key_requires_single`（en/zh-CN/zh-TW 三语）。

### 前端（web/default）

4. **`features/redemption-codes/types.ts`**：`RedemptionFormData` 增加可选 `key?: string`。
5. **`features/redemption-codes/constants.ts`**：`REDEMPTION_VALIDATION` 增
   `KEY_MIN_LENGTH: 3`、`KEY_MAX_LENGTH: 64`；`ERROR_MESSAGES`/`getRedemptionFormErrorMessages`
   增 `KEY_INVALID`、`KEY_REQUIRES_SINGLE`。
6. **`features/redemption-codes/lib/redemption-form.ts`**：schema 增可选 `key`
   （regex `[A-Za-z0-9_-]{3,64}`）；`REDEMPTION_FORM_DEFAULT_VALUES.key=''`；
   `transformFormDataToPayload` 仅在 `key` 非空时写入 payload。
7. **`features/redemption-codes/components/redemptions-mutate-drawer.tsx`**：仅创建模式
   增「Custom Code (optional)」输入框；描述说明「留空自动生成；仅数量为 1 时可用」。
8. **i18n**（`web/default/src/i18n/locales/{en,zh}.json`，并 `bun run i18n:sync`）：
   新增字段标签/描述/错误文案（英文为 key）。

### 测试

9. **`model/redemption_test.go`**：新增用例——用短自定义 key（如 `Launch-2026`）建码
   并 `Redeem` 成功（覆盖 char→varchar 后短码路径）；`IsRedemptionKeyTaken` true/false。

## 不在范围内

- 不改 `Update`（key 不可改，维持现状）。
- 不改 `Redeem` 的错误屏蔽策略（安全设计保留）。
- 不调整 `MaskedValueDisplay` 对短码的掩码显示（功能正常，仅美观略怪，避免过度设计）。
- 不处理 MySQL/PostgreSQL 大小写敏感差异（既有行为，超出本次范围）。

## 风险

- 列类型 `char(32)`→`varchar(64)` 迁移：GORM AutoMigrate 跨三库可处理；SQLite 无
  `ALTER COLUMN` 但本就存 TEXT，无影响。即便旧库未迁移，短码经 `char(32)` 填充后
  等值比较仍命中（三库 trailing space 语义一致），功能不破。
- 自定义 key 与既有 UUID key 命名空间无冲突（unique index 兜底）。
