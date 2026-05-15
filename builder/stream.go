package builder

import (
	"errors"
	"strconv"
	"time"

	"github.com/quiqxiq/roslib/stream"
)

// ErrNoStreamFlag dikembalikan StreamBuilder.Stream() bila chain selesai
// tanpa salah satu flag streaming aktif (Follow/FollowOnly/Interval).
// Tanpa flag tersebut RouterOS akan memperlakukan sentence sebagai
// query satu-shot — yang artinya listener akan langsung close.
var ErrNoStreamFlag = errors.New("builder: Stream() requires Follow(), FollowOnly(), or Interval(d)")

// StreamBuilder berasal dari PrintBuilder.Follow / FollowOnly / Interval.
// Field-nya composable — boleh Follow+Interval, FollowOnly+Interval, dst.
type StreamBuilder struct {
	p             *PrintBuilder
	follow        bool
	followOnly    bool
	interval      time.Duration
	queueSize     int
	cancelTimeout time.Duration
	onFinish      stream.FinishCallback
}

// Follow menambahkan flag `follow` ke sentence (event-driven).
func (s *StreamBuilder) Follow() *StreamBuilder { s.follow = true; return s }

// FollowOnly menambahkan flag `follow-only` (event tanpa snapshot awal).
// Mutually exclusive dengan Follow — kalau keduanya aktif, FollowOnly menang.
func (s *StreamBuilder) FollowOnly() *StreamBuilder { s.followOnly = true; return s }

// Interval menambahkan flag `interval=<d>` (RouterOS streaming polling).
// Cocok untuk command yang butuh interval untuk streaming, mis.
// `/queue/simple/print stats interval=1s` atau `/interface/print stats`.
func (s *StreamBuilder) Interval(d time.Duration) *StreamBuilder {
	s.interval = d
	return s
}

// QueueSize meng-override Client.Queue default untuk channel listener ini.
func (s *StreamBuilder) QueueSize(n int) *StreamBuilder {
	s.queueSize = n
	return s
}

// CancelTimeout mengatur batas waktu untuk perintah /cancel saat Unregister.
func (s *StreamBuilder) CancelTimeout(d time.Duration) *StreamBuilder {
	s.cancelTimeout = d
	return s
}

// OnFinish memasang callback yang dipanggil saat listener selesai. err == nil
// untuk natural completion (!done dari router), err != nil untuk connection
// drop. Lihat stream.FinishCallback untuk detail kontrak.
func (s *StreamBuilder) OnFinish(cb stream.FinishCallback) *StreamBuilder {
	s.onFinish = cb
	return s
}

// Stream mendaftarkan listener print-* ke StreamManager.
//
// Word yang dikirim: "{path}/print".
// Args yang di-prepend (urutan: streaming-flag dulu, lalu print-flag user):
//
//   - "follow-only" (kalau FollowOnly set), atau "follow" (kalau Follow set)
//   - "interval=<duration>" (kalau Interval set)
//
// Diikuti flag user (`stats`, `detail`, …) dan pairs/where seperti biasa.
//
// Validasi: minimal salah satu flag streaming aktif. Tanpa itu kembalikan
// ErrNoStreamFlag karena listener tanpa streaming flag akan langsung close
// (RouterOS perlakukan sebagai query biasa).
func (s *StreamBuilder) Stream(id string, h stream.Handler) error {
	if !s.follow && !s.followOnly && s.interval <= 0 {
		return ErrNoStreamFlag
	}

	prepend := make([]string, 0, 2)
	switch {
	case s.followOnly:
		prepend = append(prepend, "follow-only")
	case s.follow:
		prepend = append(prepend, "follow")
	}
	if s.interval > 0 {
		prepend = append(prepend, "interval="+formatDuration(s.interval))
	}

	args := append(prepend, s.p.flags...)

	spec := stream.Spec{
		ID:            id,
		Word:          s.p.path + "/print",
		Args:          args,
		Pairs:         s.p.pairs,
		Where:         s.p.where,
		Handler:       h,
		OnFinish:      s.onFinish,
		QueueSize:     s.queueSize,
		CancelTimeout: s.cancelTimeout,
	}
	return s.p.exec.RegisterStream(spec)
}

// formatDuration mengubah time.Duration menjadi format RouterOS yang ringkas,
// mis. "1s", "500ms", "2m". time.Duration.String() memberi "1m0s" untuk 1
// menit; kita inginkan "1m" supaya sentence lebih bersih.
func formatDuration(d time.Duration) string {
	switch {
	case d%time.Hour == 0:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	case d%time.Minute == 0:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d%time.Second == 0:
		return strconv.Itoa(int(d/time.Second)) + "s"
	case d%time.Millisecond == 0:
		return strconv.Itoa(int(d/time.Millisecond)) + "ms"
	}
	return d.String()
}
