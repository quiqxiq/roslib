//go:build integration

package integration

import (
	"errors"
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_HotspotUserCRUD(t *testing.T) {
	c := testutil.NewClient(t)
	hot := hotspot.New(c)
	ctx := testutil.Context(t)

	name := testutil.UniqueName(t, "user")
	id, err := hot.UserAdd(ctx, hotspot.UserAddArgs{
		Name:     name,
		Password: "p4ssw0rd",
		Comment:  "mikhmon-it",
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)
	t.Cleanup(func() { _ = hot.UserRemove(ctx, id) })

	got, err := hot.UserByName(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "mikhmon-it", got.Comment)

	// Toggle disabled
	require.NoError(t, hot.UserSetDisabled(ctx, id, true))
	got, err = hot.UserByID(ctx, id)
	require.NoError(t, err)
	assert.True(t, got.Disabled)

	// Set expiry comment (analisis §1.6 inline)
	require.NoError(t, hot.UserSetExpiry(ctx, id, "jan/06/2099 14:32:01"))
	got, err = hot.UserByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "jan/06/2099 14:32:01", got.Comment)

	// Remove
	require.NoError(t, hot.UserRemove(ctx, id))
	_, err = hot.UserByID(ctx, id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, mikrotik.ErrNotFound))
}

func TestIntegration_HotspotUserCount(t *testing.T) {
	c := testutil.NewClient(t)
	hot := hotspot.New(c)
	n, err := hot.UserCount(testutil.Context(t))
	require.NoError(t, err)
	t.Logf("total hotspot users: %d", n)
}
