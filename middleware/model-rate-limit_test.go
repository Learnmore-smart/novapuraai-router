package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service/deepseekfairuse"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFairUseRateLimitContext(body, path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", path, nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(common.KeyRequestBody, []byte(body))
	common.SetContextKey(c, constant.ContextKeyTokenUnlimited, true)
	common.SetContextKey(c, constant.ContextKeyTokenGroup, deepseekfairuse.DeepSeekV4FlashDedicatedGroup)
	common.SetContextKey(c, constant.ContextKeyUserGroup, deepseekfairuse.DeepSeekV4FlashDedicatedGroup)
	return c
}

func newModelRateLimitRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	connectionString := strings.TrimSpace(os.Getenv("DEEPSEEK_FUP_REDIS_CONN_STRING"))
	address := strings.TrimSpace(os.Getenv("DEEPSEEK_FUP_REDIS_ADDR"))
	if connectionString == "" && address == "" {
		connectionString = strings.TrimSpace(os.Getenv("REDIS_CONN_STRING"))
	}
	var options *redis.Options
	if connectionString != "" {
		var err error
		options, err = redis.ParseURL(connectionString)
		if err != nil {
			t.Skip("configured Redis connection string is unavailable for the real total-rate regression")
		}
	} else if address != "" {
		options = &redis.Options{Addr: address, Password: os.Getenv("DEEPSEEK_FUP_REDIS_PASSWORD")}
	} else {
		t.Skip("set a FUP Redis connection environment variable to run the real total-rate regression")
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		_ = client.Close()
		t.Skip("configured Redis endpoint is unavailable for the real total-rate regression")
	}
	cancel()
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestShouldBypassDeepSeekFairUseRequestRateLimitRequiresRecurringEligibility(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		path  string
		group string
		free  bool
		want  bool
	}{
		{
			name:  "exact Claude model",
			body:  `{"model":"deepseek-v4-flash-0731"}`,
			path:  "/v1/messages",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  false,
			want:  true,
		},
		{
			name:  "exact chat model",
			body:  `{"model":"deepseek-v4-flash-0731"}`,
			path:  "/v1/chat/completions",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  true,
			want:  true,
		},
		{
			name:  "exact compact responses model",
			body:  `{"model":"deepseek-v4-flash-0731"}`,
			path:  "/v1/responses/compact",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  true,
			want:  true,
		},
		{
			name:  "different platform model",
			body:  `{"model":"gpt-5"}`,
			path:  "/v1/chat/completions",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  true,
			want:  true,
		},
		{
			name:  "missing model",
			body:  `{"model":" "}`,
			path:  "/v1/chat/completions",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  true,
			want:  false,
		},
		{
			name:  "ordinary group",
			body:  `{"model":"deepseek-v4-flash-0731"}`,
			path:  "/v1/chat/completions",
			group: "default",
			free:  true,
			want:  false,
		},
		{
			name:  "finite token under subscribed account",
			body:  `{"model":"deepseek-v4-flash-0731"}`,
			path:  "/v1/chat/completions",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  false,
			want:  true,
		},
		{
			name:  "non target route",
			body:  `{"model":"deepseek-v4-flash-0731"}`,
			path:  "/v1/models",
			group: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:  true,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFairUseRateLimitContext(tt.body, tt.path)
			common.SetContextKey(c, constant.ContextKeyTokenUnlimited, tt.free)
			common.SetContextKey(c, constant.ContextKeyUserGroup, tt.group)
			assert.Equal(t, tt.want, shouldBypassDeepSeekFairUseRequestRateLimit(c))
			common.CleanupBodyStorage(c)
		})
	}
}

func TestShouldBypassDeepSeekFairUseRequestRateLimitRestoresBody(t *testing.T) {
	c := newFairUseRateLimitContext(`{"model":"deepseek-v4-flash-0731"}`, "/v1/responses")
	require.True(t, shouldBypassDeepSeekFairUseRequestRateLimit(c))
	storage, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	body, err := storage.Bytes()
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"deepseek-v4-flash-0731"}`, string(body))
	common.CleanupBodyStorage(c)
}

func TestRedisRateLimitStopsAfterTotalRejection(t *testing.T) {
	client := newModelRateLimitRedisClient(t)
	key := "rateLimit:991991"
	ctx := context.Background()
	require.NoError(t, client.Del(ctx, key).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })

	originalRedis := common.RDB
	t.Cleanup(func() { common.RDB = originalRedis })
	common.RDB = client

	gin.SetMode(gin.TestMode)
	router := gin.New()
	downstreamCalls := 0
	router.Use(func(c *gin.Context) {
		c.Set("id", 991991)
		c.Next()
	})
	router.Use(redisRateLimitHandler(60, 1, 0))
	router.POST("/", func(c *gin.Context) {
		downstreamCalls++
		c.Status(http.StatusNoContent)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest("POST", "/", nil))
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest("POST", "/", nil))

	assert.Equal(t, http.StatusNoContent, first.Code)
	assert.Equal(t, http.StatusTooManyRequests, second.Code)
	assert.Equal(t, 1, downstreamCalls)
}
