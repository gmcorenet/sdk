# gmcore-log

Structured logging library for gmcore applications.

## Features

- **Multiple handlers**: Console, File, Rotating File, Syslog
- **Structured fields**: Add context to log entries
- **Log levels**: Debug, Info, Warn, Error, Fatal
- **YAML configuration**: Configure logger from YAML files
- **JSON and Text formats**: Choose output format
- **Thread-safe**: Safe for concurrent use

## Configuration

### YAML Configuration

Create `config/log.yaml` in your app:

```yaml
level: info

handlers:
  - type: console
    params:
      format: text  # text or json

  - type: rotating
    params:
      filename: var/log/app.log
      max_size: 10485760    # 10MB
      max_backups: 5
      format: json
```

### Environment Variables

Use `%env(VAR_NAME)%` syntax in YAML to inject environment variables:

```yaml
handlers:
  - type: file
    params:
      filename: %env(LOG_FILE_PATH)%
```

Supported env files:
- `.env`
- `.env.local`
- `config/<appname>.env`

### Loading Config

```go
import "github.com/gmcorenet/sdk/gmcore-log"

cfg, err := gmcore_log.LoadConfig("/opt/gmcore/myapp")
if err != nil {
    log.Fatal(err)
}

logger, err := cfg.Build()
if err != nil {
    log.Fatal(err)
}

// Use as default logger
gmcore_log.SetLevel(gmcore_log.LevelInfo)
gmcore_log.AddHandler(handler)
```

## Usage

### Basic Logging

```go
import "github.com/gmcorenet/sdk/gmcore-log"

log := gmcore_log.New()
log.Info("Hello, %s!", "world")
log.WithField("user_id", 123).Info("User logged in")
log.WithFields(map[string]any{"order_id": 456, "total": 99.99}).Info("Order placed")
```

### Using Default Logger

```go
gmcore_log.Info("Info message")
gmcore_log.Warn("Warning message")
gmcore_log.Error("Error message")
gmcore_log.Debug("Debug message")
gmcore_log.Fatal("Fatal error")  // Exits after logging
```

### Log Levels

| Level  | Value | Description           |
|--------|-------|-----------------------|
| DEBUG  | 0     | Detailed debug info   |
| INFO   | 1     | General information   |
| WARN   | 2     | Warning conditions    |
| ERROR  | 3     | Error conditions      |
| FATAL  | 4     | Critical errors       |

### Handlers

#### Console Handler

```go
handler := gmcore_log.NewConsoleHandler(os.Stdout)
log.AddHandler(handler)
```

#### File Handler

```go
handler, err := gmcore_log.NewFileHandler("var/log/app.log")
if err != nil {
    log.Fatal(err)
}
log.AddHandler(handler)
```

#### Rotating File Handler

```go
handler, err := gmcore_log.NewRotatingFileHandler(
    "var/log/app.log",
    10*1024*1024,  // 10MB max size
    5,              // 5 backup files
)
if err != nil {
    log.Fatal(err)
}
log.AddHandler(handler)
```

#### Syslog Handler

```go
handler, err := gmcore_log.NewSyslogHandler()
if err != nil {
    log.Fatal(err)
}
log.AddHandler(handler)
```

### Formats

#### Text Format (default)

```
2024-01-15T10:30:00Z [INFO] Hello, world! {user_id=123}
```

#### JSON Format

```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","message":"Hello, world!","user_id":123}
```

## Configuration Options

### Handler Types

| Type      | Description                        |
|-----------|------------------------------------|
| `console` | Log to stdout/stderr              |
| `file`    | Log to single file                |
| `rotating`| Log to rotating files             |
| `syslog`  | Log to syslog                     |

### Handler Parameters

#### Console Handler

```yaml
handlers:
  - type: console
    params:
      format: text  # text or json
```

#### File Handler

```yaml
handlers:
  - type: file
    params:
      filename: var/log/app.log
      format: text  # text or json
```

#### Rotating Handler

```yaml
handlers:
  - type: rotating
    params:
      filename: var/log/app.log
      max_size: 10485760    # bytes
      max_backups: 5
      format: json
```

#### Syslog Handler

```yaml
handlers:
  - type: syslog
    params:
      facility: 1  # LOG_USER
      format: text
```

## Directory Structure

```
var/
└── log/
    ├── app.log       # Current log file
    ├── app.log.1     # Rotated file 1
    ├── app.log.2     # Rotated file 2
    └── app.log.3     # Rotated file 3
```

## Default Logger

The package provides a default logger for convenience:

```go
gmcore_log.SetLevel(gmcore_log.LevelInfo)
gmcore_log.AddHandler(handler)

gmcore_log.Info("Using default logger")
```
