package deepseekfairuse

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	PeakConcurrency           = 10
	DegradedConcurrency       = 1
	ConcurrentSecondsBudget   = int64(1800)
	SuccessRequestLimit       = 600
	AdmittedRequestLimit      = 750
	WindowDuration            = 10 * time.Minute
	StaleLeaseRecovery        = 45 * time.Second
	HeartbeatInterval         = 15 * time.Second
	ExhaustionStrikeThreshold = 3
	ExhaustionStrikeWindow    = time.Hour
	DegradationDuration       = 30 * time.Minute
	RiskSignalWindow          = 24 * time.Hour
	RiskIPThreshold           = 8
	RiskCountryThreshold      = 3
)

type Reason string

const (
	ReasonAlreadyAdmitted        Reason = "already_admitted"
	ReasonConcurrencyLimit       Reason = "concurrency_limit"
	ReasonConcurrentSecondsLimit Reason = "concurrent_seconds_limit"
	ReasonSuccessRequestLimit    Reason = "success_request_limit"
	ReasonAdmittedRequestLimit   Reason = "admitted_request_limit"
	ReasonLeaseNotFound          Reason = "lease_not_found"
)

type Decision struct {
	Allowed         bool
	Reason          Reason
	Snapshot        Snapshot
	NotifyRootAdmin bool
	RiskMarked      bool
}

type Snapshot struct {
	Active               int
	ConcurrentSeconds    int64
	Admitted             int
	Successful           int
	EffectiveConcurrency int
	Degraded             bool
	DegradeUntil         time.Time
	ExhaustionEvents     int
}

type clockSource interface {
	Now() time.Time
}

type memoryLease struct {
	lastHeartbeat time.Time
	lastCharged   time.Time
}

type occupancySegment struct {
	start time.Time
	end   time.Time
}

type memoryEvent struct {
	at time.Time
	id string
}

type memoryAccount struct {
	leases                  map[string]*memoryLease
	occupancy               []occupancySegment
	admitted                []memoryEvent
	successful              []memoryEvent
	exhaustionEvents        map[string]time.Time
	riskIPs                 map[string]time.Time
	riskCountries           map[string]time.Time
	riskUserAgents          map[string]time.Time
	forcedConcurrentSeconds int64
	degradeUntil            time.Time
	lastNotification        time.Time
	notificationPending     bool
}

type memoryLimiter struct {
	mu       sync.Mutex
	clock    clockSource
	accounts map[string]*memoryAccount
}

func newMemoryLimiter(clock clockSource) *memoryLimiter {
	return &memoryLimiter{clock: clock, accounts: make(map[string]*memoryAccount)}
}

func newMemoryAccount() *memoryAccount {
	return &memoryAccount{
		leases:           make(map[string]*memoryLease),
		exhaustionEvents: make(map[string]time.Time),
		riskIPs:          make(map[string]time.Time),
		riskCountries:    make(map[string]time.Time),
		riskUserAgents:   make(map[string]time.Time),
	}
}

func (m *memoryLimiter) account(key string) *memoryAccount {
	account, ok := m.accounts[key]
	if !ok {
		account = newMemoryAccount()
		m.accounts[key] = account
	}
	return account
}

func (m *memoryLimiter) acquire(accountKey string, leaseID string) Decision {
	return m.acquireWithRisk(accountKey, leaseID, RiskSignals{})
}

func (m *memoryLimiter) acquireWithRisk(accountKey string, leaseID string, risk RiskSignals) Decision {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now()
	account := m.account(accountKey)
	m.prune(account, now)
	m.chargeAll(account, now)
	riskMarked := m.recordRisk(account, risk, now)
	if _, exists := account.leases[leaseID]; exists {
		return m.decisionWithRisk(account, true, ReasonAlreadyAdmitted, false, riskMarked)
	}

	if len(account.leases) >= m.effectiveConcurrency(account, now) {
		return m.decisionWithRisk(account, false, ReasonConcurrencyLimit, false, riskMarked)
	}
	if m.concurrentSeconds(account, now) >= ConcurrentSecondsBudget {
		notify := m.recordExhaustion(account, ReasonConcurrentSecondsLimit, now)
		return m.decisionWithRisk(account, false, ReasonConcurrentSecondsLimit, notify, riskMarked)
	}
	if len(account.successful) >= SuccessRequestLimit {
		notify := m.recordExhaustion(account, ReasonSuccessRequestLimit, now)
		return m.decisionWithRisk(account, false, ReasonSuccessRequestLimit, notify, riskMarked)
	}
	if len(account.admitted) >= AdmittedRequestLimit {
		notify := m.recordExhaustion(account, ReasonAdmittedRequestLimit, now)
		return m.decisionWithRisk(account, false, ReasonAdmittedRequestLimit, notify, riskMarked)
	}

	account.leases[leaseID] = &memoryLease{lastHeartbeat: now, lastCharged: now}
	account.admitted = append(account.admitted, memoryEvent{at: now, id: leaseID})
	return m.decisionWithRisk(account, true, "", false, riskMarked)
}

func (m *memoryLimiter) heartbeat(accountKey string, leaseID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now()
	account := m.account(accountKey)
	m.prune(account, now)
	lease, ok := account.leases[leaseID]
	if !ok {
		return ErrLeaseNotFound
	}
	m.chargeLease(account, lease, now)
	if m.concurrentSeconds(account, now) >= ConcurrentSecondsBudget {
		m.recordExhaustion(account, ReasonConcurrentSecondsLimit, now)
		delete(account.leases, leaseID)
		return ErrConcurrentSecondsBudget
	}
	lease.lastHeartbeat = now
	return nil
}

func (m *memoryLimiter) release(accountKey string, leaseID string, completed bool) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now()
	account := m.account(accountKey)
	m.prune(account, now)
	lease, ok := account.leases[leaseID]
	if !ok {
		return m.snapshot(account, now)
	}
	m.chargeLease(account, lease, now)
	delete(account.leases, leaseID)
	if completed {
		account.successful = append(account.successful, memoryEvent{at: now, id: leaseID})
	}
	return m.snapshot(account, now)
}

func (m *memoryLimiter) prune(account *memoryAccount, now time.Time) {
	cutoff := now.Add(-WindowDuration)
	for leaseID, lease := range account.leases {
		if !lease.lastHeartbeat.Add(StaleLeaseRecovery).After(now) {
			end := lease.lastHeartbeat.Add(StaleLeaseRecovery)
			if end.After(now) {
				end = now
			}
			m.chargeLease(account, lease, end)
			delete(account.leases, leaseID)
		}
	}
	account.admitted = trimEvents(account.admitted, cutoff)
	account.successful = trimEvents(account.successful, cutoff)
	keptOccupancy := account.occupancy[:0]
	for _, segment := range account.occupancy {
		if !segment.end.After(cutoff) {
			continue
		}
		if segment.start.Before(cutoff) {
			segment.start = cutoff
		}
		keptOccupancy = append(keptOccupancy, segment)
	}
	account.occupancy = keptOccupancy
	for key, eventAt := range account.exhaustionEvents {
		if eventAt.Before(now.Add(-ExhaustionStrikeWindow)) {
			delete(account.exhaustionEvents, key)
		}
	}
	trimRiskSignals(account.riskIPs, now)
	trimRiskSignals(account.riskCountries, now)
	trimRiskSignals(account.riskUserAgents, now)
}

func trimEvents(events []memoryEvent, cutoff time.Time) []memoryEvent {
	kept := events[:0]
	for _, event := range events {
		if !event.at.Before(cutoff) {
			kept = append(kept, event)
		}
	}
	return kept
}

func trimRiskSignals(signals map[string]time.Time, now time.Time) {
	cutoff := now.Add(-RiskSignalWindow)
	for key, seenAt := range signals {
		if seenAt.Before(cutoff) {
			delete(signals, key)
		}
	}
}

func (m *memoryLimiter) recordRisk(account *memoryAccount, risk RiskSignals, now time.Time) bool {
	if risk.IPHMAC != "" {
		account.riskIPs[risk.IPHMAC] = now
	}
	if risk.CountryHMAC != "" {
		account.riskCountries[risk.CountryHMAC] = now
	}
	if risk.UserAgentHMAC != "" {
		account.riskUserAgents[risk.UserAgentHMAC] = now
	}
	return len(account.riskIPs) >= RiskIPThreshold || len(account.riskCountries) >= RiskCountryThreshold
}

func (m *memoryLimiter) chargeAll(account *memoryAccount, now time.Time) {
	for _, lease := range account.leases {
		m.chargeLease(account, lease, now)
	}
}

func (m *memoryLimiter) chargeLease(account *memoryAccount, lease *memoryLease, end time.Time) {
	if !end.After(lease.lastCharged) {
		return
	}
	account.occupancy = append(account.occupancy, occupancySegment{start: lease.lastCharged, end: end})
	lease.lastCharged = end
}

func (m *memoryLimiter) concurrentSeconds(account *memoryAccount, now time.Time) int64 {
	seconds := account.forcedConcurrentSeconds
	cutoff := now.Add(-WindowDuration)
	for _, segment := range account.occupancy {
		start := segment.start
		if start.Before(cutoff) {
			start = cutoff
		}
		end := segment.end
		if end.After(now) {
			end = now
		}
		if end.After(start) {
			seconds += int64(end.Sub(start) / time.Second)
		}
	}
	if seconds < 0 {
		return 0
	}
	return seconds
}

func (m *memoryLimiter) recordExhaustion(account *memoryAccount, reason Reason, now time.Time) bool {
	key := fmt.Sprintf("%s:%d", reason, now.Unix()/int64(WindowDuration/time.Second))
	if _, exists := account.exhaustionEvents[key]; exists {
		return false
	}
	account.exhaustionEvents[key] = now
	if len(account.exhaustionEvents) < ExhaustionStrikeThreshold || account.degradeUntil.After(now) {
		return false
	}
	account.degradeUntil = now.Add(DegradationDuration)
	if account.lastNotification.IsZero() || !account.lastNotification.Add(ExhaustionStrikeWindow).After(now) {
		account.lastNotification = now
		account.notificationPending = true
		return true
	}
	return false
}

func (m *memoryLimiter) effectiveConcurrency(account *memoryAccount, now time.Time) int {
	if account.degradeUntil.After(now) {
		return DegradedConcurrency
	}
	return PeakConcurrency
}

func (m *memoryLimiter) effectiveConcurrencyFor(accountKey string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	account := m.account(accountKey)
	return m.effectiveConcurrency(account, m.clock.Now())
}

func (m *memoryLimiter) snapshot(account *memoryAccount, now time.Time) Snapshot {
	return Snapshot{
		Active:               len(account.leases),
		ConcurrentSeconds:    m.concurrentSeconds(account, now),
		Admitted:             len(account.admitted),
		Successful:           len(account.successful),
		EffectiveConcurrency: m.effectiveConcurrency(account, now),
		Degraded:             account.degradeUntil.After(now),
		DegradeUntil:         account.degradeUntil,
		ExhaustionEvents:     len(account.exhaustionEvents),
	}
}

func (m *memoryLimiter) decisionWithRisk(account *memoryAccount, allowed bool, reason Reason, notify bool, riskMarked bool) Decision {
	now := m.clock.Now()
	return Decision{
		Allowed:         allowed,
		Reason:          reason,
		Snapshot:        m.snapshot(account, now),
		NotifyRootAdmin: notify,
		RiskMarked:      riskMarked,
	}
}

func (m *memoryLimiter) setConcurrentSeconds(accountKey string, seconds int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.account(accountKey).forcedConcurrentSeconds = seconds
}

func (m *memoryLimiter) active(accountKey string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	account := m.account(accountKey)
	m.prune(account, m.clock.Now())
	return len(account.leases)
}

func (m *memoryLimiter) successes(accountKey string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	account := m.account(accountKey)
	m.prune(account, m.clock.Now())
	return len(account.successful)
}

func (m *memoryLimiter) degradeUntil(accountKey string) time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.account(accountKey).degradeUntil
}

func (m *memoryLimiter) notificationDue(accountKey string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	account := m.account(accountKey)
	if !account.notificationPending {
		return false
	}
	account.notificationPending = false
	return true
}

var (
	ErrLeaseNotFound           = errors.New("deepseek fair-use lease not found")
	ErrConcurrentSecondsBudget = errors.New("deepseek fair-use concurrent-seconds budget exhausted")
)
