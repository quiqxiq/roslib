//go:build integration

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
	"github.com/stretchr/testify/require"
)

// pickInterface mencari satu interface running untuk dipakai test traffic
// stream. Kalau tidak ada (router tanpa physical port aktif), test di-skip.
func pickInterface(t *testing.T, c *roslib.Device) string {
	t.Helper()
	net := network.New(c)
	ifs, err := net.InterfaceList(testutil.Context(t))
	require.NoError(t, err)
	for _, i := range ifs {
		if i.Running && !i.Disabled {
			return i.Name
		}
	}
	t.Skip("tidak ada interface running aktif — skip traffic stream test")
	return ""
}

// TestIntegration_InterfaceTrafficStream menguji inherent streaming
// `/interface/monitor-traffic interface=<name>`. Verifikasi terima ≥1
// sentence dalam 3 detik (router emit 1Hz typical).
func TestIntegration_InterfaceTrafficStream(t *testing.T) {
	c := testutil.NewClient(t)
	iface := pickInterface(t, c)
	net := network.New(c)

	const id = "it-monitor-traffic"
	var ticks atomic.Int32
	err := net.InterfaceTrafficStream(id, iface, func(s *roslib.Sentence) {
		ticks.Add(1)
		if ticks.Load() <= 3 {
			t.Logf("traffic #%d iface=%s rx=%s tx=%s",
				ticks.Load(), s.Get("name"),
				s.Get("rx-bits-per-second"), s.Get("tx-bits-per-second"))
		}
	})
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	require.True(t, net.StopStream(id))
	require.GreaterOrEqual(t, ticks.Load(), int32(1), "minimal 1 sample dalam 3s")
}

// TestIntegration_InterfaceStatsStream menguji `/interface/print stats
// interval=1s` (counter byte/packet). Verifikasi register tidak error dan
// terima ≥1 sample dalam 3.5s.
//
// Catatan: tidak assert StopStream() return true karena beberapa varian
// RouterOS emit !done setelah snapshot awal sehingga listener di-cleanup
// natural oleh stream manager.
func TestIntegration_InterfaceStatsStream(t *testing.T) {
	c := testutil.NewClient(t)
	net := network.New(c)

	const id = "it-iface-stats"
	var ticks atomic.Int32
	err := net.InterfaceStatsStream(id, 1*time.Second, func(s *roslib.Sentence) {
		if s.Word() == "!re" {
			ticks.Add(1)
		}
	})
	require.NoError(t, err)
	t.Cleanup(func() { net.StopStream(id) })

	time.Sleep(3500 * time.Millisecond)
	require.GreaterOrEqual(t, ticks.Load(), int32(1), "minimal 1 sample dalam 3.5s")
}
