# gmcore-router

HTTP router for gmcore applications with route matching, groups, and YAML configuration.

## Features

- **Route matching**: Pattern-based path matching with parameters
- **Route groups**: Group routes with shared prefix and naming
- **YAML configuration**: Load routes from YAML files
- **Named routes**: Generate URLs by route name
- **Method matching**: Support for HTTP methods (GET, POST, etc.)
- **Context params**: Access route parameters from request context

## Configuration

### YAML Configuration

Create `config/routes.yaml` in your app:

```yaml
routes:
  home:
    path: /
    handler: HomeController.Index
    methods: [GET]

  users_list:
    path: /users
    handler: UserController.List
    methods: [GET]

  user_show:
    path: /users/{id}
    handler: UserController.Show
    methods: [GET]

  user_create:
    path: /users
    handler: UserController.Create
    methods: [POST]
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
routes:
  api:
    path: %env(API_PREFIX)%/users
    handler: UserController.List
    methods: [GET]
```

### Loading Routes

```go
import "github.com/gmcorenet/sdk/gmcore-router"

router := gmcore_router.New()

// Register handlers
registry := gmcore_router.NewDefaultHandlerRegistry()
registry.Register("HomeController.Index", func() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "Hello, World!")
    }
})

// Load config and apply
cfg, err := gmcore_router.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}
cfg.ApplyTo(router, registry)
```

## Usage

### Basic Router

```go
router := gmcore_router.New()

router.Add("GET", "/", "home", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Home Page")
})

router.Add("GET", "/about", "about", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "About Page")
})

http.ListenAndServe(":8080", router)
```

### Route Parameters

```go
router.Add("GET", "/users/{id}", "user_show", func(w http.ResponseWriter, r *http.Request) {
    id := gmcore_router.Param(r, "id")
    fmt.Fprintf(w, "User: %s", id)
})

router.Add("GET", "/posts/{year}/{month}", "archive", func(w http.ResponseWriter, r *http.Request) {
    year := gmcore_router.Param(r, "year")
    month := gmcore_router.Param(r, "month")
    fmt.Fprintf(w, "Archive: %s/%s", year, month)
})
```

### Route Groups

```go
router := gmcore_router.New()

api := router.Group("/api", "api_")

api.Add("GET", "/users", "users_list", listUsers)
api.Add("GET", "/users/{id}", "users_show", showUser)
api.Add("POST", "/users", "users_create", createUser)

admin := router.Group("/admin", "admin_")
admin.Add("GET", "/dashboard", "dashboard", adminDashboard)
admin.Add("GET", "/settings", "settings", adminSettings)
```

### Named Routes and URL Generation

```go
router := gmcore_router.New()

router.Add("GET", "/users/{id}", "user_show", showUser)

// Generate URL
url := router.URL("user_show", map[string]string{"id": "123"})
// Returns: /users/123
```

### Not Found Handler

```go
router := gmcore_router.New()

router.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
    http.Error(w, "Page Not Found", http.StatusNotFound)
})
```

## Route Configuration

### Fields

| Field    | Type       | Description                        |
|----------|------------|------------------------------------|
| `name`   | `string`   | Unique route identifier            |
| `path`   | `string`   | URL path pattern                   |
| `handler`| `string`   | Handler identifier (e.g., Controller.Method) |
| `methods`| `[]string` | Allowed HTTP methods               |

### Path Parameters

Use `{param}` syntax in paths:

```yaml
routes:
  user:
    path: /users/{id}
    handler: UserController.Show
    methods: [GET]
```

### Multiple Methods

```yaml
routes:
  form:
    path: /contact
    handler: ContactController.Form
    methods: [GET, POST]
```

## Handler Registry

The router uses a handler registry to resolve handler names to actual HTTP handlers.

### Default Registry

```go
registry := gmcore_router.NewDefaultHandlerRegistry()

registry.Register("HomeController.Index", func() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ...
    }
})
```

### Custom Registry

Implement the `HandlerRegistry` interface:

```go
type HandlerRegistry interface {
    Get(name string) HandlerInfo
}

type HandlerInfo struct {
    Controller string
    Method    string
}
```

## Accessing Route Parameters

```go
func showUser(w http.ResponseWriter, r *http.Request) {
    id := gmcore_router.Param(r, "id")
    // ...
}
```

## Route Matching

Routes are matched in order of addition. The first matching route handles the request.

### Matching Rules

- Exact path match takes precedence
- Parameter patterns (`{id}`) match any segment
- Trailing slashes are handled consistently
- HEAD requests match GET routes

## Complete Example

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/gmcorenet/sdk/gmcore-router"
)

func main() {
    router := gmcore_router.New()

    // Create handler registry
    registry := gmcore_router.NewDefaultHandlerRegistry()

    // Register handlers
    registry.Register("HomeController.Index", func() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            fmt.Fprint(w, "Welcome Home!")
        }
    })

    registry.Register("UserController.Show", func() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            id := gmcore_router.Param(r, "id")
            fmt.Fprintf(w, "User ID: %s", id)
        }
    })

    // Load routes from YAML
    cfg, err := gmcore_router.LoadConfig("/opt/gmcore/myapp")
    if err == nil {
        cfg.ApplyTo(router, registry)
    }

    http.ListenAndServe(":8080", router)
}
```
