package builder

import (
	"context"
	"testing"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roslib/cache"
)

// fakeCachedExecutor extends fakeExecutor dengan reply yang punya isi
// (Re slice non-empty) supaya ExecCached benar-benar punya sesuatu untuk
// disimpan ke cache.
type fakeCachedExecutor struct {
	*fakeExecutor
	canned *routeros.Reply
	calls  int
}

func (f *fakeCachedExecutor) RunCommand(ctx context.Context, sentence []string) (*routeros.Reply, error) {
	f.calls++
	return f.canned, nil
}

func newFakeCachedExecutor(t *testing.T) *fakeCachedExecutor {
	base := newFakeExecutor(t)
	base.c = cache.NewInMemory()
	return &fakeCachedExecutor{
		fakeExecutor: base,
		canned: &routeros.Reply{
			Re: []*proto.Sentence{
				{Word: "!re", Map: map[string]string{"address": "10.0.0.1/24", "interface": "ether1"}},
				{Word: "!re", Map: map[string]string{"address": "10.0.0.2/24", "interface": "ether2"}},
			},
		},
	}
}

// TestExecCachedHit memastikan call kedua dalam TTL hit cache (tidak panggil
// RunCommand lagi).
func TestExecCachedHit(t *testing.T) {
	ex := newFakeCachedExecutor(t)
	dev := New(ex, "/ip/address")
	ctx := context.Background()

	if _, err := dev.Print().ExecCached(ctx, time.Hour); err != nil {
		t.Fatal(err)
	}
	if ex.calls != 1 {
		t.Fatalf("after first call, RunCommand calls = %d, want 1", ex.calls)
	}

	reply, err := dev.Print().ExecCached(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if ex.calls != 1 {
		t.Errorf("after second call (hit), RunCommand calls = %d, want 1", ex.calls)
	}
	if len(reply.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(reply.Rows))
	}
	if got := reply.Rows[0].Get("address"); got != "10.0.0.1/24" {
		t.Errorf("address = %q", got)
	}
}

// TestExecCachedInvalidate memastikan setelah InvalidatePath, ExecCached
// hit router lagi.
func TestExecCachedInvalidate(t *testing.T) {
	ex := newFakeCachedExecutor(t)
	dev := New(ex, "/ip/address")
	ctx := context.Background()

	_, _ = dev.Print().ExecCached(ctx, time.Hour) // call 1 → router
	_, _ = dev.Print().ExecCached(ctx, time.Hour) // call 2 → cache
	if ex.calls != 1 {
		t.Fatalf("setup expected calls=1, got %d", ex.calls)
	}

	if err := ex.Cache().InvalidatePath(ctx, "/ip/address"); err != nil {
		t.Fatal(err)
	}

	_, _ = dev.Print().ExecCached(ctx, time.Hour) // call 3 → router (cache empty)
	if ex.calls != 2 {
		t.Errorf("after invalidate, RunCommand calls = %d, want 2", ex.calls)
	}
}

// TestExecCachedInvalidateOtherPath: invalidate path yang berbeda tidak
// mempengaruhi cache untuk path lain.
func TestExecCachedInvalidateOtherPath(t *testing.T) {
	ex := newFakeCachedExecutor(t)
	dev := New(ex, "/ip/address")
	ctx := context.Background()

	_, _ = dev.Print().ExecCached(ctx, time.Hour)
	_, _ = ex.Cache().InvalidatePath(ctx, "/ip/route"), error(nil)

	_, _ = dev.Print().ExecCached(ctx, time.Hour)
	if ex.calls != 1 {
		t.Errorf("invalidate other path should not affect, calls = %d, want 1", ex.calls)
	}
}

// TestExecCachedDeviceScoping: dua device dengan ID berbeda tidak share cache.
func TestExecCachedDeviceScoping(t *testing.T) {
	sharedCache := cache.NewInMemory()
	exA := newFakeExecutor(t)
	exA.c = sharedCache
	exA.deviceID = "dev-A"
	exB := newFakeExecutor(t)
	exB.c = sharedCache
	exB.deviceID = "dev-B"

	// Kasih kedua fake executor reply yang sama tapi count terpisah.
	calls := map[string]int{}
	fakeRun := func(id string) func(context.Context, []string) (*routeros.Reply, error) {
		return func(ctx context.Context, sentence []string) (*routeros.Reply, error) {
			calls[id]++
			return &routeros.Reply{Re: []*proto.Sentence{
				{Word: "!re", Map: map[string]string{"device": id}},
			}}, nil
		}
	}

	// Wrap exec → custom RunCommand.
	wA := wrapExecutor{Executor: exA, run: fakeRun("A")}
	wB := wrapExecutor{Executor: exB, run: fakeRun("B")}

	ctx := context.Background()
	_, _ = New(&wA, "/ip/address").Print().ExecCached(ctx, time.Hour)
	_, _ = New(&wB, "/ip/address").Print().ExecCached(ctx, time.Hour)

	if calls["A"] != 1 || calls["B"] != 1 {
		t.Errorf("device-scoped cache should not share: A=%d B=%d", calls["A"], calls["B"])
	}

	// Call A lagi → hit cache.
	_, _ = New(&wA, "/ip/address").Print().ExecCached(ctx, time.Hour)
	if calls["A"] != 1 {
		t.Errorf("A second call should hit cache, calls = %d", calls["A"])
	}
}

// wrapExecutor membungkus fakeExecutor + override RunCommand.
type wrapExecutor struct {
	Executor
	run func(context.Context, []string) (*routeros.Reply, error)
}

func (w *wrapExecutor) RunCommand(ctx context.Context, sentence []string) (*routeros.Reply, error) {
	return w.run(ctx, sentence)
}
