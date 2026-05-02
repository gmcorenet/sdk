package error

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type PanicHandler func(interface{})

var defaultPanicHandler PanicHandler = func(v interface{}) {
	fmt.Fprintf(os.Stderr, "PANIC: %v\n", v)
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "Recovered: %v\n", err)
	}
	printStack()
	os.Exit(1)
}

var panicHandler PanicHandler = defaultPanicHandler

func SetPanicHandler(handler PanicHandler) {
	panicHandler = handler
}

func Recover() {
	if r := recover(); r != nil {
		panicHandler(r)
	}
}

func Must[T any](value T, err error) T {
	if err != nil {
		panicHandler(err)
	}
	return value
}

func Try(fn func() error) (err error) {
	defer Recover()
	return fn()
}

func TryValue[T any](fn func() T) (value T, err error) {
	defer func() {
		if r := recover(); r != nil {
			value = *new(T)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn(), nil
}

func printStack() {
	var buf []byte
	for {
		ns := runtime.Stack(buf, false)
		if ns < len(buf) {
			buf = make([]byte, ns)
			continue
		}
		break
	}
	fmt.Fprintf(os.Stderr, "%s\n", buf)
}

type ExitCode int

const (
	ExitOK ExitCode = 0
	ExitUsage      ExitCode = 64
	ExitData       ExitCode = 65
	ExitNoInput    ExitCode = 66
	ExitNoUser     ExitCode = 67
	ExitNoHost     ExitCode = 68
	ExitUnavailable ExitCode = 69
	ExitSoftware   ExitCode = 70
	ExitOS         ExitCode = 71
	ExitOSFile     ExitCode = 72
	ExitCantCreate ExitCode = 73
	ExitIO         ExitCode = 74
	ExitTempFail   ExitCode = 75
	ExitProtocol   ExitCode = 76
	ExitNoPerm     ExitCode = 77
	ExitConfig     ExitCode = 78
	ExitUnknown    ExitCode = 79
)

func (c ExitCode) ToErrorCode() ErrorCode {
	switch c {
	case ExitUsage:
		return CodeInvalidInput
	case ExitData:
		return CodeInvalidInput
	case ExitNoInput:
		return CodeNotFound
	case ExitNoUser:
		return CodeNotFound
	case ExitNoHost:
		return CodeNetwork
	case ExitUnavailable:
		return CodeInternal
	case ExitSoftware:
		return CodeInternal
	case ExitOS:
		return CodeInternal
	case ExitOSFile:
		return CodeInternal
	case ExitCantCreate:
		return CodeInternal
	case ExitIO:
		return CodeNetwork
	case ExitTempFail:
		return CodeNetwork
	case ExitProtocol:
		return CodeNetwork
	case ExitNoPerm:
		return CodeForbidden
	case ExitConfig:
		return CodeConfiguration
	default:
		return CodeUnknown
	}
}

func (e *GmcoreError) ExitCode() ExitCode {
	switch e.code {
	case CodeNotFound:
		return ExitNoInput
	case CodeInvalidInput:
		return ExitUsage
	case CodeUnauthorized:
		return ExitNoPerm
	case CodeForbidden:
		return ExitNoPerm
	case CodeInternal:
		return ExitSoftware
	case CodeNetwork:
		return ExitIO
	case CodeTimeout:
		return ExitTempFail
	case CodeNotImplemented:
		return ExitUnavailable
	case CodeConfiguration:
		return ExitConfig
	default:
		return ExitUnknown
	}
}

func (e *GmcoreError) Exit() {
	os.Exit(int(e.ExitCode()))
}

func HandleSignals(handlers ...func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		for _, h := range handlers {
			h()
		}
		fmt.Fprintf(os.Stderr, "Received signal %s\n", sig)
		os.Exit(128 + int(sig.(syscall.Signal)))
	}()
}
