# gmcore-error

Error handling and panic recovery SDK for GMCore applications.

## Features

- **Error codes** - Standardized error codes for different error types
- **Error wrapping** - Context-rich error wrapping with stack traces
- **Panic recovery** - Graceful panic handling for CLI and web applications
- **Exit codes** - Standard UNIX exit codes for CLI applications
- **User vs Technical messages** - User-friendly messages with technical details available
- **Stack traces** - Automatic stack trace capture

## Installation

```bash
go get github.com/gmcorenet/gmcore-error
```

## Quick Start

```go
import "github.com/gmcorenet/gmcore-error"

// Create a new error
err := error.New(error.CodeNotFound, "User not found")
err = error.Wrap(underlyingErr, error.CodeInternal, "Failed to fetch user")

// Get error details
if ge, ok := error.AsGmcoreError(err); ok {
    fmt.Println(ge.Code())
    fmt.Println(ge.Technical())
    fmt.Println(ge.StackTrace())
    ge.Exit() // Exit with appropriate code
}
```

## Error Codes

| Code | Description |
|------|-------------|
| `CodeUnknown` | Unknown error |
| `CodeNotFound` | Resource not found |
| `CodeInvalidInput` | Invalid input provided |
| `CodeUnauthorized` | Authentication required |
| `CodeForbidden` | Permission denied |
| `CodeInternal` | Internal server error |
| `CodeNetwork` | Network error |
| `CodeTimeout` | Operation timed out |
| `CodeNotImplemented` | Feature not implemented |
| `CodeConfiguration` | Configuration error |

## Panic Recovery

```go
// Recover from panics
defer error.Recover()

// Or use Try for functions
err := error.Try(func() error {
    // code that might panic
    return nil
})

// Must - panic on error
value := error.Must(safeFunction())
```

## Exit Codes

| Exit Code | Description |
|-----------|-------------|
| `ExitOK` | Success |
| `ExitUsage` | Command line usage error |
| `ExitData` | Data format error |
| `ExitNoInput` | Cannot open input |
| `ExitNoUser` | Addressee unknown |
| `ExitNoHost` | Host unknown |
| `ExitUnavailable` | Service unavailable |
| `ExitSoftware` | Internal error |
| `ExitOS` | System error |
| `ExitOSFile` | OS file error |
| `ExitCantCreate` | Cannot create file |
| `ExitIO` | Read/write error |
| `ExitTempFail` | Temporary failure |
| `ExitProtocol` | Protocol error |
| `ExitNoPerm` | Permission denied |
| `ExitConfig` | Configuration error |

## License

MIT
