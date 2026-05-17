//go:build integration

package integration

import (
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_SchedulerCRUD(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)
	ctx := testutil.Context(t)

	name := testutil.UniqueName(t, "sched")
	disabled := true
	id, err := sys.SchedulerAdd(ctx, system.SchedulerAddArgs{
		Name:     name,
		OnEvent:  ":log info \"" + name + "\"",
		Interval: "1d",
		Disabled: &disabled,
		Comment:  "Monitor Profile mikhmon-it",
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)
	t.Cleanup(func() { _ = sys.SchedulerRemove(ctx, id) })

	got, err := sys.SchedulerByName(ctx, name)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0].ID)
	assert.True(t, got[0].Disabled)

	// Update interval
	require.NoError(t, sys.SchedulerSet(ctx, system.SchedulerSetArgs{
		ID:       id,
		Interval: "00:02:00",
	}))

	// Count > 0
	n, err := sys.SchedulerCount(ctx)
	require.NoError(t, err)
	assert.Greater(t, n, 0)
}
