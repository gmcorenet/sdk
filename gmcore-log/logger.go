package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var levelNames = []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}

func (l Level) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return "UNKNOWN"
	}
	return levelNames[l]
}

func ParseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG", "0":
		return LevelDebug
	case "info", "INFO", "1":
		return LevelInfo
	case "warn", "WARN", "WARNING", "2":
		return LevelWarn
	case "error", "ERROR", "3":
		return LevelError
	case "fatal", "FATAL", "4":
		return LevelFatal
	default:
		return LevelInfo
	}
}

type Logger struct {
	mu       sync.Mutex
	handlers []Handler
	level    Level
	fields   map[string]interface{}
}

func New() *Logger {
	return &Logger{
		level:  LevelInfo,
		fields: make(map[string]interface{}),
	}
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) AddHandler(h Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, h)
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}
	fields[key] = value
	return &Logger{
		handlers: l.handlers,
		level:    l.level,
		fields:   fields,
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := make(map[string]interface{})
	for k, v := range l.fields {
		m[k] = v
	}
	for k, v := range fields {
		m[k] = v
	}
	return &Logger{handlers: l.handlers, level: l.level, fields: m}
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	entry := Entry{
		Time:    time.Now(),
		Level:   level,
		Message: fmt.Sprintf(msg, args...),
		Fields:  l.fields,
	}
	handlers := l.handlers
	l.mu.Unlock()

	for _, h := range handlers {
		h.Handle(entry)
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) { l.log(LevelDebug, msg, args...) }
func (l *Logger) Info(msg string, args ...interface{}) { l.log(LevelInfo, msg, args...) }
func (l *Logger) Warn(msg string, args ...interface{}) { l.log(LevelWarn, msg, args...) }
func (l *Logger) Error(msg string, args ...interface{}) { l.log(LevelError, msg, args...) }
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

type Entry struct {
	Time    time.Time
	Level   Level
	Message string
	Fields  map[string]interface{}
}

type Handler interface {
	Handle(Entry)
}

type HandlerFunc func(Entry)

func (f HandlerFunc) Handle(e Entry) { f(e) }

type ConsoleHandler struct {
	Writer io.Writer
	Format Format
}

func NewConsoleHandler(w io.Writer) *ConsoleHandler {
	return &ConsoleHandler{Writer: w, Format: TextFormat{}}
}

func (h *ConsoleHandler) Handle(e Entry) {
	h.Format.FormatEntry(e, h.Writer)
}

type Format interface {
	FormatEntry(Entry, io.Writer)
}

type TextFormat struct{}

func (f TextFormat) FormatEntry(e Entry, w io.Writer) {
	fmt.Fprintf(w, "%s [%s] %s", e.Time.Format(time.RFC3339), e.Level.String(), e.Message)
	if len(e.Fields) > 0 {
		fmt.Fprint(w, " {")
		first := true
		for k, v := range e.Fields {
			if !first {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, "%s=%v", k, v)
			first = false
		}
		fmt.Fprint(w, "}")
	}
	fmt.Fprintln(w)
}

type JSONFormat struct{}

func (f JSONFormat) FormatEntry(e Entry, w io.Writer) {
	m := map[string]interface{}{
		"time":    e.Time.Format(time.RFC3339),
		"level":   e.Level.String(),
		"message": e.Message,
	}
	for k, v := range e.Fields {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	w.Write(b)
	fmt.Fprintln(w)
}

type FileHandler struct {
	Filename string
	Format   Format
	file     *os.File
	mu       sync.Mutex
}

func NewFileHandler(filename string) (*FileHandler, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileHandler{
		Filename: filename,
		Format:   TextFormat{},
		file:     f,
	}, nil
}

func (h *FileHandler) Handle(e Entry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Format.FormatEntry(e, h.file)
}

func (h *FileHandler) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

type RotatingFileHandler struct {
	Filename    string
	MaxSize     int64
	MaxBackups  int
	Format      Format
	currentSize int64
	file        *os.File
	mu          sync.Mutex
}

func NewRotatingFileHandler(filename string, maxSize int64, maxBackups int) (*RotatingFileHandler, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	info, _ := f.Stat()
	return &RotatingFileHandler{
		Filename:    filename,
		MaxSize:     maxSize,
		MaxBackups:  maxBackups,
		Format:      TextFormat{},
		currentSize: info.Size(),
		file:        f,
	}, nil
}

func (h *RotatingFileHandler) Handle(e Entry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entryLen := int64(len(e.Message) + 100)
	if h.currentSize+entryLen > h.MaxSize {
		h.rotate()
	}

	h.Format.FormatEntry(e, h.file)
	h.currentSize += entryLen
}

func (h *RotatingFileHandler) rotate() {
	h.file.Close()

	oldName := h.Filename + ".old"
	os.Rename(h.Filename, oldName)

	if h.MaxBackups > 0 {
		backup := fmt.Sprintf("%s.%d", h.Filename, h.MaxBackups)
		os.Remove(backup)
		for i := h.MaxBackups - 1; i > 0; i-- {
			src := fmt.Sprintf("%s.%d", h.Filename, i)
			dst := fmt.Sprintf("%s.%d", h.Filename, i+1)
			os.Rename(src, dst)
		}
		os.Rename(oldName, h.Filename+".1")
	}

	h.file, _ = os.OpenFile(h.Filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	h.currentSize = 0
}

func (h *RotatingFileHandler) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

type SyslogHandler struct {
	Facility int
	Format   Format
}

func NewSyslogHandler() (*SyslogHandler, error) {
	return &SyslogHandler{Facility: 1, Format: TextFormat{}}, nil
}

func (h *SyslogHandler) Handle(e Entry) {
	priority := h.Facility*8 + syslogLevel(e.Level)
	syslog(priority, e.Message)
}

func syslogLevel(level Level) int {
	switch level {
	case LevelDebug:
		return 7
	case LevelInfo:
		return 6
	case LevelWarn:
		return 4
	case LevelError:
		return 3
	case LevelFatal:
		return 2
	default:
		return 6
	}
}

func syslog(priority int, message string) {
	fmt.Fprintf(os.Stderr, "<%d>%s\n", priority, message)
}

var defaultLogger = New()

func SetLevel(level Level)                    { defaultLogger.SetLevel(level) }
func AddHandler(h Handler)                   { defaultLogger.AddHandler(h) }
func WithField(key string, v interface{}) *Logger { return defaultLogger.WithField(key, v) }
func WithFields(m map[string]interface{}) *Logger  { return defaultLogger.WithFields(m) }
func Debug(msg string, args ...interface{})   { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...interface{})    { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...interface{})    { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...interface{})   { defaultLogger.Error(msg, args...) }
func Fatal(msg string, args ...interface{})    { defaultLogger.Fatal(msg, args...) }
