# gmcore-log

Structured logging SDK for GMCore applications.

## Features

- **Multiple levels** - Debug, Info, Warn, Error, Fatal
- **Multiple handlers** - Console, File, Rotating File, Syslog
- **Multiple formatters** - Text, JSON
- **Structured fields** - Add context to log entries
- **Thread-safe** - Safe for concurrent use
- **Global logger** - Package-level convenience functions

## Installation

```bash
go get github.com/gmcorenet/gmcore-log
```

## Quick Start

```go
import "github.com/gmcorenet/gmcore-log"

// Simple usage
log.Info("Application started")
log.Error("Something went wrong", "error", err)

// With fields
log.WithField("user_id", 123).Info("User logged in")

// Multiple fields
log.WithFields(map[string]interface{}{
    "request_id": "abc123",
    "method":     "GET",
}).Warn("Slow request")
```

## Log Levels

```go
log.SetLevel(log.LevelDebug) // Set minimum level
```

| Level | Description |
|-------|-------------|
| `LevelDebug` | Debug information |
| `LevelInfo` | General information |
| `LevelWarn` | Warning messages |
| `LevelError` | Error messages |
| `LevelFatal` | Fatal errors (exits program) |

## Handlers

### Console Handler

```go
handler := log.NewConsoleHandler(os.Stdout)
logger.AddHandler(handler)
```

### File Handler

```go
handler, err := log.NewFileHandler("/var/log/app.log")
if err != nil {
    log.Fatal("Failed to open log file: %v", err)
}
logger.AddHandler(handler)
```

### Rotating File Handler

```go
handler, err := log.NewRotatingFileHandler(
    "/var/log/app.log",
    10*1024*1024, // 10MB max size
    5,            // 5 backup files
)
logger.AddHandler(handler)
```

## Formatters

### Text Format (default)

```
2024-01-15T10:30:00Z [INFO] Application started {version=1.0.0}
```

### JSON Format

```go
handler := &log.ConsoleHandler{
    Writer: os.Stdout,
    Format: log.JSONFormat{},
}
```

```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","message":"Application started","version":"1.0.0"}
```

## Logger Instance

```go
logger := log.New()
logger.SetLevel(log.LevelDebug)
logger.AddHandler(handler)

logger.Info("Custom logger message")
```

## Global Functions

For convenience, package-level functions use a default logger:

```go
log.SetLevel(log.LevelInfo)
log.AddHandler(handler)

log.Info("Using default logger")
```

## License

MIT
