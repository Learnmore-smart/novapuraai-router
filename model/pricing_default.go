package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type vendorRule struct {
	pattern string
	vendor  string
}

// defaultVendorPrefixes maps the org segment of `vendor/model` names before
// substring rules run, so nvidia/llama-* is NVIDIA rather than Meta.
var defaultVendorPrefixes = map[string]string{
	"01-ai":             "零一万物",
	"adept":             "Adept",
	"ai21":              "AI21",
	"ai21labs":          "AI21",
	"aisingapore":       "AI Singapore",
	"bigcode":           "BigCode",
	"black-forest-labs": "Black Forest Labs",
	"databricks":        "Databricks",
	"deepseek-ai":       "DeepSeek",
	"google":            "Google",
	"huggingface":       "Hugging Face",
	"ibm":               "IBM",
	"ibm-granite":       "IBM",
	"meta":              "Meta",
	"meta-llama":        "Meta",
	"mistralai":         "Mistral",
	"nvidia":            "NVIDIA",
	"writer":            "Writer",
	"zyphra":            "Zyphra",
}

// defaultVendorRules is ordered from more specific to more generic contains
// matches. A map would iterate in random order and mis-attribute prefixed
// names such as nvidia/llama-3.1-nemotron.
var defaultVendorRules = []vendorRule{
	{pattern: "@cf/", vendor: "Cloudflare"},
	{pattern: "openrouter", vendor: "OpenRouter"},
	{pattern: "deepinfra", vendor: "DeepInfra"},
	{pattern: "fireworks", vendor: "Fireworks"},
	{pattern: "together", vendor: "Together"},
	{pattern: "codestral", vendor: "Mistral"},
	{pattern: "mistral", vendor: "Mistral"},
	{pattern: "nemotron", vendor: "NVIDIA"},
	{pattern: "cosmos", vendor: "NVIDIA"},
	{pattern: "ising", vendor: "NVIDIA"},
	{pattern: "bevformer", vendor: "NVIDIA"},
	{pattern: "sparsedrive", vendor: "NVIDIA"},
	{pattern: "streampetr", vendor: "NVIDIA"},
	{pattern: "riva", vendor: "NVIDIA"},
	{pattern: "synthetic-video", vendor: "NVIDIA"},
	{pattern: "starcoder", vendor: "BigCode"},
	{pattern: "sea-lion", vendor: "AI Singapore"},
	{pattern: "dall-e", vendor: "OpenAI"},
	{pattern: "whisper", vendor: "OpenAI"},
	{pattern: "chatgpt", vendor: "OpenAI"},
	{pattern: "chatglm", vendor: "智谱"},
	{pattern: "moonshot", vendor: "Moonshot"},
	{pattern: "deepseek", vendor: "DeepSeek"},
	{pattern: "minimax", vendor: "MiniMax"},
	{pattern: "hunyuan", vendor: "腾讯"},
	{pattern: "command", vendor: "Cohere"},
	{pattern: "laguna", vendor: "Poolside"},
	{pattern: "sarvam", vendor: "Sarvam"},
	{pattern: "doubao", vendor: "字节跳动"},
	{pattern: "jimeng", vendor: "即梦"},
	{pattern: "ollama", vendor: "Ollama"},
	{pattern: "gemini", vendor: "Google"},
	{pattern: "claude", vendor: "Anthropic"},
	{pattern: "gemma", vendor: "Gemma"},
	{pattern: "llama", vendor: "Meta"},
	{pattern: "jamba", vendor: "AI21"},
	{pattern: "fuyu", vendor: "Adept"},
	{pattern: "zamba", vendor: "Zyphra"},
	{pattern: "dbrx", vendor: "Databricks"},
	{pattern: "ernie", vendor: "百度"},
	{pattern: "spark", vendor: "讯飞"},
	{pattern: "kling", vendor: "快手"},
	{pattern: "glm-", vendor: "智谱"},
	{pattern: "qwen", vendor: "阿里巴巴"},
	{pattern: "kimi", vendor: "Moonshot"},
	{pattern: "jina", vendor: "Jina"},
	{pattern: "grok", vendor: "xAI"},
	{pattern: "step", vendor: "Step"},
	{pattern: "vidu", vendor: "Vidu"},
	{pattern: "phi-", vendor: "Microsoft"},
	{pattern: "groq", vendor: "Groq"},
	{pattern: "abab", vendor: "MiniMax"},
	{pattern: "gpt", vendor: "OpenAI"},
	{pattern: "o1", vendor: "OpenAI"},
	{pattern: "o3", vendor: "OpenAI"},
	{pattern: "360", vendor: "360"},
	{pattern: "yi", vendor: "零一万物"},
}

// 供应商默认图标映射
// Values are either @lobehub/icons component specs (e.g. "OpenAI",
// "Claude.Color") or local image paths under /model-icons/ (e.g.
// "/model-icons/databricks.svg"). The frontend getLobeIcon resolver detects
// paths starting with "/" and renders an <img> tag instead of a LobeHub
// component. Prefer lobehub names when the pack already has the brand.
var defaultVendorIcons = map[string]string{
	"OpenAI":            "OpenAI",
	"Anthropic":         "Claude.Color",
	"Google":            "Gemini.Color",
	"Moonshot":          "Moonshot",
	"智谱":                "Zhipu.Color",
	"阿里巴巴":              "Qwen.Color",
	"DeepSeek":          "DeepSeek.Color",
	"MiniMax":           "Minimax.Color",
	"百度":                "Wenxin.Color",
	"讯飞":                "Spark.Color",
	"腾讯":                "Hunyuan.Color",
	"Cohere":            "Cohere.Color",
	"Cloudflare":        "Cloudflare.Color",
	"360":               "Ai360.Color",
	"零一万物":              "Yi.Color",
	"Jina":              "Jina",
	"Mistral":           "Mistral.Color",
	"xAI":               "XAI",
	"Meta":              "/model-icons/meta.svg",
	"Poolside":          "/model-icons/poolside.svg",
	"NVIDIA":            "/model-icons/nvidia.svg",
	"Step":              "/model-icons/step.png",
	"Sarvam":            "/model-icons/sarvam-ai-logo.png",
	"字节跳动":              "Doubao.Color",
	"快手":                "Kling.Color",
	"即梦":                "Jimeng.Color",
	"Vidu":              "Vidu",
	"微软":                "AzureAI",
	"Microsoft":         "AzureAI",
	"Azure":             "AzureAI",
	"Gemma":             "Gemma.Color",
	"Groq":              "Groq",
	"DeepInfra":         "DeepInfra.Color",
	"Fireworks":         "Fireworks.Color",
	"OpenRouter":        "OpenRouter.Color",
	"Together":          "Together.Color",
	"Ollama":            "Ollama",
	"Adept":             "/model-icons/adept.png",
	"AI21":              "Ai21.Color",
	"AI Singapore":      "/model-icons/aisingapore.png",
	"BigCode":           "HuggingFace.Color",
	"Black Forest Labs": "Flux.Color",
	"Databricks":        "/model-icons/databricks.svg",
	"Hugging Face":      "HuggingFace.Color",
	"IBM":               "/model-icons/ibm.svg",
	"Writer":            "/model-icons/writer.png",
	"Zyphra":            "/model-icons/zyphra.png",
}

// initDefaultVendorMapping 简化的默认供应商映射
func initDefaultVendorMapping(metaMap map[string]*Model, vendorMap map[int]*Vendor, enableAbilities []AbilityWithChannel) {
	for _, ability := range enableAbilities {
		modelName := ability.Model
		if existing, exists := metaMap[modelName]; exists {
			if existing.VendorID != 0 {
				continue
			}
			vendorName, ok := matchDefaultVendor(modelName)
			if !ok {
				continue
			}
			vendorID := getOrCreateVendor(vendorName, vendorMap)
			if vendorID == 0 {
				continue
			}
			existing.VendorID = vendorID
			if DB != nil && existing.Id > 0 {
				_ = DB.Model(existing).Update("vendor_id", vendorID).Error
			}
			continue
		}

		vendorID := 0
		if vendorName, ok := matchDefaultVendor(modelName); ok {
			vendorID = getOrCreateVendor(vendorName, vendorMap)
		}

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

func matchDefaultVendor(modelName string) (string, bool) {
	modelLower := strings.ToLower(modelName)
	if prefix, rest, ok := strings.Cut(modelLower, "/"); ok && rest != "" {
		if vendorName, found := defaultVendorPrefixes[prefix]; found {
			return vendorName, true
		}
	}
	for _, rule := range defaultVendorRules {
		if strings.Contains(modelLower, rule.pattern) {
			return rule.vendor, true
		}
	}
	return "", false
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
