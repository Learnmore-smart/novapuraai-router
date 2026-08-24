package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasSuccessfulRequestForUserIsScopedToTheRequestedUser(t *testing.T) {
	truncateTables(t)

	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 101,
		Type:   LogTypeConsume,
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 202,
		Type:   LogTypeError,
	}).Error)

	// User 303 has no successful request of their own. This is the negative
	// control that makes the cross-user isolation contract explicit.
	hasSuccess, err := HasSuccessfulRequestForUser(303)
	require.NoError(t, err)
	assert.False(t, hasSuccess)

	hasSuccess, err = HasSuccessfulRequestForUser(101)
	require.NoError(t, err)
	assert.True(t, hasSuccess)

	hasSuccess, err = HasSuccessfulRequestForUser(202)
	require.NoError(t, err)
	assert.False(t, hasSuccess)

	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 202,
		Type:   LogTypeConsume,
	}).Error)
	hasSuccess, err = HasSuccessfulRequestForUser(101)
	require.NoError(t, err)
	assert.True(t, hasSuccess)
	hasSuccess, err = HasSuccessfulRequestForUser(303)
	require.NoError(t, err)
	assert.False(t, hasSuccess)
}
