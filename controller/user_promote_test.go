package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManageUserPromoteRoot(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(db))

	rootUser := &model.User{
		Username: "super-root",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "AFF_ROOT",
	}
	require.NoError(t, db.Create(rootUser).Error)

	targetUser := &model.User{
		Username: "grok-bot",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "AFF_GROK",
	}
	require.NoError(t, db.Create(targetUser).Error)

	// 1. Root promotes common user to Root
	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/user/manage", ManageRequest{
		Id:     targetUser.Id,
		Action: "promote_root",
	}, rootUser.Id)
	ctx.Set("role", common.RoleRootUser)

	ManageUser(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	updated, err := model.GetUserById(targetUser.Id, false)
	require.NoError(t, err)
	assert.Equal(t, common.RoleRootUser, updated.Role)

	// 2. Promoting an already Root user fails
	ctx2, recorder2 := newAuthenticatedContext(t, http.MethodPost, "/api/user/manage", ManageRequest{
		Id:     targetUser.Id,
		Action: "promote_root",
	}, rootUser.Id)
	ctx2.Set("role", common.RoleRootUser)

	ManageUser(ctx2)
	require.Equal(t, http.StatusOK, recorder2.Code)
	response2 := decodeAPIResponse(t, recorder2)
	assert.False(t, response2.Success)

	// 3. Admin cannot promote to Root
	adminUser := &model.User{
		Username: "regular-admin",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "AFF_ADMIN",
	}
	require.NoError(t, db.Create(adminUser).Error)

	commonUser := &model.User{
		Username: "another-user",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "AFF_COMMON",
	}
	require.NoError(t, db.Create(commonUser).Error)

	ctx3, recorder3 := newAuthenticatedContext(t, http.MethodPost, "/api/user/manage", ManageRequest{
		Id:     commonUser.Id,
		Action: "promote_root",
	}, adminUser.Id)
	ctx3.Set("role", common.RoleAdminUser)

	ManageUser(ctx3)
	require.Equal(t, http.StatusOK, recorder3.Code)
	response3 := decodeAPIResponse(t, recorder3)
	assert.False(t, response3.Success)

	// 4. Root can demote another Root to Admin
	ctx4, recorder4 := newAuthenticatedContext(t, http.MethodPost, "/api/user/manage", ManageRequest{
		Id:     targetUser.Id,
		Action: "demote",
	}, rootUser.Id)
	ctx4.Set("role", common.RoleRootUser)

	ManageUser(ctx4)
	require.Equal(t, http.StatusOK, recorder4.Code)
	response4 := decodeAPIResponse(t, recorder4)
	require.True(t, response4.Success, response4.Message)

	demoted, err := model.GetUserById(targetUser.Id, false)
	require.NoError(t, err)
	assert.Equal(t, common.RoleAdminUser, demoted.Role)

	// 5. Root cannot demote itself
	ctx5, recorder5 := newAuthenticatedContext(t, http.MethodPost, "/api/user/manage", ManageRequest{
		Id:     rootUser.Id,
		Action: "demote",
	}, rootUser.Id)
	ctx5.Set("role", common.RoleRootUser)

	ManageUser(ctx5)
	require.Equal(t, http.StatusOK, recorder5.Code)
	response5 := decodeAPIResponse(t, recorder5)
	assert.False(t, response5.Success)
}
