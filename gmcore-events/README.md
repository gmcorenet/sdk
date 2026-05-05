# gmcore-events

Event bus and dispatcher for gmcore applications.

## Features

- **Event bus**: Publish/subscribe event system
- **Wildcard listeners**: Listen to all events with "*"
- **Once listeners**: Subscribe for single execution
- **YAML configuration**: Configure listeners from YAML files
- **Handler registry**: Manage event handlers

## Configuration

### YAML Configuration

Create `config/events.yaml` in your app:

```yaml
listeners:
  user.created:
    - type: handler
      handler: onUserCreated
      async: false

  order.completed:
    - type: handler
      handler: onOrderCompleted
      async: true
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-events"

cfg, err := gmcore_events.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

bus := gmcore_events.NewBus()
registry := gmcore_events.NewDefaultHandlerRegistry()

// Register handlers
registry.Register("onUserCreated", func(ctx context.Context, event interface{}) error {
    fmt.Println("User created:", event)
    return nil
})

// Apply config
cfg.ApplyTo(bus, registry)
```

## Usage

### Basic Event Bus

```go
import "github.com/gmcorenet/sdk/gmcore-events"

bus := gmcore_events.NewBus()

// Subscribe to event
unsubscribe := bus.Subscribe("user.created", func(ctx context.Context, event interface{}) error {
    fmt.Printf("User created: %v\n", event)
    return nil
})

// Dispatch event
bus.Dispatch(context.Background(), "user.created", map[string]interface{}{
    "id":    123,
    "email": "user@example.com",
})

// Unsubscribe
unsubscribe()
```

### Wildcard Listener

```go
// Listen to all events
bus.Subscribe("*", func(ctx context.Context, event interface{}) error {
    fmt.Printf("Event received: %v\n", event)
    return nil
})
```

### Once Listener

```go
// Executes only once, then auto-unsubscribes
bus.SubscribeOnce("app.ready", func(ctx context.Context, event interface{}) error {
    fmt.Println("App is ready!")
    return nil
})
```

### Error Handling

```go
// Collect all errors from listeners
errs := bus.DispatchCollect(ctx, "order.created", order)

if len(errs) > 0 {
    for _, err := range errs {
        fmt.Printf("Listener error: %v\n", err)
    }
}
```

### Unsubscribe All

```go
bus.UnsubscribeAll("user.created")
```

## Event Handler Registry

```go
registry := gmcore_events.NewDefaultHandlerRegistry()

registry.Register("onUserCreated", func(ctx context.Context, event interface{}) error {
    // Handle event
    return nil
})

handler := registry.Get("onUserCreated")
if handler != nil {
    handler.Handle(ctx, event)
}
```

## Configuration Options

| Event | Type     | Description                       |
|-------|----------|----------------------------------|
| `type` | `string` | Handler type (always "handler")  |
| `handler` | `string` | Handler name in registry         |
| `async` | `bool` | Run handler asynchronously        |

## Complete Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/gmcorenet/sdk/gmcore-events"
)

func main() {
    // Create bus and registry
    bus := gmcore_events.NewBus()
    registry := gmcore_events.NewDefaultHandlerRegistry()

    // Register handlers
    registry.Register("onUserCreated", func(ctx context.Context, event interface{}) error {
        fmt.Printf("User created: %v\n", event)
        return nil
    })

    registry.Register("onEmailSent", func(ctx context.Context, event interface{}) error {
        fmt.Printf("Email sent: %v\n", event)
        return nil
    })

    // Subscribe to events
    bus.Subscribe("user.created", func(ctx context.Context, event interface{}) error {
        data := event.(map[string]interface{})
        fmt.Printf("New user: %s\n", data["email"])
        return nil
    })

    // Dispatch events
    bus.Dispatch(context.Background(), "user.created", map[string]interface{}{
        "id":    1,
        "email": "user@example.com",
    })

    bus.Dispatch(context.Background(), "email.sent", "Welcome email sent")
}
```
