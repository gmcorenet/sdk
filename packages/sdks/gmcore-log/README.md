# Logger SDK

Sistema de logging estructurado con múltiples handlers y formateadores.

## Niveles de Log

```go
const (
    LevelDebug Level = iota  // 0
    LevelInfo                // 1
    LevelWarn                // 2
    LevelError               // 3
    LevelFatal               // 4
)
```

## Logger Principal

```go
logger := log.New()
logger.SetLevel(log.LevelInfo)
logger.AddHandler(handler)
```

### Métodos

- `SetLevel(level Level)` - Establece el nivel mínimo
- `AddHandler(h Handler)` - Añade un handler
- `WithField(key string, value interface{}) *Logger` - Crea logger con campo
- `WithFields(fields map[string]interface{}) *Logger` - Crea logger con múltiples campos

### Logging

```go
logger.Debug("debug message %v", arg)
logger.Info("info message")
logger.Warn("warning message")
logger.Error("error message")
logger.Fatal("fatal message") // Exit(1) después de loggear
```

## Handlers

### ConsoleHandler
```go
h := log.NewConsoleHandler(os.Stdout)
h.Format = log.TextFormat{} // o log.JSONFormat{}
```

### FileHandler
```go
h, err := log.NewFileHandler("/path/to/logfile")
h.Format = log.TextFormat{}
```

### RotatingFileHandler
```go
h, err := log.NewRotatingFileHandler("/path/to/log", maxSize, maxBackups)
```

### SyslogHandler
```go
h, _ := log.NewSyslogHandler()
```

## Formatos

### TextFormat
```
2024-01-01T12:00:00Z [INFO] message {field1=value1, field2=value2}
```

### JSONFormat
```json
{"time":"2024-01-01T12:00:00Z","level":"INFO","message":"message","field1":"value1"}
```

## Funciones Globales

```go
log.SetLevel(log.LevelDebug)
log.AddHandler(handler)
log.Info("message")
log.WithField("key", "value").Info("message")
```
