/*
Copyright (C) 2023-2026 QuantumNous / NovaPuraAI fork tests

Tests exercise the shipped in-memory 号池 selection path in channel_cache.go.
*/
package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func uintPtr(v uint) *uint    { return &v }
func int64Ptr(v int64) *int64 { return &v }

// installMemoryPool installs a minimal in-memory 号池 fixture used by
// GetRandomSatisfiedChannel when MemoryCacheEnabled is true.
// Mirrors InitChannelCache: only Status==Enabled channels appear in group2model2channels.
func installMemoryPool(t *testing.T, channels []*Channel) {
	t.Helper()

	prevMem := common.MemoryCacheEnabled
	prevIDM := channelsIDM
	prevG2M := group2model2channels
	prevAdv := channel2advancedCustomConfig
	t.Cleanup(func() {
		common.MemoryCacheEnabled = prevMem
		channelsIDM = prevIDM
		group2model2channels = prevG2M
		channel2advancedCustomConfig = prevAdv
	})

	common.MemoryCacheEnabled = true
	channelsIDM = make(map[int]*Channel, len(channels))
	group2model2channels = make(map[string]map[string][]int)
	channel2advancedCustomConfig = nil

	for _, ch := range channels {
		c := ch
		channelsIDM[c.Id] = c
		if c.Status != common.ChannelStatusEnabled {
			continue
		}
		groups := splitCSV(c.Group)
		models := splitCSV(c.Models)
		for _, g := range groups {
			if group2model2channels[g] == nil {
				group2model2channels[g] = make(map[string][]int)
			}
			for _, m := range models {
				group2model2channels[g][m] = append(group2model2channels[g][m], c.Id)
			}
		}
	}

	// Sort each group/model slice by priority DESC (same as InitChannelCache).
	for g, model2ch := range group2model2channels {
		for m, ids := range model2ch {
			sortChannelIDsByPriority(ids)
			group2model2channels[g][m] = ids
		}
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}

func sortChannelIDsByPriority(ids []int) {
	// simple insertion by GetPriority DESC using channelsIDM
	for i := 1; i < len(ids); i++ {
		j := i
		for j > 0 {
			pi := channelsIDM[ids[j-1]].GetPriority()
			pj := channelsIDM[ids[j]].GetPriority()
			if pi >= pj {
				break
			}
			ids[j-1], ids[j] = ids[j], ids[j-1]
			j--
		}
	}
}

func TestGetRandomSatisfiedChannel_EnabledOnly(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 1, Name: "live-a", Status: common.ChannelStatusEnabled,
			Group: "default", Models: "gpt-test",
			Priority: int64Ptr(10), Weight: uintPtr(100),
		},
		{
			Id: 2, Name: "dead-b", Status: common.ChannelStatusAutoDisabled,
			Group: "default", Models: "gpt-test",
			Priority: int64Ptr(10), Weight: uintPtr(100),
		},
		{
			Id: 3, Name: "manual-off", Status: common.ChannelStatusManuallyDisabled,
			Group: "default", Models: "gpt-test",
			Priority: int64Ptr(99), Weight: uintPtr(100),
		},
	})

	// Disabled channels live in channelsIDM but must not be selectable.
	require.Contains(t, channelsIDM, 2)
	require.Contains(t, channelsIDM, 3)

	for i := 0; i < 20; i++ {
		ch, err := GetRandomSatisfiedChannel("default", "gpt-test", 0, "")
		require.NoError(t, err)
		require.NotNil(t, ch)
		require.Equal(t, 1, ch.Id, "only enabled channel #1 may be selected")
	}
}

func TestGetRandomSatisfiedChannel_PriorityRetryStepsDown(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 10, Name: "p-high", Status: common.ChannelStatusEnabled,
			Group: "pool", Models: "m1",
			Priority: int64Ptr(20), Weight: uintPtr(50),
		},
		{
			Id: 11, Name: "p-mid", Status: common.ChannelStatusEnabled,
			Group: "pool", Models: "m1",
			Priority: int64Ptr(10), Weight: uintPtr(50),
		},
		{
			Id: 12, Name: "p-low", Status: common.ChannelStatusEnabled,
			Group: "pool", Models: "m1",
			Priority: int64Ptr(0), Weight: uintPtr(50),
		},
	})

	// retry=0 → highest priority tier (20)
	ch0, err := GetRandomSatisfiedChannel("pool", "m1", 0, "")
	require.NoError(t, err)
	require.NotNil(t, ch0)
	require.Equal(t, 10, ch0.Id)
	require.EqualValues(t, 20, ch0.GetPriority())

	// retry=1 → next tier (10)
	ch1, err := GetRandomSatisfiedChannel("pool", "m1", 1, "")
	require.NoError(t, err)
	require.NotNil(t, ch1)
	require.Equal(t, 11, ch1.Id)
	require.EqualValues(t, 10, ch1.GetPriority())

	// retry=2 → lowest tier (0)
	ch2, err := GetRandomSatisfiedChannel("pool", "m1", 2, "")
	require.NoError(t, err)
	require.NotNil(t, ch2)
	require.Equal(t, 12, ch2.Id)
	require.EqualValues(t, 0, ch2.GetPriority())

	// retry beyond tiers clamps to last tier
	ch3, err := GetRandomSatisfiedChannel("pool", "m1", 99, "")
	require.NoError(t, err)
	require.NotNil(t, ch3)
	require.Equal(t, 12, ch3.Id)
}

func TestGetRandomSatisfiedChannel_WeightBiasWithinPriority(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 21, Name: "heavy", Status: common.ChannelStatusEnabled,
			Group: "w", Models: "m",
			Priority: int64Ptr(5), Weight: uintPtr(90),
		},
		{
			Id: 22, Name: "light", Status: common.ChannelStatusEnabled,
			Group: "w", Models: "m",
			Priority: int64Ptr(5), Weight: uintPtr(10),
		},
	})

	const n = 400
	counts := map[int]int{}
	for i := 0; i < n; i++ {
		ch, err := GetRandomSatisfiedChannel("w", "m", 0, "")
		require.NoError(t, err)
		require.NotNil(t, ch)
		counts[ch.Id]++
	}

	require.Greater(t, counts[21], 0, "heavy channel should be selected")
	require.Greater(t, counts[22], 0, "light channel should occasionally be selected")
	// 90:10 weight with smoothing — heavy must dominate clearly
	require.Greater(t, counts[21], counts[22]*2,
		"heavy (weight 90) should be selected much more than light (weight 10): %#v", counts)
}

func TestCacheUpdateChannelStatus_RemovesDisabledFromPool(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 31, Name: "a", Status: common.ChannelStatusEnabled,
			Group: "g", Models: "m",
			Priority: int64Ptr(1), Weight: uintPtr(10),
		},
		{
			Id: 32, Name: "b", Status: common.ChannelStatusEnabled,
			Group: "g", Models: "m",
			Priority: int64Ptr(1), Weight: uintPtr(10),
		},
	})

	// Disable channel 31 via shipped cache status update path
	CacheUpdateChannelStatus(31, common.ChannelStatusAutoDisabled)

	for i := 0; i < 30; i++ {
		ch, err := GetRandomSatisfiedChannel("g", "m", 0, "")
		require.NoError(t, err)
		require.NotNil(t, ch)
		require.Equal(t, 32, ch.Id, "disabled #31 must leave the pool")
	}

	// channelsIDM still tracks the channel object with updated status
	require.Equal(t, common.ChannelStatusAutoDisabled, channelsIDM[31].Status)
}

func TestGetRandomSatisfiedChannel_NoChannelReturnsNil(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 40, Name: "other-model", Status: common.ChannelStatusEnabled,
			Group: "g", Models: "other",
			Priority: int64Ptr(1), Weight: uintPtr(10),
		},
	})

	ch, err := GetRandomSatisfiedChannel("g", "missing-model", 0, "")
	require.NoError(t, err)
	require.Nil(t, ch)
}

func TestGetRandomSatisfiedChannel_SkipsLiveCooldown(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 51, Name: "cooling", Status: common.ChannelStatusEnabled,
			Group: "g", Models: "m",
			Priority: int64Ptr(10), Weight: uintPtr(100),
			CooldownUntil: time.Now().Unix() + 300,
		},
		{
			Id: 52, Name: "healthy", Status: common.ChannelStatusEnabled,
			Group: "g", Models: "m",
			Priority: int64Ptr(10), Weight: uintPtr(100),
		},
	})

	for i := 0; i < 20; i++ {
		channel, err := GetRandomSatisfiedChannel("g", "m", 0, "")
		require.NoError(t, err)
		require.NotNil(t, channel)
		require.Equal(t, 52, channel.Id)
	}
}

func TestCacheChannelCooldownLifecycleAffectsSelectionImmediately(t *testing.T) {
	installMemoryPool(t, []*Channel{
		{
			Id: 61, Name: "only", Status: common.ChannelStatusEnabled,
			Group: "g", Models: "m",
			Priority: int64Ptr(10), Weight: uintPtr(100),
		},
	})

	CacheSetChannelCooldown(61, time.Now().Unix()+300)
	require.Positive(t, channelsIDM[61].FailureCount)
	channel, err := GetRandomSatisfiedChannel("g", "m", 0, "")
	require.NoError(t, err)
	require.Nil(t, channel)

	CacheClearChannelCooldown(61)
	require.Zero(t, channelsIDM[61].FailureCount)
	require.Zero(t, channelsIDM[61].CooldownUntil)
	channel, err = GetRandomSatisfiedChannel("g", "m", 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 61, channel.Id)
}
