package gmcore_lock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrLockNotAcquired = errors.New("lock not acquired")
	ErrLockNotHeld     = errors.New("lock not held by this owner")
	ErrRedisNotConfigured = errors.New("redis client not configured: use WithRedisClient option")
)

type Lock interface {
	Acquire(ctx context.Context) bool
	Release() error
	Extend(lifetime time.Duration) bool
}

type lock struct {
	resource string
	lifetime time.Duration
	mu       sync.Mutex
	owner    string
	acquired bool
}

func newLock(resource string, lifetime time.Duration) *lock {
	return &lock{resource: resource, lifetime: lifetime}
}

func (l *lock) Acquire(ctx context.Context) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.acquired && l.owner != "" {
		return false
	}

	l.owner = generateOwner()
	l.acquired = true
	return true
}

func (l *lock) Release() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.acquired {
		return ErrLockNotAcquired
	}

	l.acquired = false
	l.owner = ""
	return nil
}

func (l *lock) Extend(lifetime time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.acquired {
		return false
	}

	l.lifetime = lifetime
	return true
}

type Factory interface {
	CreateLock(resource string) Lock
}

type lockFactory struct {
	locks    map[string]*lock
	lifetime time.Duration
	mu       sync.RWMutex
}

func NewFactory(lifetime time.Duration) *lockFactory {
	return &lockFactory{
		locks:    make(map[string]*lock),
		lifetime: lifetime,
	}
}

func (f *lockFactory) CreateLock(resource string) Lock {
	f.mu.Lock()
	defer f.mu.Unlock()

	if l, ok := f.locks[resource]; ok {
		return l
	}

	l := newLock(resource, f.lifetime)
	f.locks[resource] = l
	return l
}

type SemaphoreLock struct {
	resource string
	maxSlots int
	slots    chan struct{}
	mu       sync.Mutex
}

func NewSemaphoreLock(resource string, maxSlots int) *SemaphoreLock {
	if maxSlots <= 0 {
		maxSlots = 1
	}
	return &SemaphoreLock{
		resource: resource,
		maxSlots: maxSlots,
		slots:    make(chan struct{}, maxSlots),
	}
}

func (l *SemaphoreLock) Acquire(ctx context.Context) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case l.slots <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	default:
		return false
	}
}

func (l *SemaphoreLock) Release() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case <-l.slots:
		return nil
	default:
		return errors.New("semaphore slot not acquired")
	}
}

func (l *SemaphoreLock) Extend(_ time.Duration) bool {
	return true
}

type RedisClient interface {
	Do(ctx context.Context, cmd string, args ...interface{}) (interface{}, error)
}

type RedisLock struct {
	resource string
	owner    string
	lifetime time.Duration
	client   RedisClient
	mu       sync.Mutex
}

type RedisLockOption func(*RedisLock)

func WithRedisClient(client RedisClient) RedisLockOption {
	return func(l *RedisLock) {
		l.client = client
	}
}

func NewRedisLock(resource string, lifetime time.Duration, opts ...RedisLockOption) *RedisLock {
	l := &RedisLock{
		resource: resource,
		lifetime: lifetime,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *RedisLock) Acquire(ctx context.Context) bool {
	if l.client == nil {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	key := l.resource
	owner := generateOwner()

	result, err := l.client.Do(ctx, "SET", key, owner, "PX", l.lifetime.Milliseconds(), "NX")
	if err != nil || result == nil {
		return false
	}

	l.owner = owner
	return true
}

func (l *RedisLock) Release() error {
	if l.client == nil {
		return ErrRedisNotConfigured
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.owner == "" {
		return nil
	}

	key := l.resource
	owner := l.owner

	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0
	`

	// TODO: Accept context parameter in future API version
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := l.client.Do(ctx, "EVAL", script, 1, key, owner)
	if err != nil {
		return fmt.Errorf("failed to release redis lock: %w", err)
	}
	l.owner = ""
	return nil
}

func (l *RedisLock) Extend(lifetime time.Duration) bool {
	if l.client == nil {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.owner == "" {
		return false
	}

	key := l.resource

	// TODO: Accept context parameter in future API version
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := l.client.Do(ctx, "PEXPIRE", key, lifetime.Milliseconds())
	if err != nil {
		return false
	}

	return result == int64(1)
}

func generateOwner() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
