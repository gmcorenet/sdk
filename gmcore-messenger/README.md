# gmcore-messenger

Message bus and worker system for gmcore applications.

## Features

- **Message bus**: Dispatch messages to handlers
- **Async dispatch**: Non-blocking message sending
- **Worker system**: Process messages from transports
- **YAML configuration**: Configure messenger from YAML files
- **Retry policy**: Configurable retry with exponential backoff
- **Type-based routing**: Route messages by type to handlers

## Configuration

### YAML Configuration

Create `config/messenger.yaml` in your app:

```yaml
worker_count: 4

retry_policy:
  max_retries: 3
  initial_delay: 1000
  max_delay: 60000
  multiplier: 2.0

transport: memory
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML:

```yaml
worker_count: %env(MESSENGER_WORKERS)%
```

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-messenger"

cfg, err := gmcore_messenger.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}
```

## Usage

### Message Bus

```go
import "github.com/gmcorenet/sdk/gmcore-messenger"

// Create bus
bus := gmcore_messenger.NewBus()

// Register handlers
bus.Register(func(msg interface{}) error {
    fmt.Printf("Received: %v\n", msg)
    return nil
}, "*")

bus.Register(func(msg interface{}) error {
    if emailMsg, ok := msg.(EmailMessage); ok {
        fmt.Printf("Email to: %s\n", emailMsg.To)
    }
    return nil
}, "EmailMessage")

// Dispatch messages
bus.Dispatch("Hello")
bus.DispatchAsync(EmailMessage{To: "user@example.com", Subject: "Hi"})
```

### Workers

```go
// Create transport and bus
transport := gmcore_messenger.NewInMemoryTransport()
bus := gmcore_messenger.NewBus()

// Register handlers
bus.Register(func(msg interface{}) error {
    fmt.Printf("Processing: %v\n", msg)
    return nil
}, "*")

// Create and start worker
worker := gmcore_messenger.NewWorker(transport, bus)
worker.Start()

// Dispatch message to transport
transport.Send([]interface{}{"Hello from queue"})

// Stop worker
worker.Stop()
```

### Custom Message Types

```go
type OrderCreated struct {
    OrderID string
    Amount  float64
}

type UserRegistered struct {
    UserID string
    Email  string
}

bus.Register(func(msg interface{}) error {
    switch m := msg.(type) {
    case OrderCreated:
        return handleOrderCreated(m)
    case UserRegistered:
        return handleUserRegistered(m)
    }
    return nil
}, "*")

// Dispatch
bus.Dispatch(OrderCreated{OrderID: "123", Amount: 99.99})
bus.Dispatch(UserRegistered{UserID: "456", Email: "user@example.com"})
```

### Retry Policy

```go
policy := gmcore_messenger.DefaultRetryPolicy()
// MaxRetries: 3
// InitialDelay: 1000ms
// MaxDelay: 60000ms
// Multiplier: 2.0

// Custom policy
policy := gmcore_messenger.RetryPolicy{
    MaxRetries:   5,
    InitialDelay: 500,
    MaxDelay:     30000,
    Multiplier:   1.5,
}

// Calculate delay for retry attempt
delay := policy.NextDelay(0) // 500ms
delay = policy.NextDelay(1) // 750ms
delay = policy.NextDelay(2) // 1125ms
```

## Message Type Detection

Messages are routed based on their Go type name:

```go
type EmailMessage struct {
    To      string
    Subject string
}

bus.Register(handler, "EmailMessage")

msg := EmailMessage{To: "test@example.com"}
bus.Dispatch(msg)  // Goes to "EmailMessage" handlers
```

Unnamed types use fallback:

```go
bus.Dispatch("string message")  // Goes to "string" handlers
bus.Dispatch(123)              // Goes to "int" handlers
bus.Dispatch(nil)              // Goes to "nil" handlers
```

## Transport Interface

Implement your own transport:

```go
type Transport interface {
    Send(messages []interface{}) error
    Receive() (interface{}, error)
    Ack(message interface{}) error
    Reject(message interface{}) error
}
```

### In-Memory Transport

```go
transport := gmcore_messenger.NewInMemoryTransport()

// Send messages to queue
transport.Send([]interface{}{
    "message 1",
    "message 2",
})

// Receive from queue
msg, _ := transport.Receive()
// msg == "message 1"
```

## Configuration Options

| Option           | Type     | Default | Description                    |
|------------------|----------|---------|--------------------------------|
| `worker_count`    | `int`    | `1`     | Number of worker goroutines     |
| `retry.max_retries` | `int` | `3`     | Maximum retry attempts         |
| `retry.initial_delay` | `int` | `1000`  | Initial delay in milliseconds   |
| `retry.max_delay` | `int`    | `60000` | Maximum delay in milliseconds   |
| `retry.multiplier` | `float64` | `2.0` | Backoff multiplier             |
| `transport`       | `string` | `memory` | Transport backend             |

## Complete Example

```go
package main

import (
    "fmt"

    "github.com/gmcorenet/sdk/gmcore-messenger"
)

type OrderMessage struct {
    OrderID string
    Amount  float64
}

func main() {
    // Setup
    transport := gmcore_messenger.NewInMemoryTransport()
    bus := gmcore_messenger.NewBus()

    // Register handlers
    bus.Register(func(msg interface{}) error {
        if order, ok := msg.(OrderMessage); ok {
            fmt.Printf("Processing order: %s (%.2f)\n", order.OrderID, order.Amount)
        }
        return nil
    }, "OrderMessage")

    // Start worker
    worker := gmcore_messenger.NewWorker(transport, bus)
    worker.Start()

    // Dispatch via transport
    transport.Send([]interface{}{
        OrderMessage{OrderID: "123", Amount: 99.99},
        OrderMessage{OrderID: "456", Amount: 149.99},
    })

    // Or dispatch directly
    bus.Dispatch(OrderMessage{OrderID: "789", Amount: 49.99})

    // Cleanup
    worker.Stop()
}
```
