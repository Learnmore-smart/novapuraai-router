package common

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type verificationValue struct {
	code string
	time time.Time
}

const (
	EmailVerificationPurpose = "v"
	PasswordResetPurpose     = "r"
)

var verificationMutex sync.Mutex
var verificationMap map[string]verificationValue

// Larger map so concurrent registrations don't evict fresh codes (MVP email verify).
var verificationMapMaxSize = 10000
var VerificationValidMinutes = 10

func GenerateVerificationCode(length int) string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	if length == 0 {
		return code
	}
	return code[:length]
}

func RegisterVerificationCodeWithKey(key string, code string, purpose string) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap[purpose+key] = verificationValue{
		code: code,
		time: time.Now(),
	}
	if len(verificationMap) > verificationMapMaxSize {
		removeExpiredPairs()
	}
}

func GetOrCreateVerificationCodeWithKey(key string, purpose string, length int) (string, bool) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()

	mapKey := purpose + key
	now := time.Now()
	if value, exists := verificationMap[mapKey]; exists && int(now.Sub(value.time).Seconds()) < VerificationValidMinutes*60 {
		return value.code, false
	}

	code := GenerateVerificationCode(length)
	verificationMap[mapKey] = verificationValue{code: code, time: now}
	if len(verificationMap) > verificationMapMaxSize {
		removeExpiredPairs()
	}
	return code, true
}

func VerifyCodeWithKey(key string, code string, purpose string) bool {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	mapKey := purpose + key
	value, okay := verificationMap[mapKey]
	now := time.Now()
	if !okay || int(now.Sub(value.time).Seconds()) >= VerificationValidMinutes*60 {
		return false
	}
	if code != value.code {
		return false
	}
	// Single-use: consume on success (MVP §5.4 / §8.1).
	delete(verificationMap, mapKey)
	return true
}

func DeleteKey(key string, purpose string) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	delete(verificationMap, purpose+key)
}

// no lock inside, so the caller must lock the verificationMap before calling!
func removeExpiredPairs() {
	now := time.Now()
	for key := range verificationMap {
		if int(now.Sub(verificationMap[key].time).Seconds()) >= VerificationValidMinutes*60 {
			delete(verificationMap, key)
		}
	}
}

func init() {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap = make(map[string]verificationValue)
}
