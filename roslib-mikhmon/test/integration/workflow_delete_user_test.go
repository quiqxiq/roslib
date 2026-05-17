//go:build integration

package integration

import (
	"errors"
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
	"github.com/quiqxiq/roslib-mikhmon/workflows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_DeleteUser_Cascade(t *testing.T) {
	c := testutil.NewClient(t)
	hot := hotspot.New(c)
	sys := system.New(c)
	ctx := testutil.Context(t)

	name := testutil.UniqueName(t, "wfu")

	// Setup: buat user + script tx + scheduler dengan nama yang sama
	uid, err := hot.UserAdd(ctx, hotspot.UserAddArgs{Name: name, Password: "p"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = hot.UserRemove(ctx, uid) }) // safety

	scrID, err := sys.ScriptAdd(ctx, system.ScriptAddArgs{
		Name:    name,
		Source:  ":log info ok",
		Comment: system.CommentTransaction,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sys.ScriptRemove(ctx, scrID) })

	disabled := true
	schID, err := sys.SchedulerAdd(ctx, system.SchedulerAddArgs{
		Name:     name,
		OnEvent:  ":log info ok",
		Interval: "1d",
		Disabled: &disabled,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sys.SchedulerRemove(ctx, schID) })

	// Action
	wf := workflows.New(c)
	require.NoError(t, workflows.DeleteUser(ctx, wf, uid))

	// Verify all gone
	_, err = hot.UserByID(ctx, uid)
	assert.True(t, errors.Is(err, mikrotik.ErrNotFound))

	scripts, err := sys.ScriptByName(ctx, name)
	require.NoError(t, err)
	assert.Empty(t, scripts)

	schedulers, err := sys.SchedulerByName(ctx, name)
	require.NoError(t, err)
	assert.Empty(t, schedulers)
}
