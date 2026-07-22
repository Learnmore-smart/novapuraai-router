# Checklist

## 根因确认
- [x] 错误「兑换失败，请稍后重试」来自 `controller.TopUp` 的 `i18n.MsgRedeemFailed`（`redeem.failed`）
- [x] `model.Redeem` 对不存在的 key 返回「无效的兑换码」→ 被吞并为 `ErrRedeemFailed`
- [x] `AddRedemption` 始终用 `common.GetUUID()` 覆盖 key，前端表单无 key 字段 → 无法创建可读码
- [x] `Redeem` 对已存在合法码逻辑正确（既有测试全过），非兑换流程功能 bug
- [x] `char(32)` 定长列对短码有填充副作用，需改 `varchar(64)`

## 后端
- [ ] `Redemption.Key` gorm tag 改为 `type:varchar(64);uniqueIndex`
- [ ] 新增 `IsRedemptionKeyTaken`（`Unscoped()`，含软删除行）
- [ ] `AddRedemption` 支持可选自定义 key（trim 后非空才启用）
- [ ] 自定义 key 校验：`Count==1`、`^[A-Za-z0-9_-]+$`、长度 3–64、唯一性
- [ ] 循环内 customKey 非空则用之，否则 `GetUUID()`
- [ ] `i18n/keys.go` 新增 3 个 Msg 常量
- [ ] en/zh-CN/zh-TW yaml 各新增 3 条翻译
- [ ] `go build ./...` 通过

## 前端
- [ ] `RedemptionFormData` 增 `key?: string`
- [ ] `REDEMPTION_VALIDATION` 增 `KEY_MIN_LENGTH=3`/`KEY_MAX_LENGTH=64`
- [ ] `ERROR_MESSAGES` + `getRedemptionFormErrorMessages` 增 `KEY_INVALID`/`KEY_REQUIRES_SINGLE`
- [ ] schema 增可选 `key`（regex + optional/空串）
- [ ] `REDEMPTION_FORM_DEFAULT_VALUES.key=''`
- [ ] `transformFormDataToPayload` 仅 key trim 非空时写入
- [ ] 创建模式增「Custom Code (optional)」输入框 + 描述
- [ ] `count > 1` 时禁用自定义 key 输入并提示
- [ ] en.json/zh.json 新增文案；`bun run i18n:sync` 同步其余语言
- [ ] `bun run typecheck` 通过
- [ ] `bun run build` 通过

## 测试
- [ ] `TestRedeemCustomShortKey`：`Launch-2026` 建码 + Redeem 成功 + quota 增加
- [ ] `TestIsRedemptionKeyTaken`：存在/不存在/软删除 三情形
- [ ] `go test ./model/... -run 'TestRedeem|TestIsRedemptionKeyTaken|TestSearchRedemptions'` 通过

## 回归与边界
- [ ] 留空 key 仍走原 UUID 生成路径（向后兼容）
- [ ] `Count>1` + 自定义 key 被拒绝（清晰错误）
- [ ] 重复自定义 key 被拒绝（含软删除）
- [ ] 既有 UUID 码兑换不受影响
- [ ] 不改 `Update`（key 不可改）
- [ ] 不改 `Redeem` 错误屏蔽策略
- [ ] 端到端（手动）：建 `Launch-2026`(cny/1000) → 用户兑换成功

## 受保护信息
- [ ] 未触碰 new-api / QuantumNous 相关品牌、版权、模块路径等受保护信息
