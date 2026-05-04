# Error SDK

Sistema de errores estructurado con códigos de error, stack traces y wrapping.

## Códigos de Error

```go
const (
    CodeUnknown ErrorCode = iota + 1  // 1
    CodeNotFound                      // 2
    CodeInvalidInput                   // 3
    CodeUnauthorized                   // 4
    CodeForbidden                      // 5
    CodeInternal                       // 6
    CodeNetwork                        // 7
    CodeTimeout                        // 8
    CodeNotImplemented                 // 9
    CodeConfiguration                  // 10
    CodeIO                             // 11
)
```

## Estructura GmcoreError

```go
type GmcoreError struct {
    code       ErrorCode
    message    string
    technical  string
    cause      error
    stackTrace []string
}
```

## Funciones de Creación

### `New(code ErrorCode, message string) *GmcoreError`
Crea un error nuevo con código y mensaje.

### `NewTech(code ErrorCode, technical string) *GmcoreError`
Crea un error con mensaje técnico.

### `Wrap(err error, code ErrorCode, message string) *GmcoreError`
Envuelve un error existente. Si ya es GmcoreError, lo retorna sin modificar.

### `WrapTech(err error, code ErrorCode, technical string) *GmcoreError`
Versión técnica del Wrap.

## Métodos de Modificación

```go
err.WithMessage(msg)       // Cambia el mensaje
err.WithTechnical(tech)    // Cambia el mensaje técnico
err.WithCause(cause)       // Establece la causa
```

## Métodos de Consulta

```go
err.Error()        // Retorna message o technical
err.Code()         // Retorna el ErrorCode
err.Unwrap()       // Retorna el error causa
err.Technical()    // Retorna detalles técnicos
err.StackTrace()   // Retorna slice de frames
```

## Funciones de Tipo

```go
IsGmcoreError(err error)       // Check if error is *GmcoreError
AsGmcoreError(err error) (*GmcoreError, bool)  // Type assertion
```

## Uso

```go
err := gmerr.Wrap(io.ErrUnexpectedEOF, gmerr.CodeIO, "failed to read config")
if err.Code() == gmerr.CodeIO {
    // handle IO error
}
```
