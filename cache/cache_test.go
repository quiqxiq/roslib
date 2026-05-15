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
	k1 := KeyOf("dev1", []string{"/ip/address/print", "detail"})
	k2 := KeyOf("dev1", []string{"/ip/address/print", "detail"})
	if k1 != k2 {
		t.Error("KeyOf should be deterministic")
	}
	k3 := KeyOf("dev1", []string{"/ip/address/print"})
	if k1 == k3 {
		t.Error("KeyOf should differ for different sentences")
	}
	// Device scoping: same sentence + different device → different key.
	k4 := KeyOf("dev2", []string{"/ip/address/print", "detail"})
	if k1 == k4 {
		t.Error("KeyOf should differ for different devices")
	}
}

func TestPathFromSentence(t *testing.T) {
	cases := []struct {
		sentence []string
		want     string
	}{
		{[]string{"/ip/address/print"}, "/ip/address"},
		{[]string{"/system/resource/print", "detail"}, "/system/resource"},
		{[]string{"/interface/monitor-traffic", "=interface=ether1"}, "/interface"},
		{[]string{"/beep"}, "/beep"},
		{[]string{}, ""},
	}
	for _, tc := range cases {
		if got := PathFromSentence(tc.sentence); got != tc.want {
			t.Errorf("PathFromSentence(%v) = %q, want %q", tc.sentence, got, tc.want)
		}
	}
}
