package influx

import (
	"context"
	"sync"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/quiqxiq/roslib/decode"
)

// TagFn mengekstrak tags dari satu sentence RouterOS untuk diisi sebagai
// Influx tags. Jangan kembalikan nil map bila tidak ada tag — return map
// kosong supaya iterasi aman.
type TagFn func(*decode.Sentence) map[string]string

// FieldFn mengekstrak fields dari satu sentence. Value boleh string, int64,
// float64, bool, atau time.Time — SDK akan handle encoding line-protocol.
type FieldFn func(*decode.Sentence) map[string]any

// Writer mengubah *decode.Sentence menjadi *influxdb3.Point + menulisnya.
// Aman dipanggil concurrent — *influxdb3.Client thread-safe.
type Writer struct {
	cli  *influxdb3.Client
	meas string
	tags TagFn
	flds FieldFn
}

// NewWriter membuat Writer dengan satu measurement + ekstraktor tag/field.
func NewWriter(cli *influxdb3.Client, measurement string, tags TagFn, fields FieldFn) *Writer {
	if tags == nil {
		tags = func(*decode.Sentence) map[string]string { return nil }
	}
	if fields == nil {
		fields = func(*decode.Sentence) map[string]any { return nil }
	}
	return &Writer{cli: cli, meas: measurement, tags: tags, flds: fields}
}

// BuildPoint mengkonversi satu sentence ke *influxdb3.Point.
// Timestamp = now; user boleh override lewat opsi (lihat WriteSentenceAt).
func (w *Writer) BuildPoint(s *decode.Sentence) *influxdb3.Point {
	p := influxdb3.NewPointWithMeasurement(w.meas).SetTimestamp(time.Now())
	for k, v := range w.tags(s) {
		p.SetTag(k, v)
	}
	for k, v := range w.flds(s) {
		p.SetField(k, v)
	}
	return p
}

// WriteSentence kirim satu sentence sebagai satu point ke Influx.
func (w *Writer) WriteSentence(ctx context.Context, s *decode.Sentence) error {
	return w.cli.WritePoints(ctx, []*influxdb3.Point{w.BuildPoint(s)})
}

// WriteBatch kirim batch point yang sudah dibangun user.
// Berguna kalau user ingin batch beberapa sentence per round-trip.
func (w *Writer) WriteBatch(ctx context.Context, points []*influxdb3.Point) error {
	return w.cli.WritePoints(ctx, points)
}

// ──────────────── BatchedWriter ────────────────

// BatchedWriter membungkus Writer dengan in-memory buffer + flush
// berkala atau saat buffer penuh. Goroutine flusher dimulai oleh Start
// dan dihentikan oleh Close.
//
// Tujuan: kurangi jumlah HTTP request saat poll high-frequency.
type BatchedWriter struct {
	w *Writer

	maxSize  int
	interval time.Duration

	mu     sync.Mutex
	buf    []*influxdb3.Point
	flush  chan struct{}
	done   chan struct{}
	cancel context.CancelFunc
}

// NewBatchedWriter membuat batcher. maxSize ≤ 0 → default 1000.
// interval ≤ 0 → default 5 detik.
func NewBatchedWriter(w *Writer, maxSize int, interval time.Duration) *BatchedWriter {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &BatchedWriter{
		w:        w,
		maxSize:  maxSize,
		interval: interval,
		buf:      make([]*influxdb3.Point, 0, maxSize),
		flush:    make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
}

// Start meluncurkan goroutine flusher. Aman dipanggil maksimal sekali.
func (b *BatchedWriter) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	b.cancel = cancel
	go b.loop(ctx)
}

// Add menambahkan point ke buffer. Trigger flush kalau buffer penuh.
func (b *BatchedWriter) Add(p *influxdb3.Point) {
	b.mu.Lock()
	b.buf = append(b.buf, p)
	full := len(b.buf) >= b.maxSize
	b.mu.Unlock()
	if full {
		select {
		case b.flush <- struct{}{}:
		default:
		}
	}
}

// AddSentence shortcut: BuildPoint dari Writer lalu Add.
func (b *BatchedWriter) AddSentence(s *decode.Sentence) {
	b.Add(b.w.BuildPoint(s))
}

// Close trigger flush final + hentikan goroutine.
func (b *BatchedWriter) Close(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	<-b.done
	return b.drain(ctx)
}

func (b *BatchedWriter) loop(ctx context.Context) {
	defer close(b.done)
	t := time.NewTicker(b.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = b.drain(ctx)
		case <-b.flush:
			_ = b.drain(ctx)
		}
	}
}

func (b *BatchedWriter) drain(ctx context.Context) error {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return nil
	}
	snapshot := b.buf
	b.buf = make([]*influxdb3.Point, 0, b.maxSize)
	b.mu.Unlock()
	return b.w.WriteBatch(ctx, snapshot)
}
