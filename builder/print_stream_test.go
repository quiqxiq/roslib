package builder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/stream"
	"github.com/sirupsen/logrus"
)

// fakeExecutor adalah implementasi Executor minimal untuk unit test:
// merekam sentence yang dikirim & spec yang di-register tanpa benar-benar
// menyentuh router.
type fakeExecutor struct {
	c cache.Cache

	runCalls    [][]string
	streamSpecs []stream.Spec
	cancelIDs   []string

	deviceID string
}

func newFakeExecutor(t *testing.T) *fakeExecutor {
	t.Helper()
	return &fakeExecutor{
		c:        cache.NoopCache{},
		deviceID: "test-device",
	}
}

func (f *fakeExecutor) RunCommand(ctx context.Context, sentence []string) (*routeros.Reply, error) {
	f.runCalls = append(f.runCalls, append([]string(nil), sentence...))
	return &routeros.Reply{}, nil
}
func (f *fakeExecutor) RegisterStream(spec stream.Spec) error {
	f.streamSpecs = append(f.streamSpecs, spec)
	return nil
}
func (f *fakeExecutor) UnregisterStream(id string) bool {
	f.cancelIDs = append(f.cancelIDs, id)
	return true
}
func (f *fakeExecutor) Cache() cache.Cache      { return f.c }
func (f *fakeExecutor) CacheTTL() time.Duration { return 0 }
func (f *fakeExecutor) Logger() *logrus.Entry   { return logrus.NewEntry(logrus.New()) }
func (f *fakeExecutor) DeviceID() string        { return f.deviceID }

// ──────────────── tests ────────────────

func TestPrintBuilderStreamRequiresFlag(t *testing.T) {
	ex := newFakeExecutor(t)
	pb := New(ex, "/queue/simple").Print().Stats()
	// StreamBuilder tidak punya .Stream() public di chain ini — kita
	// harus dapat StreamBuilder dulu lewat Follow/FollowOnly/Interval.
	// Untuk test bahwa Stream() tanpa flag tidak ada: kompilasi sendiri
	// memverifikasi. Test eksplisit: StreamBuilder kosong → error.
	sb := &StreamBuilder{p: pb}
	if err := sb.Stream("q", func(s *decode.Sentence) {}); !errors.Is(err, ErrNoStreamFlag) {
		t.Errorf("expected ErrNoStreamFlag, got %v", err)
	}
}

func TestPrintBuilderIntervalStream(t *testing.T) {
	ex := newFakeExecutor(t)
	dev := New(ex, "/queue/simple")
	err := dev.Print().Stats().Interval(time.Second).Stream("q", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ex.streamSpecs) != 1 {
		t.Fatalf("expected 1 stream spec, got %d", len(ex.streamSpecs))
	}
	spec := ex.streamSpecs[0]
	if spec.Word != "/queue/simple/print" {
		t.Errorf("Word = %q", spec.Word)
	}
	if !hasArg(spec.Args, "interval=1s") {
		t.Errorf("missing interval=1s in %v", spec.Args)
	}
	if !hasArg(spec.Args, "stats") {
		t.Errorf("missing stats in %v", spec.Args)
	}
}

func TestPrintBuilderFollowAndInterval(t *testing.T) {
	ex := newFakeExecutor(t)
	dev := New(ex, "/ip/firewall/filter")
	err := dev.Print().Follow().Interval(2 * time.Second).Stream("fw", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	spec := ex.streamSpecs[0]
	if !hasArg(spec.Args, "follow") {
		t.Errorf("missing follow in %v", spec.Args)
	}
	if !hasArg(spec.Args, "interval=2s") {
		t.Errorf("missing interval=2s in %v", spec.Args)
	}
}

func TestPrintBuilderFollowOnlyStream(t *testing.T) {
	ex := newFakeExecutor(t)
	dev := New(ex, "/log")
	err := dev.Print().FollowOnly().Stream("log", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	spec := ex.streamSpecs[0]
	if !hasArg(spec.Args, "follow-only") {
		t.Errorf("missing follow-only in %v", spec.Args)
	}
	if hasArg(spec.Args, "follow") {
		t.Errorf("should NOT have plain follow in %v", spec.Args)
	}
}

func TestPrintBuilderProplist(t *testing.T) {
	ex := newFakeExecutor(t)
	pb := New(ex, "/ip/address").Print().Proplist("address", "interface")
	sentence := pb.command()
	found := false
	for _, w := range sentence {
		if w == "proplist=address,interface" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing proplist arg in %v", sentence)
	}
}

func TestFormatDurationVariants(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{time.Second, "1s"},
		{30 * time.Second, "30s"},
		{time.Minute, "1m"},
		{2 * time.Minute, "2m"},
		{time.Hour, "1h"},
	}
	for _, tc := range cases {
		if got := formatDuration(tc.in); got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// hasArg cek apakah sebuah word ada di slice (linear scan kecil).
func hasArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

// ──────────────── tests untuk finite-stream (count/duration) ────────────────

// TestPathBuilderStreamWithCount verifikasi PathBuilder.With("count","5")
// menghasilkan spec.Pairs yang berisi count=5. Ini krusial untuk finite
// stream cleanup — router akan kirim !done setelah count habis, lalu
// Manager.consume() harus auto-hapus entry.
func TestPathBuilderStreamWithCount(t *testing.T) {
	ex := newFakeExecutor(t) // /tool/ping inherent-streaming, registry mungkin v7 vs v6 → non-strict
	err := New(ex, "/tool/ping").
		With("address", "8.8.8.8").
		With("count", "5").
		Stream("ping-5", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("Stream() err: %v", err)
	}
	if len(ex.streamSpecs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(ex.streamSpecs))
	}
	spec := ex.streamSpecs[0]
	if spec.Word != "/tool/ping" {
		t.Errorf("Word = %q; want /tool/ping", spec.Word)
	}

	// Spec.Pairs adalah []query.Pair — kita verify via command() yang
	// merangkai jadi sentence "=key=value".
	sentence := buildSentenceFromSpec(spec)
	if !hasArg(sentence, "=address=8.8.8.8") {
		t.Errorf("missing =address=8.8.8.8 in %v", sentence)
	}
	if !hasArg(sentence, "=count=5") {
		t.Errorf("missing =count=5 in %v", sentence)
	}
}

// TestPathBuilderStreamWithDuration verifikasi torch finite-stream
// dengan duration=2s.
func TestPathBuilderStreamWithDuration(t *testing.T) {
	ex := newFakeExecutor(t)
	err := New(ex, "/tool/torch").
		With("interface", "ether1").
		With("duration", "2s").
		Stream("torch-2s", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("Stream() err: %v", err)
	}
	spec := ex.streamSpecs[0]
	sentence := buildSentenceFromSpec(spec)
	if !hasArg(sentence, "=interface=ether1") {
		t.Errorf("missing =interface=ether1 in %v", sentence)
	}
	if !hasArg(sentence, "=duration=2s") {
		t.Errorf("missing =duration=2s in %v", sentence)
	}
}

// TestPathBuilderOnFinishWired verifikasi OnFinish() di PathBuilder
// memang ter-pasang ke Spec — supaya Manager.consume() bisa fire callback
// saat !done tiba.
func TestPathBuilderOnFinishWired(t *testing.T) {
	ex := newFakeExecutor(t)
	called := false
	cb := func(id string, err error) { called = true }

	err := New(ex, "/tool/ping").
		With("address", "1.1.1.1").
		With("count", "3").
		OnFinish(cb).
		Stream("ping-cb", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("Stream() err: %v", err)
	}
	spec := ex.streamSpecs[0]
	if spec.OnFinish == nil {
		t.Fatal("spec.OnFinish not wired")
	}
	// Invoke directly to verify identity.
	spec.OnFinish("ping-cb", nil)
	if !called {
		t.Error("OnFinish closure not invoked")
	}
}

// TestStreamBuilderOnFinishWired verifikasi OnFinish() di StreamBuilder
// (print+interval/follow) ter-pasang ke Spec.
func TestStreamBuilderOnFinishWired(t *testing.T) {
	ex := newFakeExecutor(t)
	called := false
	cb := func(id string, err error) { called = true }

	err := New(ex, "/queue/simple").Print().Stats().
		Interval(time.Second).
		OnFinish(cb).
		Stream("q-cb", func(s *decode.Sentence) {})
	if err != nil {
		t.Fatalf("Stream() err: %v", err)
	}
	spec := ex.streamSpecs[0]
	if spec.OnFinish == nil {
		t.Fatal("spec.OnFinish not wired")
	}
	spec.OnFinish("q-cb", nil)
	if !called {
		t.Error("OnFinish closure not invoked")
	}
}

// buildSentenceFromSpec membangun sentence equivalent untuk asersi.
func buildSentenceFromSpec(spec stream.Spec) []string {
	out := []string{spec.Word}
	out = append(out, spec.Args...)
	for _, p := range spec.Pairs {
		out = append(out, p.Word())
	}
	for _, w := range spec.Where {
		out = append(out, w.Word())
	}
	return out
}

