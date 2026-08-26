package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAuthRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	previousDB := model.DB
	previousLogDB := model.LOG_DB

	dsn := fmt.Sprintf("file:auth-role-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		model.DB = previousDB
		model.LOG_DB = previousLogDB
	})
	return db
}

func performAuthRoleRequest(t *testing.T, db *gorm.DB, sessionRole int, dbRole int, dbStatus int, minRole int) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{
		Username: "auth-role-user",
		Role:     dbRole,
		Status:   dbStatus,
		Group:    "default",
		AffCode:  "AFF_AUTH",
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("auth-role-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", user.Username)
		session.Set("role", sessionRole)
		session.Set("id", user.Id)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/test", authHelperBind(minRole), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"role":    c.GetInt("role"),
		})
	})

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	router.ServeHTTP(loginRecorder, loginRequest)
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	request.Header.Set("New-Api-User", fmt.Sprintf("%d", user.Id))
	for _, cookie := range loginRecorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func authHelperBind(minRole int) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHelper(c, minRole)
	}
}

func TestAuthHelperUsesDatabaseRoleNotSessionRole(t *testing.T) {
	db := setupAuthRoleTestDB(t)

	recorder := performAuthRoleRequest(t, db, common.RoleAdminUser, common.RoleRootUser, common.UserStatusEnabled, common.RoleRootUser)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, true, body["success"])
	assert.Equal(t, float64(common.RoleRootUser), body["role"])
}

func TestAuthHelperRejectsStaleRootSessionAfterDemote(t *testing.T) {
	db := setupAuthRoleTestDB(t)

	recorder := performAuthRoleRequest(t, db, common.RoleRootUser, common.RoleAdminUser, common.UserStatusEnabled, common.RoleRootUser)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, false, body["success"])
}

func TestAuthHelperRejectsDisabledUserDespiteEnabledSession(t *testing.T) {
	db := setupAuthRoleTestDB(t)

	recorder := performAuthRoleRequest(t, db, common.RoleAdminUser, common.RoleAdminUser, common.UserStatusDisabled, common.RoleAdminUser)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, false, body["success"])
}
