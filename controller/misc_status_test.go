package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatusExposesAuthoritativeRegisterPromo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousEnabled := common.RegisterPromoEnabled
	previousAmount := common.RegisterPromoCNYYuan
	t.Cleanup(func() {
		common.RegisterPromoEnabled = previousEnabled
		common.RegisterPromoCNYYuan = previousAmount
	})
	common.RegisterPromoEnabled = true
	common.RegisterPromoCNYYuan = 10

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	GetStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, true, response.Data["register_promo_enabled"])
	assert.Equal(t, float64(10), response.Data["register_promo_amount"])
	assert.Equal(t, "CNY", response.Data["register_promo_currency"])
	assert.NotEqual(t, float64(50), response.Data["register_promo_amount"])
}
