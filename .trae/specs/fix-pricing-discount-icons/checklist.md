# Checklist

## 全局折扣持久化
- [x] 视觉模式下开启开关 + 填写折扣率后，自动保存到后端（`updateOption.mutateAsync`）
- [x] 视觉模式下关闭开关后，自动保存并移除 `*` 键（保留各模型原有 per-model 折扣率）
- [x] 折扣率输入框快速连续输入时，保存请求做防抖处理（500ms）
- [x] 非法值（空、0、负数、>1、非数字）不触发保存（`setGlobalDiscountDraft` 抛错拦截）
- [x] 保存后 `modelNormalizedDefaults.current.ModelDiscount` 和 `savedModelValues` 同步更新
- [ ] 端到端浏览器验证：刷新页面后开关仍开启、折扣率仍在（需用户手动确认）

## 模型广场价格 2 位小数
- [x] `formatPrice` 的 `formatCurrencyFromUSD` 调用使用 `digitsLarge: 2, digitsSmall: 2`
- [x] `formatGroupPrice` 的 `formatCurrencyFromUSD` 调用使用 `digitsLarge: 2, digitsSmall: 2`
- [x] `formatRequestPrice` 的 `formatCurrencyFromUSD` 调用使用 `digitsLarge: 2, digitsSmall: 2`
- [x] `formatFixedPrice` 的 `formatCurrencyFromUSD` 调用使用 `digitsLarge: 2, digitsSmall: 2`
- [x] `formatModelBillingAmount` 本地计费货币路径保持原样不动
- [x] 折扣百分比徽标仍为整数（`Math.round`，未修改）

## 隐藏「暂无描述」
- [x] `model-card.tsx` 第 279 行将 `t('No description available.')` 改为 `''`
- [x] 保留 `<p>` 元素和 className，维持卡片布局高度
- [x] 未删除 `t` 导入（其他地方仍在使用）
- [x] 未修改 i18n locale 文件
- [x] 未修改 `model-details.tsx`（本就不显示占位文案）

## 模型广场折扣显示
- [x] 划线原价 + 折扣后价 + 百分比徽标的渲染逻辑已存在（`model-card.tsx` 第 117-199 行、`pricing-columns.tsx`）
- [x] 修复全局折扣持久化后，API 将返回 `discount` 字段，折扣显示自动生效
- [ ] 端到端浏览器验证：设置全局折扣后模型广场显示划线原价 + 折扣价（需用户手动确认）

## 模型图标覆盖
- [x] `defaultVendorRules` 新增 9 条映射：gemma、phi-、codestral、openrouter、groq、together、fireworks、deepinfra、ollama
- [x] `defaultVendorIcons` 新增 7 条映射：Gemma.Color、Groq、DeepInfra.Color、Fireworks.Color、OpenRouter.Color、Together.Color、Ollama
- [x] 通过 LobeHub 图标库源码确认每个图标的 Color 变体存在性
- [x] `gemma` → 独立供应商 `Gemma`（使用 `Gemma.Color` 专属图标，非 Google/Gemini）
- [x] 后端编译通过：`go build ./...` exit 0
- [x] 后端测试通过：`go test ./model/... -run TestPricing` ok
- [ ] 端到端浏览器验证：gemma 模型在三处视图均显示 Gemma 图标（需用户手动确认）

## 图标渲染一致性
- [x] 卡片视图（`model-card.tsx`）：`model.icon || vendor_icon` → getLobeIcon(28) → 首字母 fallback
- [x] 表格视图（`pricing-columns.tsx`）：新增首字母 fallback，逻辑与卡片视图一致
- [x] 详情抽屉（`model-details.tsx`）：新增首字母 fallback，逻辑与卡片视图一致

## 构建与测试
- [x] `go build ./...` 成功通过
- [x] `go test ./model/... -run TestPricing` 通过
- [x] `bun run typecheck`（tsgo -b）通过
- [x] `bun run build 2>&1` 成功通过（exit 0，built in 22.2s）
- [ ] `model-global-discount.test.ts` 未运行（项目未配置 vitest/jest 测试运行器，该文件为类型检查用）
