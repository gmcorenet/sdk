# gmcore-cache

Caching library for gmcore applications with multiple backend support.

## Features

- **Multiple backends**: Memory, Redis (extensible)
- **TTL support**: Automatic expiration of cached items
- **Key prefixing**: Avoid collisions between apps
- **YAML configuration**: Configure cache from YAML files
- **Thread-safe**: Safe for concurrent use
- **Pool chaining**: Chain multiple cache pools

## Configuration

### YAML Configuration

Create `config/cache.yaml` in your app:

```yaml
adapter: memory  # memory, redis, etc.
ttl: 3600       # Time to live in seconds (default: 3600)
prefix: myapp_  # Key prefix to avoid collisions

# Adapter-specific params
params:
  redis_url: %env(REDIS_URL)%
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
adapter: redis
params:
  redis_url: %env(REDIS_URL)%
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-cache"

cfg, err := gmcore_cache.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

manager, err := gmcore_cache.CreateManager(cfg)
if err != nil {
    log.Fatal(err)
}
```

## Usage

### Basic Operations

```go
import "github.com/gmcorenet/sdk/gmcore-cache"

manager, _ := gmcore_cache.CreateManager(&gmcore_cache.Config{
    Adapter: "memory",
    TTL:     3600,
    Prefix:  "myapp_",
})

// Set a value
manager.Set("user:123", map[string]any{"name": "John", "email": "john@example.com"})

// Get a value
user, found := manager.Get("user:123")
if found {
    fmt.Println(user)
}

// Check if key exists
exists := manager.Has("user:123")

// Delete a key
manager.Delete("user:123")

// Clear all
manager.Clear()
```

### Cache Item with Expiration

```go
// Create item with expiration
item := &gmcore_cache.Item{}
item.SetKey("temp:456").
    SetValue("temporary data").
    ExpiresAfter(5 * time.Minute)

manager.Save(item)
```

### Chaining Pools

```go
// Create a chain: L1 (Memory) -> L2 (Redis)
l1 := gmcore_cache.NewArrayPool()
l2 := redisPool // your redis pool

chain := gmcore_cache.NewChainPool(l1, l2)
manager := gmcore_cache.NewManager(chain, "app_", 3600)
```

### Registering Custom Adapters

```go
import "github.com/gmcorenet/sdk/gmcore-cache"

gmcore_cache.RegisterAdapter("myredis", func(cfg *gmcore_cache.Config) (gmcore_cache.CacheManager, error) {
    // Your Redis adapter implementation
    return myRedisManager, nil
})
```

## Adapters

### Memory Adapter

In-memory cache using map with TTL support.

```yaml
adapter: memory
ttl: 3600
prefix: app_
```

**Pros**: Fast, no external dependencies
**Cons**: Not shared between processes, data lost on restart

### Redis Adapter

Redis-backed cache for distributed caching.

```yaml
adapter: redis
ttl: 3600
prefix: app_
params:
  redis_url: redis://localhost:6379/0
```

**Pros**: Shared between processes, persistent
**Cons**: Requires Redis server

## Cache Manager Interface

```go
type CacheManager interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}) error
    Delete(key string) error
    Clear() error
    Has(key string) bool
}
```

## Configuration Options

| Option   | Type     | Default   | Description                  |
|----------|----------|-----------|------------------------------|
| `adapter`| `string` | `memory`  | Cache backend to use         |
| `ttl`    | `int`    | `3600`    | Default TTL in seconds       |
| `prefix` | `string` | `cache_`  | Key prefix for namespacing   |

## Key Prefixing

Prefix prevents collisions between different apps:

```go
manager := gmcore_cache.NewManager(pool, "myapp_", 3600)

manager.Set("user:123", data)  // Actually stores "myapp_user:123"
manager.Get("user:123")        // Actually gets "myapp_user:123"
```

## TTL (Time To Live)

Set expiration time for cached items:

```go
// Default TTL from config
manager.Set("key", value)  // Expires after configured TTL

// Custom expiration
item := newItem("key", value)
item.ExpiresAfter(30 * time.Minute)
pool.Save(item)
```

## Default Logger

The package provides a default pool for convenience:

```go
pool := gmcore_cache.NewArrayPool()
pool.Save(item)
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/gmcorenet/sdk/gmcore-cache"
)

func main() {
    // Load configuration
    cfg, err := gmcore_cache.LoadConfig("/opt/gmcore/myapp")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create cache manager
    manager, err := gmcore_cache.CreateManager(cfg)
    if err != nil {
        log.Fatalf("Failed to create manager: %v", err)
    }

    // Use cache
    manager.Set("key", "value")
    val, found := manager.Get("key")
    if found {
        fmt.Printf("Found: %v\n", val)
    }
}
```
