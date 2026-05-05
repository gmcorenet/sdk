# gmcore-session

Session management for gmcore applications with multiple store backends.

## Features

- **Multiple stores**: Memory, Redis (extensible)
- **Flash messages**: One-time notification messages
- **YAML configuration**: Configure sessions from YAML files
- **Cookie options**: Configurable cookie settings
- **Context helpers**: Store/retrieve sessions from context

## Configuration

### YAML Configuration

Create `config/session.yaml` in your app:

```yaml
name: myapp_session
lifetime: 3600

path: /
domain: ""
secure: true
http_only: true
same_site: strict
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
name: %env(SESSION_NAME)%
lifetime: 3600
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-session"

cfg, err := gmcore_session.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

config := gmcore_session.NewManagerConfig().
    WithStore(gmcore_session.NewMemoryStore()).
    WithName(cfg.Name).
    WithLifetime(cfg.Lifetime)

manager := config.Build()
```

## Usage

### Basic Session Management

```go
import "github.com/gmcorenet/sdk/gmcore-session"

// Create store and manager
store := gmcore_session.NewMemoryStore()
manager := gmcore_session.NewManager(store, "myapp", 3600*time.Second)

// Start session
session, err := manager.Start(w, r)
if err != nil {
    log.Fatal(err)
}

// Set values
session.Set("user_id", 123)
session.Set("username", "john")

// Get values
userID := session.Get("user_id")
username := session.Get("username")

// Check key exists
if session.Has("user_id") {
    // ...
}

// Remove value
session.Remove("username")

// Clear all values
session.Clear()

// Destroy session
session.Destroy()
```

### Flash Messages

```go
// Set flash message
session.Flash("Welcome back!")

// Get flash messages (one-time)
flashes := session.GetFlashes()
for _, msg := range flashes {
    fmt.Println(msg)
}
```

### HTTP Middleware Pattern

```go
func sessionMiddleware(next http.Handler) http.Handler {
    store := gmcore_session.NewMemoryStore()
    manager := gmcore_session.NewManager(store, "app", 3600*time.Second)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, err := manager.Start(w, r)
        if err != nil {
            http.Error(w, "Session error", 500)
            return
        }

        ctx := gmcore_session.SaveToContext(r.Context(), session)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Context Helpers

```go
// Save session to context
ctx := gmcore_session.SaveToContext(r.Context(), session)

// Retrieve session from context
session := gmcore_session.FromContext(ctx)
if session != nil {
    userID := session.Get("user_id")
}
```

## Store Interface

Implement your own store:

```go
type Store interface {
    New(sid string) (Session, error)
    Get(sid string) (Session, error)
    Save(Session) error
    Delete(sid string) error
}
```

### Memory Store

```go
store := gmcore_session.NewMemoryStore()
```

### Redis Store (example)

```go
type RedisStore struct {
    client *redis.Client
}

func (s *RedisStore) New(sid string) (Session, error) {
    // Create new session
}

func (s *RedisStore) Get(sid string) (Session, error) {
    // Get session from Redis
}
```

## Cookie Configuration

| Option    | Type     | Default   | Description                      |
|-----------|----------|-----------|----------------------------------|
| `name`     | `string` | -         | Cookie name                      |
| `path`     | `string` | `/`       | Cookie path                      |
| `domain`   | `string` | -         | Cookie domain                    |
| `secure`   | `bool`   | `true`    | Require HTTPS                    |
| `http_only` | `bool`   | `true`    | JavaScript inaccessible          |
| `same_site` | `string` | `strict`  | SameSite mode (strict/lax/none) |
| `lifetime` | `int`    | `3600`    | Session lifetime in seconds      |

### SameSite Options

- `strict` - Cookies only sent on same-site requests
- `lax` - Cookies sent on same-site and cross-site GET requests
- `none` - Cookies sent in all contexts (requires Secure)

## Session Interface

```go
type Session interface {
    ID() string
    Get(key string) interface{}
    Set(key string, value interface{})
    Remove(key string)
    Has(key string) bool
    Keys() []string
    Clear()
    Destroy()
    Flash(message string)
    GetFlashes() []string
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/gmcorenet/sdk/gmcore-session"
)

func main() {
    store := gmcore_session.NewMemoryStore()
    manager := gmcore_session.NewManager(store, "myapp", 3600*time.Second)

    http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        session, _ := manager.Start(w, r)
        session.Set("user", "john")
        session.Flash("Login successful!")
        fmt.Fprintf(w, "Logged in!")
    })

    http.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
        session, _ := manager.Start(w, r)

        flashes := session.GetFlashes()
        for _, f := range flashes {
            fmt.Fprintf(w, "Flash: %s\n", f)
        }

        if user := session.Get("user"); user != nil {
            fmt.Fprintf(w, "User: %s", user)
        } else {
            fmt.Fprintf(w, "Not logged in")
        }
    })

    http.ListenAndServe(":8080", nil)
}
```
