# Tasks

- [ ] Task 1: 后端模型与数据访问层
  - [ ] SubTask 1.1: `model/redemption.go` 将 `Redemption.Key` 的 gorm tag 由 `type:char(32);uniqueIndex` 改为 `type:varchar(64);uniqueIndex`
  - [ ] SubTask 1.2: `model/redemption.go` 新增 `IsRedemptionKeyTaken(key string) (bool, error)`，使用 `DB.Unscoped().Where(keyCol+" = ?", key)` 计数（含软删除行），key 列名按 PostgreSQL/其它库分别用 `"key"`/`` `key` ``
  - [ ] SubTask 1.3: `go build ./...` 通过

- [ ] Task 2: 后端控制器支持自定义 key
  - [ ] SubTask 2.1: `controller/redemption.go` 的 `AddRedemption` 在现有校验之后、循环之前，处理 `customKey := strings.TrimSpace(redemption.Key)`
  - [ ] SubTask 2.2: customKey 非空时校验 `Count == 1`（否则 `MsgRedemptionKeyRequiresSingle`）、字符集 `^[A-Za-z0-9_-]+$` 且长度 3–64（否则 `MsgRedemptionKeyInvalid`）、`IsRedemptionKeyTaken`（为真则 `MsgRedemptionKeyExists`）
  - [ ] SubTask 2.3: 循环内 `code := customKey` 非空则用之，否则 `common.GetUUID()`；将 `cleanRedemption.Key = code`
  - [ ] SubTask 2.4: `go build ./...` 通过

- [ ] Task 3: 后端 i18n
  - [ ] SubTask 3.1: `i18n/keys.go` 新增 `MsgRedemptionKeyInvalid`/`MsgRedemptionKeyExists`/`MsgRedemptionKeyRequiresSingle` 三个常量
  - [ ] SubTask 3.2: `i18n/locales/en.yaml`、`zh-CN.yaml`、`zh-TW.yaml` 各新增三条对应翻译

- [ ] Task 4: 前端类型与校验
  - [ ] SubTask 4.1: `features/redemption-codes/types.ts` 的 `RedemptionFormData` 增加可选 `key?: string`
  - [ ] SubTask 4.2: `features/redemption-codes/constants.ts` 的 `REDEMPTION_VALIDATION` 增 `KEY_MIN_LENGTH: 3`、`KEY_MAX_LENGTH: 64`；`ERROR_MESSAGES` 增 `KEY_INVALID`、`KEY_REQUIRES_SINGLE`；`getRedemptionFormErrorMessages` 增对应插值文案
  - [ ] SubTask 4.3: `features/redemption-codes/lib/redemption-form.ts` schema 增可选 `key`（`z.string().regex(/^[A-Za-z0-9_-]{3,64}$/).optional().or(z.literal(''))`），`REDEMPTION_FORM_DEFAULT_VALUES` 增 `key: ''`，`transformFormDataToPayload` 仅在 `data.key` trim 后非空时写入 `key`
  - [ ] SubTask 4.4: `RedemptionFormValues` 类型增 `key?: string`

- [ ] Task 5: 前端表单 UI
  - [ ] SubTask 5.1: `features/redemption-codes/components/redemptions-mutate-drawer.tsx` 在创建模式（`!isUpdate`）下、「Quantity」字段附近新增「Custom Code (optional)」`Input`，绑定 `key`
  - [ ] SubTask 5.2: 字段下 `FormDescription` 说明「留空自动生成随机码；仅当数量为 1 时生效」；当 `count > 1` 时禁用该输入并提示
  - [ ] SubTask 5.3: 提交逻辑（`handleSubmit`）无需改 key 自动填充逻辑（key 为空即自动生成）

- [ ] Task 6: 前端 i18n
  - [ ] SubTask 6.1: `web/default/src/i18n/locales/en.json`、`zh.json` 新增字段标签/描述/错误英文源串与中文翻译
  - [ ] SubTask 6.2: 在 `web/default` 下执行 `bun run i18n:sync` 同步其余语言（fr/ru/ja/vi/zh-TW）

- [ ] Task 7: 后端测试
  - [ ] SubTask 7.1: `model/redemption_test.go` 新增 `TestRedeemCustomShortKey`：用 `Launch-2026` 建 `Redemption`（Status=Enabled, Quota=1000）→ `Redeem("Launch-2026", user)` 成功且 user.Quota 增加
  - [ ] SubTask 7.2: 新增 `TestIsRedemptionKeyTaken`：存在/不存在/软删除三种情形
  - [ ] SubTask 7.3: `go test ./model/... -run 'TestRedeem|TestIsRedemptionKeyTaken|TestSearchRedemptions'` 通过

- [ ] Task 8: 验证与回归
  - [ ] SubTask 8.1: `go build ./...` 通过
  - [ ] SubTask 8.2: `cd web/default && bun run typecheck` 通过
  - [ ] SubTask 8.3: `cd web/default && bun run build` 通过
  - [ ] SubTask 8.4: 端到端（需用户手动）：管理员创建 key=`Launch-2026`、currency=cny、amount=1000、max_redeems>1 的码 → 普通用户输入 `Launch-2026` 兑换成功

# Task Dependencies

- Task 1 → Task 2（控制器依赖 `IsRedemptionKeyTaken`）
- Task 3 独立，与 Task 1/2 并行
- Task 4 → Task 5（UI 依赖类型/schema）
- Task 6 依赖 Task 5 文案确定
- Task 7 依赖 Task 1
- Task 8 依赖全部完成
