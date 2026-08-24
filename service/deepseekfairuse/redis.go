package deepseekfairuse

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrRedisUnavailable = errors.New("deepseek fair-use redis unavailable")

type Limiter struct {
	client      *redis.Client
	runOverride func(context.Context, string, string, string, RiskSignals, bool) (Decision, error)
}

func New(client *redis.Client) *Limiter {
	return &Limiter{client: client}
}

type Lease struct {
	limiter  *Limiter
	account  string
	id       string
	mu       sync.Mutex
	released bool
}

func (l *Limiter) Acquire(ctx context.Context, accountHMAC string, leaseID string, risk RiskSignals) (*Lease, Decision, error) {
	if accountHMAC == "" || leaseID == "" {
		return nil, Decision{}, errors.New("deepseek fair-use account and lease are required")
	}
	decision, err := l.run(ctx, "acquire", accountHMAC, leaseID, risk, false)
	if err != nil {
		return nil, Decision{}, err
	}
	if !decision.Allowed {
		return nil, decision, nil
	}
	return &Lease{limiter: l, account: accountHMAC, id: leaseID}, decision, nil
}

func (l *Limiter) run(ctx context.Context, action string, accountHMAC string, leaseID string, risk RiskSignals, completed bool) (Decision, error) {
	if l == nil {
		return Decision{}, ErrRedisUnavailable
	}
	if l.runOverride != nil {
		return l.runOverride(ctx, action, accountHMAC, leaseID, risk, completed)
	}
	if l.client == nil {
		return Decision{}, ErrRedisUnavailable
	}
	values, err := fairUseRedisScript.Run(ctx, l.client, accountKeys(accountHMAC),
		action,
		leaseID,
		risk.IPHMAC,
		risk.CountryHMAC,
		risk.UserAgentHMAC,
		boolArgument(completed),
	).Result()
	if err != nil {
		return Decision{}, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}
	return decodeDecision(values)
}

func (l *Lease) Heartbeat(ctx context.Context) error {
	if l == nil || l.limiter == nil {
		return ErrRedisUnavailable
	}
	decision, err := l.limiter.run(ctx, "heartbeat", l.account, l.id, RiskSignals{}, false)
	if err != nil {
		return err
	}
	if decision.Allowed {
		return nil
	}
	if decision.Reason == ReasonLeaseNotFound {
		return ErrLeaseNotFound
	}
	return fmt.Errorf("deepseek fair-use heartbeat rejected: %s", decision.Reason)
}

func (l *Lease) Release(ctx context.Context, completed bool) (Snapshot, error) {
	if l == nil || l.limiter == nil {
		return Snapshot{}, ErrRedisUnavailable
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	decision, err := l.limiter.run(ctx, "release", l.account, l.id, RiskSignals{}, completed && !l.released)
	if err != nil {
		return Snapshot{}, err
	}
	l.released = true
	return decision.Snapshot, nil
}

// StartHeartbeat keeps a lease alive until the returned stop function is
// called. A heartbeat failure stops the loop and is surfaced to the caller so
// the relay can cancel the upstream request instead of silently exceeding the
// global concurrency budget.
func (l *Lease) StartHeartbeat(ctx context.Context, onError func(error)) func() {
	if l == nil {
		return func() {}
	}
	hbCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(HeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				if err := l.Heartbeat(hbCtx); err != nil {
					if onError != nil {
						onError(err)
					}
					return
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func accountKeys(accountHMAC string) []string {
	prefix := "deepseek:fup:v1:{" + accountHMAC + "}"
	return []string{
		prefix + ":live",
		prefix + ":lease_expiry",
		prefix + ":window",
		prefix + ":total",
		prefix + ":success",
		prefix + ":penalty",
		prefix + ":risk_ip",
		prefix + ":risk_country",
		prefix + ":risk_ua",
		prefix + ":penalty:events",
	}
}

func boolArgument(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func decodeDecision(raw interface{}) (Decision, error) {
	values, ok := raw.([]interface{})
	if !ok || len(values) < 11 {
		return Decision{}, fmt.Errorf("invalid deepseek fair-use response: %T", raw)
	}
	numbers := make([]int64, 0, 10)
	for _, index := range []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		value, err := valueInt64(values[index])
		if err != nil {
			return Decision{}, fmt.Errorf("invalid deepseek fair-use field %d: %w", index, err)
		}
		numbers = append(numbers, value)
	}
	return Decision{
		Allowed:         numbers[0] == 1,
		Reason:          Reason(valueString(values[1])),
		NotifyRootAdmin: numbers[7] == 1,
		RiskMarked:      numbers[8] == 1,
		Snapshot: Snapshot{
			Active:               int(numbers[1]),
			ConcurrentSeconds:    numbers[2],
			Admitted:             int(numbers[3]),
			Successful:           int(numbers[4]),
			EffectiveConcurrency: int(numbers[5]),
			Degraded:             numbers[5] == DegradedConcurrency,
			DegradeUntil:         unixSeconds(numbers[6]),
			ExhaustionEvents:     int(numbers[9]),
		},
	}, nil
}

func valueString(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}

func valueInt64(value interface{}) (int64, error) {
	switch value := value.(type) {
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case string:
		return parseNumericString(value)
	case []byte:
		return parseNumericString(string(value))
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, fmt.Errorf("invalid fair-use numeric value %v", value)
		}
		return int64(value), nil
	default:
		return 0, fmt.Errorf("invalid fair-use numeric value %T", value)
	}
}

func parseNumericString(value string) (int64, error) {
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, fmt.Errorf("invalid fair-use numeric value %q", value)
	}
	return int64(parsed), nil
}

func unixSeconds(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.Unix(value, 0)
}
