package cache

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryGetMiss(t *testing.T) {
	c := NewInMemory()
	v, hit, err := c.Get(context.Background(), "nope")
	if err != nil {
		t.Fatal(err)
	}
	if hit || v != nil {
		t.Errorf("expected miss, got hit=%v val=%v", hit, v)
	}
}

func TestInMemorySetGet(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	v, hit, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Fatal("expected hit")
	}
	if string(v) != "v" {
		t.Errorf("got %q, want v", string(v))
	}
}

func TestInMemoryExpiry(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()
	_ = c.Set(ctx, "k", []byte("v"), 20*time.Millisecond)

	if _, hit, _ := c.Get(ctx, "k"); !hit {
		t.Fatal("expected hit before expiry")
	}
	time.Sleep(40 * time.Millisecond)
	if _, hit, _ := c.Get(ctx, "k"); hit {
		t.Error("expected miss after expiry")
	}
}

func TestNoopCache(t *testing.T) {
	var c Cache = NoopCache{}
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), time.Second); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := c.Get(ctx, "k"); hit {
		t.Error("NoopCache should never hit")
	}
}

func TestKeyOfDeterministic(t *testing.T) {
	k1 := KeyOf([]string{"/ip/address/print", "detail"})
	k2 := KeyOf([]string{"/ip/address/print", "detail"})
	if k1 != k2 {
		t.Error("KeyOf should be deterministic")
	}
	k3 := KeyOf([]string{"/ip/address/print"})
	if k1 == k3 {
		t.Error("KeyOf should differ for different sentences")
	}
}
