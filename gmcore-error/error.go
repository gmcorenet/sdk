package error

import (
	"fmt"
	"runtime"
	"strings"
)

type ErrorCode int

const (
	CodeUnknown ErrorCode = iota + 1
	CodeNotFound
	CodeInvalidInput
	CodeUnauthorized
	CodeForbidden
	CodeInternal
	CodeNetwork
	CodeTimeout
	CodeNotImplemented
	CodeConfiguration
)

type GmcoreError struct {
	code       ErrorCode
	message    string
	technical  string
	cause      error
	stackTrace []string
}

func (e *GmcoreError) Error() string {
	if e.message != "" {
		return e.message
	}
	return e.technical
}

func (e *GmcoreError) Unwrap() error {
	return e.cause
}

func (e *GmcoreError) Code() ErrorCode {
	return e.code
}

func (e *GmcoreError) Technical() string {
	return e.technical
}

func (e *GmcoreError) StackTrace() []string {
	return e.stackTrace
}

func New(code ErrorCode, message string) *GmcoreError {
	return &GmcoreError{
		code:       code,
		message:    message,
		technical:  message,
		stackTrace: captureStack(),
	}
}

func NewTech(code ErrorCode, technical string) *GmcoreError {
	return &GmcoreError{
		code:       code,
		message:    technical,
		technical:  technical,
		stackTrace: captureStack(),
	}
}

func Wrap(err error, code ErrorCode, message string) *GmcoreError {
	if err == nil {
		return nil
	}
	if ge, ok := err.(*GmcoreError); ok {
		return ge
	}
	return &GmcoreError{
		code:       code,
		message:    message,
		technical:  fmt.Sprintf("%s: %v", message, err),
		cause:      err,
		stackTrace: captureStack(),
	}
}

func WrapTech(err error, code ErrorCode, technical string) *GmcoreError {
	if err == nil {
		return nil
	}
	if ge, ok := err.(*GmcoreError); ok {
		return ge
	}
	return &GmcoreError{
		code:       code,
		message:    technical,
		technical:  fmt.Sprintf("%s: %v", technical, err),
		cause:      err,
		stackTrace: captureStack(),
	}
}

func (e *GmcoreError) WithMessage(msg string) *GmcoreError {
	e.message = msg
	return e
}

func (e *GmcoreError) WithTechnical(tech string) *GmcoreError {
	e.technical = tech
	return e
}

func (e *GmcoreError) WithCause(cause error) *GmcoreError {
	e.cause = cause
	return e
}

func captureStack() []string {
	var frames []string
	const depth = 32
	for i := 1; i < depth; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}
		frames = append(frames, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
	}
	return frames
}

func (e *GmcoreError) String() string {
	var b strings.Builder
	b.WriteString("Error: ")
	b.WriteString(e.message)
	b.WriteString("\nCode: ")
	b.WriteString(codeToString(e.code))
	if e.cause != nil {
		b.WriteString("\nCause: ")
		b.WriteString(e.cause.Error())
	}
	if len(e.stackTrace) > 0 {
		b.WriteString("\nStack:")
		for _, frame := range e.stackTrace[:min(5, len(e.stackTrace))] {
			b.WriteString("\n  ")
			b.WriteString(frame)
		}
	}
	return b.String()
}

func codeToString(code ErrorCode) string {
	switch code {
	case CodeNotFound:
		return "NOT_FOUND"
	case CodeInvalidInput:
		return "INVALID_INPUT"
	case CodeUnauthorized:
		return "UNAUTHORIZED"
	case CodeForbidden:
		return "FORBIDDEN"
	case CodeInternal:
		return "INTERNAL"
	case CodeNetwork:
		return "NETWORK"
	case CodeTimeout:
		return "TIMEOUT"
	case CodeNotImplemented:
		return "NOT_IMPLEMENTED"
	case CodeConfiguration:
		return "CONFIGURATION"
	default:
		return "UNKNOWN"
	}
}

func IsGmcoreError(err error) bool {
	_, ok := err.(*GmcoreError)
	return ok
}

func AsGmcoreError(err error) (*GmcoreError, bool) {
	ge, ok := err.(*GmcoreError)
	return ge, ok
}
