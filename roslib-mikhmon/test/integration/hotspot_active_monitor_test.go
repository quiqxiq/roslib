//go:build integration

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/stretchr/testify/require"
)

// TestIntegration_HotspotMonitorActiveCount menguji bahwa poll
// `/ip/hotspot/active/print count-only` panggil handler ≥2 kali dalam window.
func TestIntegration_HotspotMonitorActiveCount(t *testing.T) {
	c := testutil.NewClient(t)
	hot := hotspot.New(c)

	const id = "it-monitor-active-count"
	var ticks atomic.Int32
	err := hot.MonitorActiveCount(id, 1*time.Second, func(n int) {
		ticks.Add(1)
		t.Logf("active count tick #%d: %d", ticks.Load(), n)
	})
	require.NoError(t, err)

	time.Sleep(3500 * time.Millisecond) // ≥3 tick (1s interval)
	require.True(t, hot.StopMonitor(id))
	require.GreaterOrEqual(t, ticks.Load(), int32(2), "minimal 2 tick dalam 3.5s")
}
