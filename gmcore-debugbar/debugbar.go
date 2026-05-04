package gmcore_debugbar

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"
)

type DebugBar struct {
	panels  []Panel
	enabled bool
}

type Panel interface {
	GetName() string
	GetHTML() string
}

func New() *DebugBar {
	return &DebugBar{enabled: true, panels: make([]Panel, 0)}
}

func (d *DebugBar) Enable()  { d.enabled = true }
func (d *DebugBar) Disable() { d.enabled = false }
func (d *DebugBar) IsEnabled() bool { return d.enabled }

func (d *DebugBar) AddPanel(p Panel) {
	d.panels = append(d.panels, p)
}

func (d *DebugBar) Render() string {
	if !d.enabled {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<div id="debug-bar" style="position:fixed;bottom:0;left:0;right:0;background:#2d2d2d;color:#fff;padding:8px;font-size:12px;z-index:99999;font-family:monospace;">`)
	for _, p := range d.panels {
		sb.WriteString(fmt.Sprintf(`<span style="margin-right:15px;" title="%s">%s</span>`,
			html.EscapeString(p.GetName()),
			html.EscapeString(p.GetHTML())))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

type responseWriter struct {
	http.ResponseWriter
	written       bool
	statusCode    int
	contentType   string
	htmlDetected  bool
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.written = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	if rw.htmlDetected || strings.Contains(http.DetectContentType(b), "text/html") {
		rw.htmlDetected = true
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (d *DebugBar) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !d.enabled {
			next.ServeHTTP(w, r)
			return
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		if rw.htmlDetected && rw.statusCode == http.StatusOK {
			debugHTML := d.Render()
			js := fmt.Sprintf(`<script>(function(){var d=document.getElementById('debug-bar');if(d)d.insertAdjacentHTML('beforebegin',%q);})();</script>`,
				html.EscapeString(debugHTML))
			fmt.Fprint(w, js)
		}
	})
}

type RequestsPanel struct {
	count int
}

func NewRequestsPanel() *RequestsPanel {
	return &RequestsPanel{}
}

func (p *RequestsPanel) GetName() string { return "requests" }
func (p *RequestsPanel) GetHTML() string { return fmt.Sprintf("Requests: %d", p.count) }

type TimelinePanel struct {
	events []TimelineEvent
}

type TimelineEvent struct {
	Name      string
	Start     time.Time
	End       time.Time
	Duration  time.Duration
}

func (p *TimelinePanel) GetName() string { return "timeline" }
func (p *TimelinePanel) GetHTML() string {
	return fmt.Sprintf("Events: %d", len(p.events))
}

type MemoryPanel struct {
	alloc uint64
}

func NewMemoryPanel() *MemoryPanel {
	return &MemoryPanel{}
}

func (p *MemoryPanel) GetName() string { return "memory" }
func (p *MemoryPanel) GetHTML() string {
	return fmt.Sprintf("Memory: %.2f MB", float64(p.alloc)/1024/1024)
}

type DatabasePanel struct {
	queries []string
}

func NewDatabasePanel() *DatabasePanel {
	return &DatabasePanel{queries: make([]string, 0)}
}

func (p *DatabasePanel) AddQuery(query string) {
	p.queries = append(p.queries, query)
}

func (p *DatabasePanel) GetName() string { return "database" }
func (p *DatabasePanel) GetHTML() string {
	return fmt.Sprintf("Queries: %d", len(p.queries))
}

type CachePanel struct {
	hits   int
	misses int
}

func NewCachePanel() *CachePanel {
	return &CachePanel{}
}

func (p *CachePanel) GetName() string { return "cache" }
func (p *CachePanel) GetHTML() string {
	total := p.hits + p.misses
	if total == 0 {
		return "Cache: N/A"
	}
	ratio := float64(p.hits) / float64(total) * 100
	return fmt.Sprintf("Cache: %.0f%%", ratio)
}

type LogsPanel struct {
	logs []string
}

func NewLogsPanel() *LogsPanel {
	return &LogsPanel{logs: make([]string, 0)}
}

func (p *LogsPanel) AddLog(log string) {
	p.logs = append(p.logs, log)
}

func (p *LogsPanel) GetName() string { return "logs" }
func (p *LogsPanel) GetHTML() string {
	return fmt.Sprintf("Logs: %d", len(p.logs))
}
