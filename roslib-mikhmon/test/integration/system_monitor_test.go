//go:build integration

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
	"github.com/stretchr/testify/require"
)

// TestIntegration_SystemMonitorResource menguji poll `/system/resource/print`
// emit !re tiap tick. Verifikasi ada minimal 1 sentence dengan field
// uptime + cpu-load.
func TestIntegration_SystemMonitorResource(t *testing.T) {
	c := testutil.NewClient(t)
	sys := system.New(c)

	const id = "it-monitor-resource"
	var ticks atomic.Int32
	var lastUptime string
	err := sys.MonitorResource(id, 1*time.Second, func(s *roslib.Sentence) {
		if s.Word() != "!re" {
			return
		}
		ticks.Add(1)
		lastUptime = s.Get("uptime")
		t.Logf("resource tick #%d: uptime=%s cpu=%s", ticks.Load(), lastUptime, s.Get("cpu-load"))
	})
	require.NoError(t, err)

	time.Sleep(3500 * time.Millisecond)
	require.True(t, sys.StopMonitor(id))
	require.GreaterOrEqual(t, ticks.Load(), int32(2), "minimal 2 tick dalam 3.5s")
	require.NotEmpty(t, lastUptime, "uptime field harus terisi")
}
