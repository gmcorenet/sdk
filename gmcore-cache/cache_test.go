package gmcore_cache

import (
	"testing"
	"time"
)

func TestItem_ExpiresAfter(t *testing.T) {
	i := newItem("key", "value")
	i.ExpiresAfter(time.Second)

	if i.IsExpired() {
		t.Error("item should not be expired immediately")
	}
}

func TestItem_IsExpired(t *testing.T) {
	i := newItem("key", "value")
	i.ExpiresAfter(-time.Second)

	if !i.IsExpired() {
		t.Error("item should be expired")
	}
}

func TestArrayPool_GetItem(t *testing.T) {
	pool := NewArrayPool()

	item := pool.GetItem("nonexistent")
	if item.IsHit() {
		t.Error("nonexistent key should not be a hit")
	}

	pool.Save(newItem("key", "value"))
	item = pool.GetItem("key")
	if !item.IsHit() {
		t.Error("existing key should be a hit")
	}
	if item.Get() != "value" {
		t.Error("item value should be 'value'")
	}
}

func TestArrayPool_HasItem(t *testing.T) {
	pool := NewArrayPool()

	if pool.HasItem("nonexistent") {
		t.Error("nonexistent key should not exist")
	}

	pool.Save(newItem("key", "value"))
	if !pool.HasItem("key") {
		t.Error("key should exist after save")
	}
}

func TestArrayPool_Clear(t *testing.T) {
	pool := NewArrayPool()
	pool.Save(newItem("key1", "value1"))
	pool.Save(newItem("key2", "value2"))

	pool.Clear()

	if pool.HasItem("key1") {
		t.Error("key1 should not exist after clear")
	}
	if pool.HasItem("key2") {
		t.Error("key2 should not exist after clear")
	}
}

func TestArrayPool_DeleteItem(t *testing.T) {
	pool := NewArrayPool()
	pool.Save(newItem("key", "value"))

	pool.DeleteItem("key")

	if pool.HasItem("key") {
		t.Error("key should not exist after delete")
	}
}

func TestMemoryAdapter(t *testing.T) {
	adapter := NewMemoryAdapter()

	adapter.Set("key1", "value1")
	adapter.Set("key2", 42)

	val, hit := adapter.Get("key1")
	if !hit {
		t.Error("key1 should be a hit")
	}
	if val != "value1" {
		t.Errorf("key1 value should be 'value1', got %v", val)
	}

	if !adapter.Has("key1") {
		t.Error("Has(key1) should be true")
	}

	adapter.Delete("key1")
	if adapter.Has("key1") {
		t.Error("key1 should not exist after delete")
	}

	adapter.Clear()
	if adapter.Has("key2") {
		t.Error("key2 should not exist after clear")
	}
}

func TestManager(t *testing.T) {
	pool := NewArrayPool()
	manager := NewManager(pool, "test_", 3600)

	manager.Set("key", "value")

	val, hit := manager.Get("key")
	if !hit {
		t.Error("key should be a hit")
	}
	if val != "value" {
		t.Errorf("key value should be 'value', got %v", val)
	}

	if !manager.Has("key") {
		t.Error("Has(key) should be true")
	}

	manager.Delete("key")
	if manager.Has("key") {
		t.Error("key should not exist after delete")
	}
}

func TestManager_TTL(t *testing.T) {
	pool := NewArrayPool()
	manager := NewManager(pool, "", 1)

	manager.Set("key", "value")

	_, hit := manager.Get("key")
	if !hit {
		t.Error("key should be a hit immediately")
	}

	time.Sleep(2 * time.Second)

	_, hit = manager.Get("key")
	if hit {
		t.Error("key should be expired after TTL")
	}
}

func TestManager_Clear(t *testing.T) {
	pool := NewArrayPool()
	manager := NewManager(pool, "", 0)

	manager.Set("key1", "value1")
	manager.Set("key2", "value2")

	manager.Clear()

	if manager.Has("key1") {
		t.Error("key1 should not exist after clear")
	}
}

func TestCreateManager(t *testing.T) {
	cfg := &Config{
		Adapter: "memory",
		Prefix:  "test_",
		TTL:     3600,
	}

	manager, err := CreateManager(cfg)
	if err != nil {
		t.Fatalf("CreateManager failed: %v", err)
	}

	manager.Set("key", "value")
	if !manager.Has("key") {
		t.Error("key should exist")
	}
}

func TestCreateManager_UnknownAdapter(t *testing.T) {
	cfg := &Config{
		Adapter: "unknown",
	}

	_, err := CreateManager(cfg)
	if err != ErrAdapterNotFound {
		t.Errorf("expected ErrAdapterNotFound, got %v", err)
	}
}

func TestRegisterAdapter(t *testing.T) {
	RegisterAdapter("custom", func(cfg *Config) (CacheManager, error) {
		return NewMemoryAdapter(), nil
	})

	cfg := &Config{
		Adapter: "custom",
	}

	manager, err := CreateManager(cfg)
	if err != nil {
		t.Fatalf("CreateManager failed: %v", err)
	}

	manager.Set("key", "value")
	if !manager.Has("key") {
		t.Error("key should exist")
	}
}
