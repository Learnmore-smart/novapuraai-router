package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// 简化的供应商映射规则
var defaultVendorRules = map[string]string{
	"gpt":        "OpenAI",
	"dall-e":     "OpenAI",
	"whisper":    "OpenAI",
	"o1":         "OpenAI",
	"o3":         "OpenAI",
	"claude":     "Anthropic",
	"gemini":     "Google",
	"moonshot":   "Moonshot",
	"kimi":       "Moonshot",
	"chatglm":    "智谱",
	"glm-":       "智谱",
	"qwen":       "阿里巴巴",
	"deepseek":   "DeepSeek",
	"abab":       "MiniMax",
	"minimax":    "MiniMax",
	"ernie":      "百度",
	"spark":      "讯飞",
	"hunyuan":    "腾讯",
	"command":    "Cohere",
	"@cf/":       "Cloudflare",
	"360":        "360",
	"yi":         "零一万物",
	"jina":       "Jina",
	"mistral":    "Mistral",
	"grok":       "xAI",
	"llama":      "Meta",
	"laguna":     "Poolside",
	"nemotron":   "NVIDIA",
	"step":       "Step",
	"sarvam":     "Sarvam",
	"doubao":     "字节跳动",
	"kling":      "快手",
	"jimeng":     "即梦",
	"vidu":       "Vidu",
	"gemma":      "Gemma",
	"phi-":       "Microsoft",
	"codestral":  "Mistral",
	"openrouter": "OpenRouter",
	"groq":       "Groq",
	"together":   "Together",
	"fireworks":  "Fireworks",
	"deepinfra":  "DeepInfra",
	"ollama":     "Ollama",
}

// 供应商默认图标映射
// Values are either @lobehub/icons component specs (e.g. "OpenAI",
// "Claude.Color") or local image paths under /model-icons/ (e.g.
// "/model-icons/meta.svg"). The frontend getLobeIcon resolver detects paths
// starting with "/" and renders an <img> tag instead of a LobeHub component.
var defaultVendorIcons = map[string]string{
	"OpenAI":     "OpenAI",
	"Anthropic":  "Claude.Color",
	"Google":     "Gemini.Color",
	"Moonshot":   "Moonshot",
	"智谱":         "Zhipu.Color",
	"阿里巴巴":       "Qwen.Color",
	"DeepSeek":   "DeepSeek.Color",
	"MiniMax":    "Minimax.Color",
	"百度":         "Wenxin.Color",
	"讯飞":         "Spark.Color",
	"腾讯":         "Hunyuan.Color",
	"Cohere":     "Cohere.Color",
	"Cloudflare": "Cloudflare.Color",
	"360":        "Ai360.Color",
	"零一万物":       "Yi.Color",
	"Jina":       "Jina",
	"Mistral":    "Mistral.Color",
	"xAI":        "XAI",
	"Meta":       "/model-icons/meta.svg",
	"Poolside":   "/model-icons/poolside.svg",
	"NVIDIA":     "/model-icons/nvidia-logo-horz.svg",
	"Step":       "/model-icons/step.png",
	"Sarvam":     "/model-icons/sarvam-ai-logo.png",
	"字节跳动":       "Doubao.Color",
	"快手":         "Kling.Color",
	"即梦":         "Jimeng.Color",
	"Vidu":       "Vidu",
	"微软":         "AzureAI",
	"Microsoft":  "AzureAI",
	"Azure":      "AzureAI",
	"Gemma":      "Gemma.Color",
	"Groq":       "Groq",
	"DeepInfra":  "DeepInfra.Color",
	"Fireworks":  "Fireworks.Color",
	"OpenRouter": "OpenRouter.Color",
	"Together":   "Together.Color",
	"Ollama":     "Ollama",
}

// initDefaultVendorMapping 简化的默认供应商映射
func initDefaultVendorMapping(metaMap map[string]*Model, vendorMap map[int]*Vendor, enableAbilities []AbilityWithChannel) {
	for _, ability := range enableAbilities {
		modelName := ability.Model
		if _, exists := metaMap[modelName]; exists {
			continue
		}

		// 匹配供应商
		vendorID := 0
		modelLower := strings.ToLower(modelName)
		for pattern, vendorName := range defaultVendorRules {
			if strings.Contains(modelLower, pattern) {
				vendorID = getOrCreateVendor(vendorName, vendorMap)
				break
			}
		}

		// 创建模型元数据
		metaMap[modelName] = &Model{
			ModelName: modelName,
			VendorID:  vendorID,
			Status:    1,
			NameRule:  NameRuleExact,
		}
	}
}

// 查找或创建供应商
func getOrCreateVendor(vendorName string, vendorMap map[int]*Vendor) int {
	// 查找现有供应商
	for id, vendor := range vendorMap {
		if vendor.Name == vendorName {
			return id
		}
	}

	// 创建新供应商
	newVendor := &Vendor{
		Name:   vendorName,
		Status: 1,
		Icon:   getDefaultVendorIcon(vendorName),
	}

	if err := newVendor.Insert(); err != nil {
		return 0
	}

	vendorMap[newVendor.Id] = newVendor
	return newVendor.Id
}

// 获取供应商默认图标
func getDefaultVendorIcon(vendorName string) string {
	if icon, exists := defaultVendorIcons[vendorName]; exists {
		return icon
	}
	return ""
}

// SyncDefaultVendorIcons updates existing Vendor rows so their icon column
// matches the current defaultVendorIcons map. This is called once on startup
// after migration so that changes to the default icon map (e.g. fixing the
// Meta -> Ollama mapping, or adding NVIDIA / Poolside / Step / Sarvam) are
// reflected without requiring manual DB edits. Vendors whose name is not in
// the default map are left untouched, preserving admin customizations for
// any vendor that was hand-added through the UI.
func SyncDefaultVendorIcons() {
	if DB == nil {
		return
	}
	var vendors []Vendor
	if err := DB.Find(&vendors).Error; err != nil {
		common.SysError(fmt.Sprintf("SyncDefaultVendorIcons load failed: %v", err))
		return
	}
	updated := 0
	for _, v := range vendors {
		defaultIcon, ok := defaultVendorIcons[v.Name]
		if !ok || defaultIcon == "" || v.Icon == defaultIcon {
			continue
		}
		if err := DB.Model(&Vendor{}).Where("id = ?", v.Id).Update("icon", defaultIcon).Error; err != nil {
			common.SysError(fmt.Sprintf("SyncDefaultVendorIcons update failed vendor=%s: %v", v.Name, err))
			continue
		}
		updated++
	}
	if updated > 0 {
		common.SysLog(fmt.Sprintf("SyncDefaultVendorIcons: updated %d vendor icon(s) to match defaults", updated))
	}
}
