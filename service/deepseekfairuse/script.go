package deepseekfairuse

import (
	_ "embed"

	"github.com/go-redis/redis/v8"
)

// FairUseScript is exported for contract tests and operational inspection.
// Account keys are built as deepseek:fup:v1:{account_hmac}:*.
//
//go:embed lua/deepseek_fair_use.lua
var FairUseScript string

var fairUseRedisScript = redis.NewScript(FairUseScript)
