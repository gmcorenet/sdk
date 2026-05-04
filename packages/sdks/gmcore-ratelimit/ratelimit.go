package gmcore_ratelimit

import (
	"sync"
	"time"
)

type Rule struct {
	Name      string
	Limit     int
	Window    time.Duration
	RawWindow string
}

type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
}

type limiter struct {
	tokens    map[string][]time.Time
	rate      int
	window    time.Duration
	mu        sync.Mutex
}

func NewRateLimiter(rate int, window time.Duration) *limiter {
	return &limiter{
		tokens: make(map[string][]time.Time),
		rate:   rate,
		window: window,
	}
}

func (l *limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.window)

	tokens := l.tokens[key]
	valid := make([]time.Time, 0)
	for _, t := range tokens {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= l.rate {
		l.tokens[key] = valid
		return false
	}

	valid = append(valid, now)
	l.tokens[key] = valid
	return true
}

func (l *limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.tokens, key)
}

func (l *limiter) GetRemaining(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.window)

	count := 0
	for _, t := range l.tokens[key] {
		if t.After(windowStart) {
			count++
		}
	}
	return l.rate - count
}

type SlidingWindowLimiter struct {
	limiter *limiter
}

func NewSlidingWindow(rate int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{limiter: NewRateLimiter(rate, window)}
}

func (l *SlidingWindowLimiter) Allow(key string) bool {
	return l.limiter.Allow(key)
}

func (l *SlidingWindowLimiter) Reset(key string) {
	l.limiter.Reset(key)
}

type TokenBucketLimiter struct {
	buckets map[string]*bucket
	rate    int
	size    int
	mu      sync.Mutex
}

type bucket struct {
	tokens    float64
	lastRefill time.Time
}

func NewTokenBucket(rate, size int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		size:    size,
	}
}

func (l *TokenBucketLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.size), lastRefill: time.Now()}
		l.buckets[key] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * float64(l.rate)
	if b.tokens > float64(l.size) {
		b.tokens = float64(l.size)
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

type FixedWindowLimiter struct {
	windows map[string]*fixedWindow
	rate    int
	window  time.Duration
	mu      sync.Mutex
}

type fixedWindow struct {
	count      int
	windowStart time.Time
}

func NewFixedWindow(rate int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		windows: make(map[string]*fixedWindow),
		rate:    rate,
		window:  window,
	}
}

func (l *FixedWindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Truncate(l.window)

	b, ok := l.windows[key]
	if !ok || b.windowStart.Before(windowStart) {
		l.windows[key] = &fixedWindow{count: 1, windowStart: windowStart}
		return true
	}

	if b.count >= l.rate {
		return false
	}

	b.count++
	return true
}

func (l *FixedWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
}
