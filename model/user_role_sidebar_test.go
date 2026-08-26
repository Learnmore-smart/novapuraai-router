package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToBaseUserIncludesRole(t *testing.T) {
	user := User{
		Id:       9,
		Username: "root-cache",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Email:    "root@example.com",
	}
	cache := user.ToBaseUser()
	require.NotNil(t, cache)
	assert.Equal(t, common.RoleRootUser, cache.Role)
	assert.Equal(t, user.Id, cache.Id)
	assert.Equal(t, user.Username, cache.Username)
	assert.Equal(t, user.Status, cache.Status)
}

func TestApplyDefaultSidebarModulesEnablesSettingsForRoot(t *testing.T) {
	admin := User{Role: common.RoleAdminUser}
	admin.ApplyDefaultSidebarModules()
	assert.False(t, sidebarSettingEnabled(t, &admin))

	root := User{Role: common.RoleRootUser}
	root.ApplyDefaultSidebarModules()
	assert.True(t, sidebarSettingEnabled(t, &root))
}

func sidebarSettingEnabled(t *testing.T, user *User) bool {
	t.Helper()
	var modules map[string]map[string]any
	require.NoError(t, common.Unmarshal([]byte(user.GetSetting().SidebarModules), &modules))
	admin := modules["admin"]
	require.NotNil(t, admin)
	setting, ok := admin["setting"].(bool)
	require.True(t, ok)
	return setting
}
