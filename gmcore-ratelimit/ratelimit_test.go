package gmcore_ratelimit

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	if rl.rate != 10 {
		t.Fatalf("expected rate 10, got %d", rl.rate)
	}
	if rl.window != time.Minute {
		t.Fatalf("expected window 1m, got %v", rl.window)
	}
	if rl.tokens == nil {
		t.Fatal("tokens map should be initialized")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)

	for i := 0; i < 5; i++ {
		if !rl.Allow("key1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	if rl.Allow("key1") {
		t.Fatal("6th request should be denied")
	}

	if !rl.Allow("key2") {
		t.Fatal("different key should be allowed")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	rl.Allow("key1")
	rl.Allow("key1")
	if rl.Allow("key1") {
		t.Fatal("3rd request should be denied")
	}

	rl.Reset("key1")

	if !rl.Allow("key1") {
		t.Fatal("after reset, request should be allowed again")
	}
}

func TestRateLimiter_GetRemaining(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	if rl.GetRemaining("key1") != 3 {
		t.Fatalf("expected 3 remaining, got %d", rl.GetRemaining("key1"))
	}

	rl.Allow("key1")
	if rl.GetRemaining("key1") != 2 {
		t.Fatalf("expected 2 remaining, got %d", rl.GetRemaining("key1"))
	}

	rl.Allow("key1")
	rl.Allow("key1")
	if rl.GetRemaining("key1") != 0 {
		t.Fatalf("expected 0 remaining, got %d", rl.GetRemaining("key1"))
	}
}

func TestRule(t *testing.T) {
	r := Rule{
		Name:      "login",
		Limit:     5,
		Window:    time.Minute,
		RawWindow: "60s",
	}
	if r.Name != "login" {
		t.Fatalf("unexpected name: %s", r.Name)
	}
	if r.Limit != 5 {
		t.Fatalf("unexpected limit: %d", r.Limit)
	}
}

func TestNewSlidingWindow(t *testing.T) {
	sw := NewSlidingWindow(10, time.Minute)
	if sw == nil {
		t.Fatal("NewSlidingWindow returned nil")
	}

	if !sw.Allow("key") {
		t.Fatal("first request should be allowed")
	}
}

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 20)
	if tb == nil {
		t.Fatal("NewTokenBucket returned nil")
	}
	if tb.rate != 10 {
		t.Fatalf("expected rate 10, got %d", tb.rate)
	}
	if tb.size != 20 {
		t.Fatalf("expected size 20, got %d", tb.size)
	}

	if !tb.Allow("key") {
		t.Fatal("first request should be allowed")
	}
}

func TestTokenBucket_Exhausted(t *testing.T) {
	tb := NewTokenBucket(1, 5)

	for i := 0; i < 5; i++ {
		if !tb.Allow("key") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	if tb.Allow("key") {
		t.Fatal("bucket should be exhausted")
	}
}

func TestNewFixedWindow(t *testing.T) {
	fw := NewFixedWindow(5, time.Minute)
	if fw == nil {
		t.Fatal("NewFixedWindow returned nil")
	}
	if fw.rate != 5 {
		t.Fatalf("expected rate 5, got %d", fw.rate)
	}

	if !fw.Allow("key") {
		t.Fatal("first request should be allowed")
	}
}

func TestFixedWindow_Exhausted(t *testing.T) {
	fw := NewFixedWindow(3, time.Hour)

	for i := 0; i < 3; i++ {
		if !fw.Allow("key") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	if fw.Allow("key") {
		t.Fatal("4th request should be denied")
	}
}

func TestFixedWindow_DifferentKeys(t *testing.T) {
	fw := NewFixedWindow(2, time.Hour)

	fw.Allow("key1")
	fw.Allow("key1")

	if fw.Allow("key1") {
		t.Fatal("key1 should be denied")
	}
	if !fw.Allow("key2") {
		t.Fatal("key2 should be allowed")
	}
}

func TestFixedWindow_Reset(t *testing.T) {
	fw := NewFixedWindow(2, time.Hour)

	fw.Allow("key")
	fw.Allow("key")
	fw.Reset("key")

	if !fw.Allow("key") {
		t.Fatal("after reset, key should be allowed")
	}
}

func TestSlidingWindow_Reset(t *testing.T) {
	sw := NewSlidingWindow(2, time.Hour)

	sw.Allow("key")
	sw.Allow("key")
	sw.Reset("key")

	if !sw.Allow("key") {
		t.Fatal("after reset, key should be allowed")
	}
}

func TestTokenBucket_Reset(t *testing.T) {
	tb := NewTokenBucket(1, 3)

	tb.Allow("key")
	tb.Allow("key")
	tb.Reset("key")

	if !tb.Allow("key") {
		t.Fatal("after reset, key should be allowed")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(100, 2)

	tb.Allow("key")
	time.Sleep(20 * time.Millisecond)

	if !tb.Allow("key") {
		t.Fatal("after refill interval, request should be allowed")
	}
}
