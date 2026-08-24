package ratio_setting

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var modelPricingOptionKeys = map[string]struct{}{
	"ModelPrice":                   {},
	"ModelRatio":                   {},
	"CompletionRatio":              {},
	"CacheRatio":                   {},
	"CreateCacheRatio":             {},
	"ImageRatio":                   {},
	"AudioRatio":                   {},
	"AudioCompletionRatio":         {},
	"ModelDiscount":                {},
	"billing_setting.billing_mode": {},
	"billing_setting.billing_expr": {},
}

var numericModelPricingOptionKeys = map[string]struct{}{
	"ModelPrice":           {},
	"ModelRatio":           {},
	"CompletionRatio":      {},
	"CacheRatio":           {},
	"CreateCacheRatio":     {},
	"ImageRatio":           {},
	"AudioRatio":           {},
	"AudioCompletionRatio": {},
	"ModelDiscount":        {},
}

// IsModelPricingOptionKey reports whether an option changes public or runtime
// model pricing and therefore requires pricing-cache invalidation.
func IsModelPricingOptionKey(key string) bool {
	_, ok := modelPricingOptionKeys[key]
	return ok
}

// ValidateModelPricingJSON validates a JSON object whose keys are model names.
// Model names ending in a quote are rejected at write boundaries so malformed
// upstream identities cannot enter runtime pricing maps.
func ValidateModelPricingJSON(jsonStr string) error {
	var values map[string]json.RawMessage
	if err := common.Unmarshal([]byte(jsonStr), &values); err != nil {
		return err
	}
	if values == nil {
		return fmt.Errorf("model pricing JSON must be an object")
	}
	return validateModelPricingNames(values)
}

func validateModelPricingNames(values map[string]json.RawMessage) error {
	for modelName := range values {
		if strings.TrimSpace(modelName) == "" {
			return fmt.Errorf("model name must not be empty")
		}
		if strings.TrimSpace(modelName) != modelName {
			return fmt.Errorf("invalid model name %q: surrounding whitespace is not allowed", modelName)
		}
		if common.IsInvalidModelName(modelName) {
			return fmt.Errorf("invalid model name %q: trailing quote is not allowed", modelName)
		}
	}
	return nil
}

// RepairModelPricingJSON repairs only the exact malformed DeepSeek identity
// found in persisted pricing maps. A conflicting canonical value is never
// merged or overwritten.
func RepairModelPricingJSON(jsonStr string) (string, bool, error) {
	var values map[string]json.RawMessage
	if err := common.Unmarshal([]byte(jsonStr), &values); err != nil {
		return jsonStr, false, err
	}
	if values == nil {
		return jsonStr, false, fmt.Errorf("model pricing JSON must be an object")
	}

	malformedName := common.CanonicalDeepSeekV4Flash0731 + `"`
	malformedValue, malformedFound := values[malformedName]
	canonicalName := common.CanonicalDeepSeekV4Flash0731
	changed := false
	if malformedFound {
		if canonicalValue, canonicalFound := values[canonicalName]; canonicalFound {
			equal, err := equivalentJSONValues(canonicalValue, malformedValue)
			if err != nil {
				return jsonStr, false, err
			}
			if !equal {
				return jsonStr, false, fmt.Errorf("conflicting pricing values for %q and %q", malformedName, canonicalName)
			}
			delete(values, malformedName)
		} else {
			delete(values, malformedName)
			values[canonicalName] = malformedValue
		}
		changed = true
	}
	if err := validateModelPricingNames(values); err != nil {
		return jsonStr, false, err
	}
	if !changed {
		return jsonStr, false, nil
	}

	repaired, err := common.Marshal(values)
	if err != nil {
		return jsonStr, false, err
	}
	return string(repaired), true, nil
}

func equivalentJSONValues(first, second json.RawMessage) (bool, error) {
	var firstValue, secondValue any
	if err := common.Unmarshal(first, &firstValue); err != nil {
		return false, err
	}
	if err := common.Unmarshal(second, &secondValue); err != nil {
		return false, err
	}
	return reflect.DeepEqual(firstValue, secondValue), nil
}

// ValidateModelPricingOptionValue applies model-name validation only to
// options that are persisted as model-keyed pricing maps.
func ValidateModelPricingOptionValue(key, value string) error {
	if _, ok := modelPricingOptionKeys[key]; !ok {
		return nil
	}
	if err := ValidateModelPricingJSON(value); err != nil {
		return err
	}
	if _, ok := numericModelPricingOptionKeys[key]; !ok {
		return nil
	}

	values, err := parseModelPricingNumberMap(value)
	if err != nil {
		return err
	}
	for modelName, number := range values {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("%s[%q] must be finite", key, modelName)
		}
		switch key {
		case "ModelRatio":
			if number < 0 {
				return fmt.Errorf("%s[%q] must not be negative", key, modelName)
			}
		case "CompletionRatio":
			if number <= 0 {
				return fmt.Errorf("%s[%q] must be greater than zero", key, modelName)
			}
		case "ModelDiscount":
			if number <= 0 || number > 1 {
				return fmt.Errorf("%s[%q] must be greater than zero and at most one", key, modelName)
			}
		default:
			if number < 0 {
				return fmt.Errorf("%s[%q] must not be negative", key, modelName)
			}
		}
	}
	return nil
}

func normalizeModelPricingOptionValue(key, value string) (string, error) {
	if err := ValidateModelPricingOptionValue(key, value); err != nil {
		return value, err
	}
	normalized, _, err := RepairModelPricingOptionValue(key, value)
	if err != nil {
		return value, err
	}
	return normalized, nil
}

func parseModelPricingNumberMap(value string) (map[string]float64, error) {
	values := make(map[string]float64)
	if err := common.Unmarshal([]byte(value), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, fmt.Errorf("model pricing JSON must be an object")
	}
	return values, nil
}

// ValidateModelPricingOptionChanges validates the final state produced by a
// single or bulk settings write. Legacy token entries may continue to use the
// runtime completion fallback, but a new/changed token ratio, removal of a
// per-request price, or removal of an explicit completion ratio cannot create
// token pricing without a positive output ratio.
func ValidateModelPricingOptionChanges(changes, current map[string]string) error {
	for key, value := range changes {
		if err := ValidateModelPricingOptionValue(key, value); err != nil {
			return err
		}
	}

	finalValue := func(key string) string {
		if value, ok := changes[key]; ok {
			return value
		}
		if value, ok := current[key]; ok {
			return value
		}
		return "{}"
	}
	modelPrices, err := parseModelPricingNumberMap(finalValue("ModelPrice"))
	if err != nil {
		return fmt.Errorf("ModelPrice: %w", err)
	}
	modelRatios, err := parseModelPricingNumberMap(finalValue("ModelRatio"))
	if err != nil {
		return fmt.Errorf("ModelRatio: %w", err)
	}
	completionRatios, err := parseModelPricingNumberMap(finalValue("CompletionRatio"))
	if err != nil {
		return fmt.Errorf("CompletionRatio: %w", err)
	}

	currentPrices, _ := parseModelPricingNumberMap(current["ModelPrice"])
	currentRatios, _ := parseModelPricingNumberMap(current["ModelRatio"])
	currentCompletions, _ := parseModelPricingNumberMap(current["CompletionRatio"])
	for modelName, ratio := range modelRatios {
		if _, pricedPerRequest := modelPrices[modelName]; pricedPerRequest {
			continue
		}
		completion, hasCompletion := completionRatios[modelName]
		if hasCompletion && completion > 0 {
			continue
		}
		previousRatio, previouslyTokenPriced := currentRatios[modelName]
		_, previouslyPerRequest := currentPrices[modelName]
		_, previouslyExplicitCompletion := currentCompletions[modelName]
		becameTokenPriced := !previouslyTokenPriced || previousRatio != ratio || previouslyPerRequest
		removedCompletion := previouslyExplicitCompletion
		if becameTokenPriced || removedCompletion {
			return fmt.Errorf("CompletionRatio[%q] must be explicitly provided for token pricing", modelName)
		}
	}
	for modelName := range completionRatios {
		if _, hasRatio := modelRatios[modelName]; hasRatio {
			continue
		}
		if _, pricedPerRequest := modelPrices[modelName]; pricedPerRequest {
			continue
		}
		if _, changed := changes["CompletionRatio"]; changed {
			return fmt.Errorf("CompletionRatio[%q] requires a matching ModelRatio", modelName)
		}
	}
	return nil
}

// RepairModelPricingOptionValue repairs only persisted model-keyed pricing
// options. Non-pricing options pass through unchanged.
func RepairModelPricingOptionValue(key, value string) (string, bool, error) {
	if _, ok := modelPricingOptionKeys[key]; !ok {
		return value, false, nil
	}
	repaired, changed, err := RepairModelPricingJSON(value)
	if err != nil {
		return value, false, err
	}
	if key != "ModelRatio" && key != "CompletionRatio" {
		return repaired, changed, nil
	}

	values, err := parseModelPricingNumberMap(repaired)
	if err != nil {
		return value, false, err
	}
	if _, exists := values[common.CanonicalDeepSeekV4Flash0731]; !exists {
		if key == "ModelRatio" {
			values[common.CanonicalDeepSeekV4Flash0731] = 0.11
		} else {
			values[common.CanonicalDeepSeekV4Flash0731] = 3
		}
		changed = true
	}
	if !changed {
		return value, false, nil
	}
	encoded, err := common.Marshal(values)
	if err != nil {
		return value, false, err
	}
	return string(encoded), true, nil
}
