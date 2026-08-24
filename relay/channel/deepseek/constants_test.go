package deepseek

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelListContainsCanonicalDeepSeekV4Flash0731(t *testing.T) {
	assert.Contains(t, ModelList, "deepseek-v4-flash-0731")
	assert.NotContains(t, ModelList, "deepseek-v4-flash-0731\"")
	assert.Contains(t, ModelList, "deepseek-v4-flash-none")
	assert.Contains(t, ModelList, "deepseek-v4-flash-max")
}
