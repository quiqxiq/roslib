package builder

import (
	"time"

	"github.com/quiqxiq/roslib/stream"
)

// streamableMode menentukan flag yang ditambahkan saat menyusun listener
// dari PrintBuilder.Follow / FollowOnly.
type streamableMode int

const (
	modeFollow streamableMode = iota
	modeFollowOnly
)

func (m streamableMode) flag() string {
	if m == modeFollowOnly {
		return "follow-only"
	}
	return "follow"
}

// StreamBuilder berasal dari PrintBuilder.Follow / FollowOnly.
// Memungkinkan tweak queue size + cancel timeout sebelum Register.
type StreamBuilder struct {
	p             *PrintBuilder
	mode          streamableMode
	queueSize     int
	cancelTimeout time.Duration
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

// Stream mendaftarkan listener print-follow ke StreamManager.
// Word yang dikirim: "{path}/print" dengan flag follow/follow-only di depan.
func (s *StreamBuilder) Stream(id string, h stream.Handler) error {
	flags := append([]string{s.mode.flag()}, s.p.flags...)
	spec := stream.Spec{
		ID:            id,
		Word:          s.p.path + "/print",
		Args:          flags,
		Pairs:         s.p.pairs,
		Where:         s.p.where,
		Handler:       h,
		QueueSize:     s.queueSize,
		CancelTimeout: s.cancelTimeout,
	}
	return s.p.exec.RegisterStream(spec)
}
