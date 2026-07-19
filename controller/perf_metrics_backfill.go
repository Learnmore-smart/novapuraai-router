package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

const (
	// perfMetricsBackfillLookbackHours defines the window we look back to decide
	// whether a model already has perf data. Models with at least one bucket in
	// this window are skipped. 24h matches the default model-square window so a
	// model that produced data today is not re-tested.
	perfMetricsBackfillLookbackHours = 24

	// perfMetricsBackfillMaxRetries caps how many distinct channels we try for a
	// single model before giving up. Each retry picks the next candidate channel
	// so a flaky upstream does not waste the whole retry budget.
	perfMetricsBackfillMaxRetries = 3

	// perfMetricsBackfillRetryDelay is the backoff between retries for the same
	// model. It is intentionally short because we already switch to a different
	// channel on each retry.
	perfMetricsBackfillRetryDelay = 5 * time.Second

	// perfMetricsBackfillRequestInterval paces successive model tests so the
	// backfill task does not hammer upstream providers when many models lack
	// data.
	perfMetricsBackfillRequestInterval = 2 * time.Second

	perfMetricsBackfillDefaultIntervalMinutes = 360
	perfMetricsBackfillMinIntervalMinutes     = 60
)

type perfMetricsBackfillSummary struct {
	TotalModels     int `json:"total_models"`
	TestedModels    int `json:"tested_models"`
	SucceededModels int `json:"succeeded_models"`
	FailedModels    int `json:"failed_models"`
	SkippedModels   int `json:"skipped_models"`
}

// perfMetricsBackfillHandler runs the scheduled "auto-test models with no perf
// data" job. It enumerates models that are currently offered (present in
// pricing) but have no perf_metric buckets in the lookback window, then runs a
// channel test against each one so the model details page shows latency /
// success data even before real traffic arrives. The retry mechanism tries up
// to perfMetricsBackfillMaxRetries candidate channels per model.
type perfMetricsBackfillHandler struct{}

func (perfMetricsBackfillHandler) Type() string { return model.SystemTaskTypePerfMetricsBackfill }

func (perfMetricsBackfillHandler) Enabled() bool {
	return common.GetEnvOrDefaultBool("PERF_METRICS_BACKFILL_TASK_ENABLED", true)
}

func (perfMetricsBackfillHandler) Interval() time.Duration {
	intervalMinutes := common.GetEnvOrDefault("PERF_METRICS_BACKFILL_TASK_INTERVAL_MINUTES", perfMetricsBackfillDefaultIntervalMinutes)
	if intervalMinutes < perfMetricsBackfillMinIntervalMinutes {
		intervalMinutes = perfMetricsBackfillDefaultIntervalMinutes
	}
	return time.Duration(intervalMinutes) * time.Minute
}

func (perfMetricsBackfillHandler) NewPayload() any { return nil }

func (perfMetricsBackfillHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	summary, err := runPerfMetricsBackfillTask(ctx, service.NewSystemTaskProgressReporter(task, runnerID))
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

// runPerfMetricsBackfillTask is the core implementation. It is split out so the
// handler is a thin wrapper and the logic is testable in isolation.
func runPerfMetricsBackfillTask(ctx context.Context, report func(processed, total int)) (perfMetricsBackfillSummary, error) {
	summary := perfMetricsBackfillSummary{}

	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return summary, err
	}

	// Models currently offered come from pricing meta — this is the same source
	// the model square / rankings use, so we stay consistent with what users
	// see.
	pricings := model.GetPricing()
	if len(pricings) == 0 {
		return summary, nil
	}

	// Models with at least one perf_metric bucket in the lookback window already
	// have data; skip them.
	sinceTs := model.PerfMetricStartTime(perfMetricsBackfillLookbackHours)
	modelsWithData, err := model.GetPerfMetricModelNamesSince(sinceTs)
	if err != nil {
		return summary, err
	}
	modelsWithDataSet := make(map[string]struct{}, len(modelsWithData))
	for _, name := range modelsWithData {
		modelsWithDataSet[name] = struct{}{}
	}

	// Build a map: model name -> ordered list of enabled channels that serve it.
	// We use GetAllChannels(selectAll=true) so the key material is available for
	// the relay pipeline; channel tests need to actually call upstream.
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		return summary, err
	}
	modelToChannels := make(map[string][]*model.Channel)
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			modelToChannels[modelName] = append(modelToChannels[modelName], channel)
		}
	}

	// Build the work list: models that are offered, have a serving channel, and
	// have no perf data in the lookback window.
	modelsToTest := make([]string, 0, len(pricings))
	for _, p := range pricings {
		if _, hasData := modelsWithDataSet[p.ModelName]; hasData {
			summary.SkippedModels++
			continue
		}
		if _, hasChannel := modelToChannels[p.ModelName]; !hasChannel {
			summary.SkippedModels++
			continue
		}
		modelsToTest = append(modelsToTest, p.ModelName)
	}
	summary.TotalModels = len(modelsToTest)

	for index, modelName := range modelsToTest {
		if ctx != nil && ctx.Err() != nil {
			break
		}
		if report != nil {
			report(index, summary.TotalModels)
		}

		if success := testModelWithRetry(ctx, modelName, modelToChannels[modelName], testUserID); success {
			summary.SucceededModels++
		} else {
			summary.FailedModels++
		}
		summary.TestedModels++

		// Pace successive model tests so the backfill does not hammer upstreams.
		if ctx != nil && ctx.Err() == nil && index+1 < len(modelsToTest) {
			select {
			case <-ctx.Done():
			case <-time.After(perfMetricsBackfillRequestInterval):
			}
		}
	}

	if report != nil && (ctx == nil || ctx.Err() == nil) {
		report(summary.TotalModels, summary.TotalModels)
	}
	return summary, nil
}

// testModelWithRetry tries up to perfMetricsBackfillMaxRetries candidate
// channels for the same model. A success on any channel short-circuits the
// remaining retries. The backoff between retries is short because we already
// switch to a different channel (different upstream) on each attempt.
func testModelWithRetry(ctx context.Context, modelName string, candidateChannels []*model.Channel, testUserID int) bool {
	maxAttempts := perfMetricsBackfillMaxRetries
	if maxAttempts > len(candidateChannels) {
		maxAttempts = len(candidateChannels)
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx != nil && ctx.Err() != nil {
			return false
		}
		channel := candidateChannels[attempt]
		result := testChannel(ctx, channel, testUserID, modelName, "", shouldUseStreamForAutomaticChannelTest(channel), nil)
		if result.localErr == nil && result.newAPIError == nil {
			return true
		}
		common.SysLog(fmt.Sprintf(
			"perf_metrics_backfill: model=%s attempt=%d/%d channel_id=%d failed: localErr=%v newAPIError=%v",
			modelName, attempt+1, maxAttempts, channel.Id, result.localErr, result.newAPIError,
		))
		// Backoff before the next candidate channel, unless this was the last attempt.
		if attempt+1 < maxAttempts {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(perfMetricsBackfillRetryDelay):
			}
		}
	}
	return false
}
