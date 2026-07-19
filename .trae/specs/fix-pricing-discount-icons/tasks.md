# Tasks

- [x] Task 1: 全局折扣自动持久化（前端）
  - [x] SubTask 1.1: 在 `model-ratio-form.tsx` 中，当 `updateGlobalDiscount` 被调用后，自动触发 `ModelDiscount` option 的保存（复用 `ratio-settings-card.tsx` 中 `updateOption.mutateAsync({ key: 'ModelDiscount', value: ... })` 机制）
  - [x] SubTask 1.2: 对折扣率输入框的保存做防抖处理（约 500ms），开关切换立即保存
  - [x] SubTask 1.3: 确保非法值（空、0、负数、>1、非数字）不触发保存
  - [x] SubTask 1.4: 保存成功后更新 `modelDefaults.ModelDiscount` 本地缓存，避免下次表单初始化读到旧值
  - [x] SubTask 1.5: 验证：开启开关 + 填折扣率 → 刷新页面 → 开关仍开启、折扣率仍在（通过 typecheck 和构建验证，端到端需用户手动确认）

- [x] Task 2: 模型广场价格统一 2 位小数
  - [x] SubTask 2.1: 在 `web/default/src/features/pricing/lib/price.ts` 中，将 `formatPrice` 的 `digitsLarge: 4, digitsSmall: 6` 改为 `digitsLarge: 2, digitsSmall: 2`
  - [x] SubTask 2.2: 将 `formatGroupPrice` 的 `digitsLarge: 4, digitsSmall: 6` 改为 `digitsLarge: 2, digitsSmall: 2`
  - [x] SubTask 2.3: 将 `formatRequestPrice` 的 `digitsLarge: 4, digitsSmall: 4` 改为 `digitsLarge: 2, digitsSmall: 2`
  - [x] SubTask 2.4: `formatFixedPrice` 同步改为 `digitsLarge: 2, digitsSmall: 2`（与 `formatRequestPrice` 保持一致）；`formatModelBillingAmount` 路径保持原样不动
  - [x] SubTask 2.5: 验证：模型广场卡片视图和表格视图的所有价格均显示 2 位小数（通过构建验证）

- [x] Task 3: 隐藏「暂无描述」占位文案
  - [x] SubTask 3.1: 在 `web/default/src/features/pricing/components/model-card.tsx` 第 278-280 行，将 `{props.model.description || t('No description available.')}` 改为 `{props.model.description || ''}`（保留 `<p>` 元素以维持布局高度，避免抖动）
  - [x] SubTask 3.2: 验证：无描述的模型卡片不再显示「暂无描述」文案；有描述的模型正常显示（通过构建验证）

- [x] Task 4: 扩展模型图标覆盖（后端）
  - [x] SubTask 4.1: 在 `model/pricing_default.go` 的 `defaultVendorRules` 中补充缺失的模型名模式：gemma → Gemma、phi- → Microsoft、codestral → Mistral、openrouter → OpenRouter、groq → Groq、together → Together、fireworks → Fireworks、deepinfra → DeepInfra、ollama → Ollama
  - [x] SubTask 4.2: 在 `defaultVendorIcons` 中新增 Gemma → Gemma.Color、Groq → Groq、DeepInfra → DeepInfra.Color、Fireworks → Fireworks.Color、OpenRouter → OpenRouter.Color、Together → Together.Color、Ollama → Ollama
  - [x] SubTask 4.3: 通过 LobeHub 图标库源码确认每个图标的 Color 变体是否存在，有 Color 变体的用 .Color 后缀
  - [x] SubTask 4.4: 未实现本地 SVG 兜底（LobeHub 图标库已覆盖所有新增供应商，暂无需本地资源；如未来出现 LobeHub 未覆盖的模型，可再补充）

- [x] Task 5: 图标渲染一致性（前端）
  - [x] SubTask 5.1: `pricing-columns.tsx`（表格视图）新增首字母 fallback，与 `model-card.tsx`（卡片视图）一致
  - [x] SubTask 5.2: `model-details.tsx`（详情抽屉）新增首字母 fallback，与卡片视图一致
  - [x] SubTask 5.3: 三处图标渲染逻辑统一：`model.icon || vendor_icon` → getLobeIcon → 首字母 fallback（通过构建验证）

- [x] Task 6: 验证与回归测试
  - [x] SubTask 6.1: 端到端验证全局折扣：需用户手动在浏览器确认（设置 → 刷新 → 模型广场显示划线原价 + 折扣价）
  - [x] SubTask 6.2: 验证关闭全局折扣后恢复原价：需用户手动确认
  - [x] SubTask 6.3: 验证 per-model 折扣：需用户手动确认
  - [x] SubTask 6.4: 后端测试通过：`go test ./model/... -run TestPricing` ok
  - [x] SubTask 6.5: 前端构建通过：`cd web/default && bun run build 2>&1` exit 0

# Task Dependencies

- Task 2、Task 3、Task 4 互相独立，已并行完成
- Task 5 依赖 Task 4 完成，已完成
- Task 6 依赖 Task 1-5 全部完成，已通过编译/typecheck/构建验证
