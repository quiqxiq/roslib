package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestInvalidatePath_InMemory memastikan setelah Invalidate, Get jadi miss.
func TestInvalidatePath_InMemory(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()

	_ = c.SetForPath(ctx, "/ip/address", "k1", []byte("v1"), time.Hour)

	if _, hit, _ := c.Get(ctx, "k1"); !hit {
		t.Fatal("expected hit before invalidate")
	}
	if err := c.InvalidatePath(ctx, "/ip/address"); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := c.Get(ctx, "k1"); hit {
		t.Error("expected miss after invalidate")
	}
}

// TestInvalidatePath_Partial memastikan invalidate satu path tidak
// menghapus entry dari path lain.
func TestInvalidatePath_Partial(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()

	_ = c.SetForPath(ctx, "/ip/address", "addr1", []byte("a1"), time.Hour)
	_ = c.SetForPath(ctx, "/ip/address", "addr2", []byte("a2"), time.Hour)
	_ = c.SetForPath(ctx, "/ip/address", "addr3", []byte("a3"), time.Hour)
	_ = c.SetForPath(ctx, "/ip/route", "rt1", []byte("r1"), time.Hour)
	_ = c.SetForPath(ctx, "/ip/route", "rt2", []byte("r2"), time.Hour)

	if err := c.InvalidatePath(ctx, "/ip/address"); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"addr1", "addr2", "addr3"} {
		if _, hit, _ := c.Get(ctx, k); hit {
			t.Errorf("%s should be miss after invalidate", k)
		}
	}
	for _, k := range []string{"rt1", "rt2"} {
		if _, hit, _ := c.Get(ctx, k); !hit {
			t.Errorf("%s should still hit", k)
		}
	}
}

// TestInvalidatePath_NotFound memastikan invalidate path yang tidak ada
// tidak error.
func TestInvalidatePath_NotFound(t *testing.T) {
	c := NewInMemory()
	if err := c.InvalidatePath(context.Background(), "/nope"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestInvalidatePath_AfterExpiry memastikan TTL expiry tidak bocor reference
// di pathIdx (regress prevention).
func TestInvalidatePath_AfterExpiry(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()

	_ = c.SetForPath(ctx, "/ip/address", "k1", []byte("v1"), 20*time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	// Trigger lazy expiry via Get
	if _, hit, _ := c.Get(ctx, "k1"); hit {
		t.Error("expected expired miss")
	}
	// Invalidate harus jadi no-op (pathIdx sudah bersih atau membersih sendiri)
	if err := c.InvalidatePath(ctx, "/ip/address"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestNoopInvalidate memastikan NoopCache.InvalidatePath tidak error.
func TestNoopInvalidate(t *testing.T) {
	var c Cache = NoopCache{}
	if err := c.InvalidatePath(context.Background(), "/ip/address"); err != nil {
		t.Errorf("NoopCache.InvalidatePath should be no-op, got %v", err)
	}
}

// TestConcurrentSetInvalidate menjamin tidak ada race / panic saat
// Set/Invalidate concurrent. Test ini fokus ke goroutine safety, bukan
// correctness — hasil akhir bisa apa saja (race condition di domain).
func TestConcurrentSetInvalidate(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()
	stop := make(chan struct{})

	var wg sync.WaitGroup

	// Goroutine A: terus Set untuk berbagai path.
	wg.Add(1)
	go func() {
		defer wg.Done()
		paths := []string{"/ip/address", "/ip/route", "/queue/simple"}
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				p := paths[i%len(paths)]
				key := p + ":k"
				_ = c.SetForPath(ctx, p, key, []byte("v"), time.Hour)
				i++
			}
		}
	}()

	// Goroutine B: terus Invalidate dan Get.
	wg.Add(1)
	go func() {
		defer wg.Done()
		paths := []string{"/ip/address", "/ip/route", "/queue/simple", "/nonexistent"}
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				_ = c.InvalidatePath(ctx, paths[i%len(paths)])
				_, _, _ = c.Get(ctx, paths[i%len(paths)]+":k")
				i++
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// TestSetWithoutPath memastikan Set (tanpa SetForPath) tidak ter-track di
// pathIdx — sehingga InvalidatePath tidak menghapusnya. Ini design tradeoff:
// path tracking opt-in lewat SetForPath.
func TestSetWithoutPath(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()

	_ = c.Set(ctx, "untracked-key", []byte("v"), time.Hour)
	_ = c.InvalidatePath(ctx, "/anything")

	if _, hit, _ := c.Get(ctx, "untracked-key"); !hit {
		t.Error("Set (non-path) should not be affected by InvalidatePath")
	}
}

// TestInvalidatePathForDevice_Scoped verifikasi scoped invalidate hanya hapus
// entry milik deviceID tertentu. Skenario fleet dengan shared cache: rb1 +
// rb2 sama-sama cache "/ip/address"; invalidate di rb1 tidak boleh nge-bust
// entry rb2.
func TestInvalidatePathForDevice_Scoped(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()

	rb1Key := KeyOf("rb1", []string{"/ip/address/print"})
	rb2Key := KeyOf("rb2", []string{"/ip/address/print"})

	_ = c.SetForPath(ctx, "/ip/address", rb1Key, []byte("rb1-data"), time.Hour)
	_ = c.SetForPath(ctx, "/ip/address", rb2Key, []byte("rb2-data"), time.Hour)

	if err := c.InvalidatePathForDevice(ctx, "rb1", "/ip/address"); err != nil {
		t.Fatalf("scoped invalidate: %v", err)
	}

	if _, hit, _ := c.Get(ctx, rb1Key); hit {
		t.Error("rb1 entry should be invalidated")
	}
	if _, hit, _ := c.Get(ctx, rb2Key); !hit {
		t.Error("rb2 entry should be preserved (different device)")
	}

	// Invalidate rb2 also — pathIdx[path] harus hilang sekarang.
	if err := c.InvalidatePathForDevice(ctx, "rb2", "/ip/address"); err != nil {
		t.Fatalf("scoped invalidate rb2: %v", err)
	}
	if _, hit, _ := c.Get(ctx, rb2Key); hit {
		t.Error("rb2 entry should be invalidated after second call")
	}
}

// TestInvalidatePathForDevice_NoMatch — deviceID yang tidak punya entry
// adalah no-op.
func TestInvalidatePathForDevice_NoMatch(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()

	rb1Key := KeyOf("rb1", []string{"/ip/address/print"})
	_ = c.SetForPath(ctx, "/ip/address", rb1Key, []byte("rb1-data"), time.Hour)

	if err := c.InvalidatePathForDevice(ctx, "nonexistent", "/ip/address"); err != nil {
		t.Fatalf("scoped invalidate nonexistent: %v", err)
	}
	if _, hit, _ := c.Get(ctx, rb1Key); !hit {
		t.Error("rb1 entry should be preserved")
	}
}
