package ratio_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// modelDiscountMap stores per-model discount multipliers applied on top of the
// configured price. A value must stay within (0, 1]: 1 means no discount, 0.8
// bills the model at 80% of its configured price. Values outside that range
// are rejected on update and ignored on read so a bad option payload can never
// inflate a charge or drive it negative.
var modelDiscountMap = types.NewRWMap[string, float64]()

func isValidModelDiscount(rate float64) bool {
	// rate > 0 excludes NaN and negatives; rate <= 1 excludes +Inf and markups.
	return rate > 0 && rate <= 1
}

// GetModelDiscount returns the effective discount multiplier for a model and
// whether an active discount (a valid rate below 1) is configured.
func GetModelDiscount(name string) (float64, bool) {
	name = FormatMatchingModelName(name)
	rate, ok := modelDiscountMap.Get(name)
	if !ok || !isValidModelDiscount(rate) || rate == 1 {
		return 1, false
	}
	return rate, true
}

func ModelDiscount2JSONString() string {
	return modelDiscountMap.MarshalJSONString()
}

func UpdateModelDiscountByJSONString(jsonStr string) error {
	var parsed map[string]float64
	if err := common.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return err
	}
	for name, rate := range parsed {
		if !isValidModelDiscount(rate) {
			return fmt.Errorf("model discount for %q must be within (0, 1], got %v", name, rate)
		}
	}
	return types.LoadFromJsonStringWithCallback(modelDiscountMap, jsonStr, InvalidateExposedDataCache)
}

func GetModelDiscountCopy() map[string]float64 {
	return modelDiscountMap.ReadAll()
}
