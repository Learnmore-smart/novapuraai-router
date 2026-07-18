package router

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/default
var defaultOnlyBuildFS embed.FS

//go:embed testdata/default/index.html
var defaultOnlyIndexPage []byte

func TestSetWebRouterServesDefaultFrontendWithoutClassicAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	require.NotPanics(t, func() {
		SetWebRouter(engine, ThemeAssets{
			DefaultBuildFS:   defaultOnlyBuildFS,
			DefaultIndexPage: defaultOnlyIndexPage,
		})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	engine.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "NovaPura default fixture")
}
