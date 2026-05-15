// Package stream menyediakan StreamManager: mengelola listener long-running
// di koneksi RouterOS dengan mode async (tag demux). Semua listener berbagi
// satu koneksi (connStream); jika koneksi reconnect, ReattachAll
// mendaftarkan ulang seluruh listener.
package stream

import (
	"context"
	"time"

	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/query"
)

// Handler dipanggil untuk setiap !re sentence dari listener.
// Dipanggil di goroutine internal — handler harus return cepat.
type Handler func(*decode.Sentence)

// Spec adalah blueprint listener — bukan instance live. Spec disimpan oleh
// Manager untuk keperluan re-attach pasca reconnect.
//
// Word adalah first word sentence yang dikirim ke router, mis.
// "/log/print" (untuk streamable-print) atau "/interface/monitor-traffic"
// (untuk inherent-streaming). Builder yang bertanggung jawab menyusun Word
// secara benar.
//
// Args adalah flag-word tambahan (mis. "follow", "follow-only", "interval=1s")
// yang dirangkai setelah Word.
type Spec struct {
	ID      string
	Word    string
	Args    []string
	Pairs   []query.Pair
	Where   []query.WherePair
	Handler Handler

	// QueueSize override Client.Queue default. 0 = pakai default dari client.
	QueueSize int

	// CancelTimeout adalah timeout saat Unregister mengirim /cancel.
	// 0 = pakai default (5 detik).
	CancelTimeout time.Duration
}

// defaultCancelTimeout adalah waktu tunggu maksimum untuk perintah /cancel
// ke router saat listener di-unregister.
const defaultCancelTimeout = 5 * time.Second

// command membangun sentence Listen.
func (s *Spec) command() []string {
	return query.BuildSentence(s.Word, s.Args, s.Pairs, s.Where)
}

func (s *Spec) cancelTimeout() time.Duration {
	if s.CancelTimeout > 0 {
		return s.CancelTimeout
	}
	return defaultCancelTimeout
}

// listener adalah instance hidup dari Spec: pointer ke reply RouterOS + ctx.
type listener struct {
	spec   Spec
	reply  listenReply
	cancel context.CancelFunc
}
