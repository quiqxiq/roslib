// Package poll menyediakan PollEngine: batching command polling RouterOS
// berdasarkan interval (interval-group batching). Tiap interval unik berbagi
// satu ticker — di tiap tick semua command di group di-fire concurrent via
// connection async (tag demux), bukan serial.
package poll

import (
	"time"

	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/query"
)

// Handler dipanggil untuk setiap !re sentence dari hasil poll.
// Dipanggil di goroutine internal poll engine — handler harus return cepat.
type Handler func(*decode.Sentence)

// Config mendeskripsikan satu poll: command apa, seberapa sering, dan
// handler hasilnya.
//
// ID unik per Engine; dipakai untuk Unregister.
// Args adalah kata-kata sentence setelah Path. Elemen pertama adalah
// action (default "print"); elemen berikutnya adalah flag bebas seperti
// "detail" atau "stats". Pairs (=k=v) dan Where (?k=v) ditambahkan
// sesudahnya oleh query.BuildSentence.
type Config struct {
	ID       string
	Path     string
	Args     []string
	Pairs    []query.Pair
	Where    []query.WherePair
	Interval time.Duration
	Handler  Handler

	// Timeout per command. Kalau 0 → engine pakai Interval sebagai cap.
	Timeout time.Duration

	// MaxInFlight membatasi jumlah command yang sedang berjalan untuk poll
	// ini secara global. 0 = tak terbatas. Berguna saat router lambat:
	// kalau tick sebelumnya belum selesai, tick baru di-skip.
	MaxInFlight int
}

// command membangun sentence yang dikirim ke router.
//
// Args[0] dipakai sebagai action ("print" default), sisanya jadi flag.
func (c *Config) command() []string {
	args := c.Args
	if len(args) == 0 {
		args = []string{"print"}
	}
	action := args[0]
	flags := args[1:]
	return query.BuildSentence(c.Path+"/"+action, flags, c.Pairs, c.Where)
}

// effectiveTimeout mengembalikan Timeout atau fallback ke Interval.
func (c *Config) effectiveTimeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return c.Interval
}
