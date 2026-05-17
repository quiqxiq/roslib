//go:build integration

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/stretchr/testify/require"
)

// TestIntegration_HotspotActiveStream menguji ActiveStream snapshot-and-follow
// dapat di-register tanpa error.
//
// Catatan: pada RouterOS tertentu, `/ip/hotspot/active/print follow` dengan
// tabel kosong langsung mengirim !done — listener di-cleanup natural oleh
// stream manager. Test tidak assert StopActiveStream() return true karena
// natural cleanup adalah perilaku valid.
func TestIntegration_HotspotActiveStream(t *testing.T) {
	c := testutil.NewClient(t)
	hot := hotspot.New(c)

	const id = "it-active-stream"
	var got atomic.Int32
	err := hot.ActiveStream(id, func(s *roslib.Sentence) {
		got.Add(1)
		t.Logf("active stream sentence: word=%s user=%s", s.Word(), s.Get("user"))
	})
	require.NoError(t, err)
	t.Cleanup(func() { hot.StopActiveStream(id) })

	time.Sleep(2 * time.Second)
	t.Logf("active stream: %d sentences diterima", got.Load())
}

// TestIntegration_HotspotActiveStreamFollowOnly menguji FollowOnly variant —
// hanya event baru, tanpa snapshot awal.
func TestIntegration_HotspotActiveStreamFollowOnly(t *testing.T) {
	c := testutil.NewClient(t)
	hot := hotspot.New(c)

	const id = "it-active-follow-only"
	var got atomic.Int32
	err := hot.ActiveStreamFollowOnly(id, func(s *roslib.Sentence) {
		got.Add(1)
	})
	require.NoError(t, err)
	t.Cleanup(func() { hot.StopActiveStream(id) })

	time.Sleep(1 * time.Second)
	t.Logf("active follow-only: %d sentences (kosong wajar — hanya event baru)", got.Load())
}
