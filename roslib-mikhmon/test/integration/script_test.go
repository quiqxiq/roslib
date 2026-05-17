//go:build integration

package integration

import (
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ScriptCRUD(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)
	ctx := testutil.Context(t)

	name := testutil.UniqueName(t, "script")
	id, err := sys.ScriptAdd(ctx, system.ScriptAddArgs{
		Name:    name,
		Source:  ":log info \"" + name + "\"",
		Comment: system.CommentTransaction,
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)
	t.Cleanup(func() { _ = sys.ScriptRemove(ctx, id) })

	// Verify findable by name
	scripts, err := sys.ScriptByName(ctx, name)
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, id, scripts[0].ID)

	// Verify findable by comment "mikhmon"
	all, err := sys.ScriptByComment(ctx, system.CommentTransaction)
	require.NoError(t, err)
	found := false
	for _, s := range all {
		if s.ID == id {
			found = true
			break
		}
	}
	assert.True(t, found, "script tidak ditemukan dalam filter comment=mikhmon")

	// Update source
	require.NoError(t, sys.ScriptSet(ctx, system.ScriptSetArgs{
		ID:     id,
		Source: ":log info \"updated\"",
	}))

	// Remove
	require.NoError(t, sys.ScriptRemove(ctx, id))
	gone, err := sys.ScriptByID(ctx, id)
	_ = gone
	assert.Error(t, err, "script harus tidak ditemukan setelah remove")
}
