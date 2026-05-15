package builder

import (
	"time"

	"github.com/quiqxiq/roslib/query"
	"github.com/quiqxiq/roslib/stream"
)

// PathBuilder adalah entry point chain untuk satu path RouterOS, mis.
// "/ip/address" (composable dengan Print/Add/Set/Remove) atau
// "/interface/monitor-traffic" (inherent-streaming yang dipakai .Stream()
// langsung dari sini).
type PathBuilder struct {
	exec  Executor
	path  string
	pairs []query.Pair
	args  []string
}

// New membuat PathBuilder baru. Biasanya dipanggil oleh device.Path().
func New(exec Executor, path string) *PathBuilder {
	return &PathBuilder{exec: exec, path: path}
}

// Path mengembalikan path RouterOS yang dipegang builder.
func (b *PathBuilder) Path() string { return b.path }

// With menambahkan pair "=key=value" — dipakai inherent-streaming
// (mis. `Path("/tool/ping").With("address","8.8.8.8")`) atau dipakai
// di Run/Add helper level path.
func (b *PathBuilder) With(key, value string) *PathBuilder {
	b.pairs = append(b.pairs, query.NewPair(key, value))
	return b
}

// Arg menambahkan flag-word bebas (tanpa "=") untuk inherent-streaming.
// Contoh: `Path("/tool/ping").Arg("arp-ping").With("address","x")`.
func (b *PathBuilder) Arg(word string) *PathBuilder {
	b.args = append(b.args, word)
	return b
}

// Print memulai chain Print: tambahkan Where/Args lalu Exec / Stream.
func (b *PathBuilder) Print() *PrintBuilder {
	return &PrintBuilder{exec: b.exec, path: b.path}
}

// Stream mendaftarkan inherently-streaming command (monitor-traffic, ping,
// torch, sniffer, .../monitor, dll). Tidak menambahkan kata "follow" —
// router emit !re langsung dari command itu sendiri.
//
// Word sentence: Path apa adanya. Pair via With() dan flag via Arg().
//
// Untuk print-with-follow (mis. /log atau /ip/firewall/filter), gunakan
// Print().Follow().Stream() / Print().FollowOnly().Stream() — bukan
// method ini.
func (b *PathBuilder) Stream(id string, h stream.Handler) error {
	spec := stream.Spec{
		ID:      id,
		Word:    b.path,
		Args:    b.args,
		Pairs:   b.pairs,
		Handler: h,
	}
	return b.exec.RegisterStream(spec)
}

// CancelTimeout passthrough untuk stream.Spec.CancelTimeout. Optional.
// Dipakai sebelum Stream() kalau /cancel butuh window khusus.
type withCancelTimeout struct {
	pb *PathBuilder
	d  time.Duration
}

// CancelTimeout returns a wrapper that, when followed by Stream(),
// applies the given /cancel timeout. Sengaja terpisah agar method-chain
// PathBuilder tetap pendek untuk use-case umum.
func (b *PathBuilder) CancelTimeout(d time.Duration) *withCancelTimeout {
	return &withCancelTimeout{pb: b, d: d}
}

// Stream dengan custom cancel timeout.
func (w *withCancelTimeout) Stream(id string, h stream.Handler) error {
	spec := stream.Spec{
		ID:            id,
		Word:          w.pb.path,
		Args:          w.pb.args,
		Pairs:         w.pb.pairs,
		Handler:       h,
		CancelTimeout: w.d,
	}
	return w.pb.exec.RegisterStream(spec)
}
