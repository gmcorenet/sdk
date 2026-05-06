package gmcore_debugbar

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gmcorenet/framework/container"
	"github.com/gmcorenet/framework/router"
	"github.com/gmcorenet/framework/routing"
)

func init() {
	routing.RegisterMiddlewareProvider(func(ctr *container.Container, r *router.Router) (func(http.Handler) http.Handler, bool) {
		env := "dev"
		if e, err := ctr.Get("app_env"); err == nil {
			if s, ok := e.(string); ok {
				env = s
			}
		}
		db := New(env)
		if !db.IsEnabled() {
			return nil, false
		}
		ctr.Set("debugbar", db)

		r.Handle("^_profiler/latest/?$", []string{"GET"}, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			db.ProfilerHandler().ServeHTTP(w, r)
		})
		r.Handle("^_profiler/?$", []string{"GET"}, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			db.ProfilerHandler().ServeHTTP(w, r)
		})

		return db.Middleware, true
	})
}

type DebugBar struct {
	panels     []Panel
	enabled    bool
	collectors []DataCollector
	mu         sync.RWMutex
}

type Panel interface {
	Name() string
	Icon() string
	Render(w http.ResponseWriter, r *http.Request)
}

type DataCollector interface {
	Collect(r *http.Request, profile *Profile)
}

type Profile struct {
	Method      string
	URL         string
	StatusCode  int
	Route       string
	Controller  string
	IP          string
	Time        time.Time
	Duration    time.Duration
	Memory      uint64
	Timeline    []TimelineEntry
	Queries     []QueryEntry
	Logs        []LogEntry
	CacheStats  CacheStats
	Headers     map[string]string
	Exception   *ExceptionEntry
	mu          sync.Mutex
}

type TimelineEntry struct {
	Name     string
	Start    time.Time
	Duration time.Duration
}

type QueryEntry struct {
	SQL      string
	Params   []interface{}
	Duration time.Duration
}

type LogEntry struct {
	Level   string
	Message string
	Time    time.Time
}

type CacheStats struct {
	Hits      int
	Misses    int
	Keys      []string
	TotalSize int64
}

type ExceptionEntry struct {
	Message string
	Code    int
	File    string
	Line    int
	Trace   string
}

func New(env string) *DebugBar {
	db := &DebugBar{
		panels:     make([]Panel, 0),
		enabled:    env == "dev",
		collectors: make([]DataCollector, 0),
	}

	db.AddPanel(&RequestPanel{})
	db.AddPanel(&TimelinePanel{events: make([]TimelineEntry, 0)})
	db.AddPanel(&MemoryPanel{})
	db.AddPanel(&DatabasePanel{queries: make([]QueryEntry, 0)})
	db.AddPanel(&CachePanel{})
	db.AddPanel(&LogPanel{entries: make([]LogEntry, 0)})
	db.AddPanel(&RoutePanel{})
	db.AddPanel(&ConfigPanel{})
	db.AddPanel(&ExceptionPanel{})

	db.AddCollector(&RequestCollector{})
	db.AddCollector(&MemoryCollector{})

	return db
}

func (db *DebugBar) Enable()  { db.enabled = true }
func (db *DebugBar) Disable() { db.enabled = false }
func (db *DebugBar) IsEnabled() bool { return db.enabled }

func (db *DebugBar) AddPanel(p Panel) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.panels = append(db.panels, p)
}

func (db *DebugBar) AddCollector(c DataCollector) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.collectors = append(db.collectors, c)
}

func (db *DebugBar) AddQuery(sql string, params []interface{}, d time.Duration) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if dp, ok := p.(*DatabasePanel); ok {
			dp.queries = append(dp.queries, QueryEntry{SQL: sql, Params: params, Duration: d})
		}
	}
}

func (db *DebugBar) AddLog(level, message string) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if lp, ok := p.(*LogPanel); ok {
			lp.entries = append(lp.entries, LogEntry{Level: level, Message: message, Time: time.Now()})
		}
	}
}

func (db *DebugBar) AddTimeline(name string, d time.Duration) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if tp, ok := p.(*TimelinePanel); ok {
			tp.events = append(tp.events, TimelineEntry{Name: name, Start: time.Now().Add(-d), Duration: d})
		}
	}
}

func (db *DebugBar) Stopwatch(name string) func() {
	start := time.Now()
	return func() {
		db.AddTimeline(name, time.Since(start))
	}
}

func (db *DebugBar) SetException(msg string, code int, file string, line int, trace string) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if ep, ok := p.(*ExceptionPanel); ok {
			ep.exception = &ExceptionEntry{Message: msg, Code: code, File: file, Line: line, Trace: trace}
		}
	}
}

func (db *DebugBar) SetCacheStats(hits, misses int, keys []string, size int64) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if cp, ok := p.(*CachePanel); ok {
			cp.hits = hits
			cp.misses = misses
			cp.keys = keys
			cp.size = size
		}
	}
}

func (db *DebugBar) SetRoutes(routes []string) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if rp, ok := p.(*RoutePanel); ok {
			rp.routes = routes
		}
	}
}

func (db *DebugBar) SetConfig(cfg map[string]string) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, p := range db.panels {
		if cp, ok := p.(*ConfigPanel); ok {
			cp.configs = cfg
		}
	}
}

type profilerResponseWriter struct {
	http.ResponseWriter
	profile    *Profile
	debugbar   *DebugBar
	statusCode int
	written    bool
}

func (rw *profilerResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.profile.StatusCode = statusCode
	rw.written = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *profilerResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func (db *DebugBar) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !db.enabled {
			next.ServeHTTP(w, r)
			return
		}

		startTime := time.Now()
		profile := &Profile{
			Method: r.Method,
			URL:    r.URL.String(),
			IP:     r.RemoteAddr,
			Time:   time.Now(),
		}

		for _, c := range db.collectors {
			c.Collect(r, profile)
		}

		for _, p := range db.panels {
			if tp, ok := p.(*TimelinePanel); ok {
				tp.events = make([]TimelineEntry, 0)
			}
			if rp, ok := p.(*RequestPanel); ok {
				rp.method = r.Method
				rp.url = r.URL.String()
				rp.statusCode = http.StatusOK
			}
		}

		rw := &profilerResponseWriter{ResponseWriter: w, profile: profile, debugbar: db, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		profile.Duration = time.Since(startTime)
		profile.StatusCode = rw.statusCode

		contentType := w.Header().Get("Content-Type")
		if strings.Contains(contentType, "text/html") && rw.statusCode < 400 {
			html := db.renderToolbar(rw.statusCode, profile)
			fmt.Fprint(w, html)
		}

		db.saveProfile(profile)
	})
}

func (db *DebugBar) renderToolbar(statusCode int, profile *Profile) string {
	var sb strings.Builder

	sb.WriteString(`<style>
#sfToolbar{position:fixed;bottom:0;left:0;right:0;z-index:99999;font:12px/1.4 system-ui,sans-serif}
.sf-toolbar{display:flex;background:#222;color:#fff;height:36px;align-items:center;padding:0 8px}
.sf-toolbar-item{display:flex;align-items:center;margin-right:4px;padding:0 8px;height:36px;cursor:pointer;border-left:1px solid #444}
.sf-toolbar-item:hover{background:#333}
.sf-toolbar-item .label{font-weight:bold;margin-right:4px}
.sf-toolbar-item .value{color:#aaa}
.sf-toolbar-info{display:none;background:#1a1a1a;color:#ddd;padding:12px;max-height:300px;overflow-y:auto;border-top:1px solid #444;font-family:monospace}
.sf-toolbar-info.open{display:block}
.sf-toolbar-status{display:inline-block;width:16px;height:16px;border-radius:50%;margin-right:6px}
.sf-status-2xx{background:#4f805d}.sf-status-3xx{background:#a5a515}.sf-status-4xx{background:#a73535}.sf-status-5xx{background:#a73535}
.sf-toolbar-reset{float:right;color:#888;text-decoration:none}
.sf-toolbar-reset:hover{color:#fff}
</style>`)

	sb.WriteString(`<div id="sfToolbar"><div class="sf-toolbar" onclick="var i=event.target.closest('.sf-toolbar-item');if(i){var d=i.nextElementSibling;if(d)d.classList.toggle('open')}">`)

	statusClass := "sf-status-2xx"
	if statusCode >= 500 {
		statusClass = "sf-status-5xx"
	} else if statusCode >= 400 {
		statusClass = "sf-status-4xx"
	} else if statusCode >= 300 {
		statusClass = "sf-status-3xx"
	}

	sb.WriteString(fmt.Sprintf(`<span class="sf-toolbar-item"><span class="sf-toolbar-status %s"></span><span class="value">%d</span></span>`, statusClass, statusCode))

	for _, p := range db.panels {
		sb.WriteString(fmt.Sprintf(`<span class="sf-toolbar-item"><span class="label">%s</span><span class="value">%s</span></span>`, p.Icon(), p.Name()))
		sb.WriteString(fmt.Sprintf(`<div class="sf-toolbar-info">%s</div>`, html.EscapeString(db.renderPanelContent(p, profile))))
	}

	sb.WriteString(fmt.Sprintf(`<span class="sf-toolbar-item"><span class="label">%s</span><span class="value">%s</span></span>`, "⏱", profile.Duration.Round(time.Millisecond).String()))
	sb.WriteString(fmt.Sprintf(`<span class="sf-toolbar-item"><span class="label">%s</span><span class="value">%.1f MB</span></span>`, "📊", float64(profile.Memory)/1024/1024))

	profLink := "/_profiler/latest"
	sb.WriteString(fmt.Sprintf(`<a class="sf-toolbar-reset" href="%s" target="_blank">📋</a>`, profLink))

	sb.WriteString(`</div></div>`)
	return sb.String()
}

func (db *DebugBar) renderPanelContent(p Panel, profile *Profile) string {
	var sb strings.Builder
	switch panel := p.(type) {
	case *RequestPanel:
		sb.WriteString(fmt.Sprintf("Method: %s\nURL: %s\nIP: %s\nRoute: %s\nController: %s\nStatus: %d\nTime: %s\n",
			profile.Method, profile.URL, profile.IP, profile.Route, profile.Controller, profile.StatusCode, profile.Time.Format("15:04:05")))
	case *TimelinePanel:
		for _, e := range panel.events {
			sb.WriteString(fmt.Sprintf("%-30s %s\n", e.Name, e.Duration.Round(time.Microsecond)))
		}
		sb.WriteString(fmt.Sprintf("\nTotal: %s", profile.Duration.Round(time.Millisecond)))
	case *DatabasePanel:
		sb.WriteString(fmt.Sprintf("Queries: %d\n", len(panel.queries)))
		for _, q := range panel.queries {
			sb.WriteString(fmt.Sprintf("\n%s [%s]", q.SQL, q.Duration.Round(time.Millisecond)))
		}
	case *LogPanel:
		for _, l := range panel.entries {
			sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", l.Time.Format("15:04:05"), l.Level, l.Message))
		}
	case *RoutePanel:
		for _, r := range panel.routes {
			sb.WriteString(r + "\n")
		}
	case *ConfigPanel:
		keys := make([]string, 0, len(panel.configs))
		for k := range panel.configs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("%-20s = %s\n", k, panel.configs[k]))
		}
	case *ExceptionPanel:
		if panel.exception != nil {
			sb.WriteString(fmt.Sprintf("%s\nFile: %s:%d\n\n%s", panel.exception.Message, panel.exception.File, panel.exception.Line, panel.exception.Trace))
		} else {
			sb.WriteString("No exceptions")
		}
	case *CachePanel:
		total := panel.hits + panel.misses
		ratio := 0.0
		if total > 0 {
			ratio = float64(panel.hits) / float64(total) * 100
		}
		sb.WriteString(fmt.Sprintf("Hits: %d | Misses: %d | Ratio: %.0f%% | Keys: %d | Size: %d B", panel.hits, panel.misses, ratio, len(panel.keys), panel.size))
	default:
		sb.WriteString(p.Name())
	}
	return sb.String()
}

func (db *DebugBar) saveProfile(profile *Profile) {
	os.MkdirAll("var/profiler", 0755)
	data, _ := json.MarshalIndent(profile, "", "  ")
	name := fmt.Sprintf("var/profiler/%s.json", profile.Time.Format("20060102-150405"))
	os.WriteFile(name, data, 0644)
}

func (db *DebugBar) ProfilerHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		entries, _ := os.ReadDir("var/profiler")
		var profiles []Profile
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, _ := os.ReadFile("var/profiler/" + e.Name())
			var p Profile
			json.Unmarshal(data, &p)
			if p.URL != "" {
				profiles = append(profiles, p)
			}
		}

		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].Time.After(profiles[j].Time)
		})

		w.Write([]byte(`<html><head><title>GMCore Profiler</title>
<style>body{font:14px system-ui;margin:20px;background:#f5f5f5}
table{width:100%%;border-collapse:collapse;background:#fff;box-shadow:0 2px 4px rgba(0,0,0,.1)}
th,td{padding:10px 15px;text-align:left;border-bottom:1px solid #eee}
th{background:#222;color:#fff;font-weight:600}
tr:hover{background:#f9f9f9}
.token{font-family:monospace;font-size:12px;color:#888}
.badge{display:inline-block;padding:2px 8px;border-radius:3px;font-size:11px;font-weight:600}
.badge-2xx{background:#4f805d;color:#fff}.badge-3xx{background:#a5a515;color:#fff}.badge-4xx{background:#a73535;color:#fff}.badge-5xx{background:#a73535;color:#fff}
</style></head><body><h1>🔍 GMCore Profiler</h1>`))

		if len(profiles) == 0 {
			w.Write([]byte("<p>No profiles yet. Make some requests to your app.</p>"))
		} else {
			n := 50
			if len(profiles) < n {
				n = len(profiles)
			}
			w.Write([]byte(fmt.Sprintf(`<p>%d profiles (showing last %d)</p><table><tr><th>Time</th><th>Method</th><th>Status</th><th>URL</th><th>Duration</th><th>Memory</th></tr>`, len(profiles), n)))

			for i, p := range profiles {
				if i >= n {
					break
				}
				badge := "badge-2xx"
				if p.StatusCode >= 500 {
					badge = "badge-5xx"
				} else if p.StatusCode >= 400 {
					badge = "badge-4xx"
				} else if p.StatusCode >= 300 {
					badge = "badge-3xx"
				}
				w.Write([]byte(fmt.Sprintf(`<tr><td>%s</td><td><span class="token">%s</span></td><td><span class="badge %s">%d</span></td><td>%s</td><td>%s</td><td>%.1f MB</td></tr>`,
					p.Time.Format("15:04:05"),
					p.Method, badge, p.StatusCode,
					html.EscapeString(p.URL),
					p.Duration.Round(time.Millisecond).String(),
					float64(p.Memory)/1024/1024,
				)))
			}
			w.Write([]byte(`</table>`))
		}
		w.Write([]byte(`</body></html>`))
	})
}

type RequestPanel struct {
	method     string
	url        string
	statusCode int
}

func (p *RequestPanel) Name() string {
	if p.statusCode >= 500 {
		return fmt.Sprintf("%d", p.statusCode)
	}
	if p.statusCode >= 400 {
		return fmt.Sprintf("%d", p.statusCode)
	}
	return fmt.Sprintf("%d", p.statusCode)
}
func (p *RequestPanel) Icon() string { return "📡" }
func (p *RequestPanel) Render(w http.ResponseWriter, r *http.Request) {}

type TimelinePanel struct {
	events []TimelineEntry
}

func (p *TimelinePanel) Name() string { return fmt.Sprintf("%d ms", 0) }
func (p *TimelinePanel) Icon() string { return "⏱" }
func (p *TimelinePanel) Render(w http.ResponseWriter, r *http.Request) {}

type MemoryPanel struct{}

func (p *MemoryPanel) Name() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("%.1f MB", float64(m.Alloc)/1024/1024)
}
func (p *MemoryPanel) Icon() string { return "📊" }
func (p *MemoryPanel) Render(w http.ResponseWriter, r *http.Request) {}

type DatabasePanel struct {
	queries []QueryEntry
}

func (p *DatabasePanel) Name() string { return fmt.Sprintf("%d", len(p.queries)) }
func (p *DatabasePanel) Icon() string { return "🗄" }
func (p *DatabasePanel) Render(w http.ResponseWriter, r *http.Request) {}

type CachePanel struct {
	hits   int
	misses int
	keys   []string
	size   int64
}

func (p *CachePanel) Name() string { return fmt.Sprintf("%d", p.hits+p.misses) }
func (p *CachePanel) Icon() string { return "💾" }
func (p *CachePanel) Render(w http.ResponseWriter, r *http.Request) {}

type LogPanel struct {
	entries []LogEntry
}

func (p *LogPanel) Name() string {
	errs := 0
	for _, e := range p.entries {
		if e.Level == "error" || e.Level == "critical" {
			errs++
		}
	}
	if errs > 0 {
		return fmt.Sprintf("%d ⚠", errs)
	}
	return fmt.Sprintf("%d", len(p.entries))
}
func (p *LogPanel) Icon() string { return "📜" }
func (p *LogPanel) Render(w http.ResponseWriter, r *http.Request) {}

type RoutePanel struct {
	routes []string
}

func (p *RoutePanel) Name() string { return fmt.Sprintf("%d", len(p.routes)) }
func (p *RoutePanel) Icon() string { return "🔀" }
func (p *RoutePanel) Render(w http.ResponseWriter, r *http.Request) {}

type ConfigPanel struct {
	configs map[string]string
}

func (p *ConfigPanel) Name() string { return fmt.Sprintf("%d", len(p.configs)) }
func (p *ConfigPanel) Icon() string { return "⚙" }
func (p *ConfigPanel) Render(w http.ResponseWriter, r *http.Request) {}

type ExceptionPanel struct {
	exception *ExceptionEntry
}

func (p *ExceptionPanel) Name() string {
	if p.exception != nil {
		return "1 ⚠"
	}
	return "0"
}
func (p *ExceptionPanel) Icon() string { return "💥" }
func (p *ExceptionPanel) Render(w http.ResponseWriter, r *http.Request) {}

type RequestCollector struct{}

func (c *RequestCollector) Collect(r *http.Request, profile *Profile) {
	profile.IP = r.RemoteAddr
	profile.Headers = make(map[string]string)
	for k, v := range r.Header {
		profile.Headers[k] = strings.Join(v, ", ")
	}
}

type MemoryCollector struct{}

func (c *MemoryCollector) Collect(r *http.Request, profile *Profile) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	profile.Memory = m.Alloc
}
