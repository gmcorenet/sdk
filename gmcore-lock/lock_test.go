package gmcore_lock

import (
	"context"
	"testing"
	"time"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory(10 * time.Second)
	if factory == nil {
		t.Fatal("NewFactory returned nil")
	}
	if factory.lifetime != 10*time.Second {
		t.Fatalf("expected lifetime 10s, got %v", factory.lifetime)
	}
}

func TestFactory_CreateLock(t *testing.T) {
	factory := NewFactory(time.Minute)

	l := factory.CreateLock("resource-1")
	if l == nil {
		t.Fatal("CreateLock returned nil")
	}

	l2 := factory.CreateLock("resource-1")
	if l != l2 {
		t.Fatal("CreateLock should return the same lock for the same resource")
	}

	l3 := factory.CreateLock("resource-2")
	if l == l3 {
		t.Fatal("CreateLock should return different locks for different resources")
	}
}

func TestLock_Acquire(t *testing.T) {
	factory := NewFactory(time.Minute)
	l := factory.CreateLock("acquire-test")

	acquired := l.Acquire(context.Background())
	if !acquired {
		t.Fatal("first Acquire should succeed")
	}

	acquired = l.Acquire(context.Background())
	if acquired {
		t.Fatal("second Acquire should fail when already held")
	}
}

func TestLock_Release(t *testing.T) {
	factory := NewFactory(time.Minute)
	l := factory.CreateLock("release-test")

	err := l.Release()
	if err != ErrLockNotAcquired {
		t.Fatalf("expected ErrLockNotAcquired, got %v", err)
	}

	l.Acquire(context.Background())
	err = l.Release()
	if err != nil {
		t.Fatalf("Release after Acquire should succeed: %v", err)
	}

	err = l.Release()
	if err != ErrLockNotAcquired {
		t.Fatalf("expected ErrLockNotAcquired after double release, got %v", err)
	}
}

func TestLock_Extend(t *testing.T) {
	factory := NewFactory(time.Minute)
	l := factory.CreateLock("extend-test")

	if l.Extend(time.Hour) {
		t.Fatal("Extend should fail if not acquired")
	}

	l.Acquire(context.Background())
	if !l.Extend(time.Hour) {
		t.Fatal("Extend should succeed after Acquire")
	}
}

func TestSemaphoreLock_Acquire(t *testing.T) {
	sl := NewSemaphoreLock("test-resource", 2)

	acquired := sl.Acquire(context.Background())
	if !acquired {
		t.Fatal("first Acquire should succeed")
	}

	acquired = sl.Acquire(context.Background())
	if !acquired {
		t.Fatal("second Acquire should succeed (maxSlots=2)")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	acquired = sl.Acquire(ctx)
	if acquired {
		t.Fatal("Acquire on cancelled context should fail")
	}
}

func TestSemaphoreLock_Release(t *testing.T) {
	sl := NewSemaphoreLock("test-resource", 2)

	sl.Acquire(context.Background())
	err := sl.Release()
	if err != nil {
		t.Fatalf("Release should succeed after Acquire: %v", err)
	}

	err = sl.Release()
	if err == nil {
		t.Fatal("Release on empty semaphore should fail")
	}
}

func TestSemaphoreLock_Extend(t *testing.T) {
	sl := NewSemaphoreLock("test-resource", 1)
	if !sl.Extend(time.Hour) {
		t.Fatal("SemaphoreLock Extend should always return true")
	}
}

func TestNewSemaphoreLock_DefaultMaxSlots(t *testing.T) {
	sl := NewSemaphoreLock("test", 0)
	if sl.maxSlots != 1 {
		t.Fatalf("expected maxSlots=1 for value <= 0, got %d", sl.maxSlots)
	}
}

func TestRedisLock_WithoutClient(t *testing.T) {
	rl := NewRedisLock("resource", time.Minute)
	acquired := rl.Acquire(context.Background())
	if acquired {
		t.Fatal("RedisLock Acquire should fail without client")
	}
}

func TestRedisLock_Release_WithoutClient(t *testing.T) {
	rl := NewRedisLock("resource", time.Minute)
	err := rl.Release()
	if err != ErrRedisNotConfigured {
		t.Fatalf("expected ErrRedisNotConfigured, got %v", err)
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrLockNotAcquired.Error() != "lock not acquired" {
		t.Fatal("unexpected ErrLockNotAcquired message")
	}
	if ErrLockNotHeld.Error() != "lock not held by this owner" {
		t.Fatal("unexpected ErrLockNotHeld message")
	}
	if ErrRedisNotConfigured.Error() != "redis client not configured: use WithRedisClient option" {
		t.Fatal("unexpected ErrRedisNotConfigured message")
	}
}

func TestGenerateOwner(t *testing.T) {
	owner := generateOwner()
	if owner == "" {
		t.Fatal("generateOwner should return non-empty string")
	}
	if len(owner) < 10 {
		t.Fatal("generateOwner should return a reasonably long string")
	}
}

func TestFactory_ConcurrentAccess(t *testing.T) {
	factory := NewFactory(time.Minute)
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			l := factory.CreateLock("shared-resource")
			l.Acquire(context.Background())
			l.Release()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
