//go:build integration

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/syslog"
	"github.com/stretchr/testify/require"
)

// TestIntegration_LogStream menguji `/log/print follow-only` (analisis §1.5).
// Pola sama dengan roslib cmd/integration STREAM/LOG-FOLLOW: subscribe, tunggu
// 3 detik, hentikan. Router idle umumnya tetap menghasilkan entry log.
func TestIntegration_LogStream(t *testing.T) {
	c := testutil.NewClient(t)
	sys := syslog.New(c)

	const id = "it-log-stream"
	var got atomic.Int32
	err := sys.LogStream(id, "", func(s *roslib.Sentence) {
		got.Add(1)
		if got.Load() <= 3 {
			t.Logf("log[%d] %s | %s | %s", got.Load(), s.Get("time"), s.Get("topics"), s.Get("message"))
		}
	})
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	sys.StopLogStream(id)
	t.Logf("log stream: %d entries dalam 3s (0 wajar jika router idle)", got.Load())
}

// TestIntegration_LogStreamFiltered menguji LogStream dengan filter topics
// non-kosong. Tidak assert count — beberapa router idle tidak punya event
// untuk topics filter sempit.
func TestIntegration_LogStreamFiltered(t *testing.T) {
	c := testutil.NewClient(t)
	sys := syslog.New(c)

	const id = "it-log-stream-filtered"
	var got atomic.Int32
	err := sys.LogStream(id, "system", func(s *roslib.Sentence) {
		got.Add(1)
	})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)
	sys.StopLogStream(id)
	t.Logf("log stream filtered: %d entries (topics=system)", got.Load())
}
