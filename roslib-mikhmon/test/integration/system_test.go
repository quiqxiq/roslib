//go:build integration

package integration

import (
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Identity(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)

	id, err := sys.Identity(testutil.Context(t))
	require.NoError(t, err)
	assert.NotEmpty(t, id.Name)
	t.Logf("router identity = %q", id.Name)
}

func TestIntegration_Resource(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)

	res, err := sys.Resource(testutil.Context(t))
	require.NoError(t, err)
	assert.NotEmpty(t, res.Version)
	assert.NotEmpty(t, res.BoardName)
	t.Logf("router %s, version %s, uptime %s", res.BoardName, res.Version, res.Uptime)
}

func TestIntegration_Clock(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)

	clk, err := sys.Clock(testutil.Context(t))
	require.NoError(t, err)
	assert.NotEmpty(t, clk.Date)
	assert.NotEmpty(t, clk.Time)
	t.Logf("router clock: %s %s (%s)", clk.Date, clk.Time, clk.TimeZoneName)
}

func TestIntegration_Routerboard(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)

	rb, err := sys.Routerboard(testutil.Context(t))
	require.NoError(t, err)
	t.Logf("routerboard: model=%s firmware=%s/%s", rb.Model, rb.CurrentFirmware, rb.UpgradeFirmware)
}

func TestIntegration_LoggingByPrefix(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)

	// Pakai prefix yang TIDAK akan exist supaya tidak nge-track existing.
	prefix := testutil.UniqueName(t, "logprefix")
	n, err := sys.LoggingByPrefix(testutil.Context(t), prefix)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "prefix unik tidak boleh return row")
}
