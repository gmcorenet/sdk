package gmcoredebugbar

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type contextKey string

const requestScopeKey contextKey = "gmcore_debugbar_scope"

type QueryEntry struct {
	SQL        string        `json:"sql"`
	Args       []string      `json:"args"`
	DurationMS int64         `json:"duration_ms"`
	StartedAt  time.Time     `json:"started_at"`
	Meta       []ValueMetric `json:"meta"`
}

type ValueMetric struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RequestEntry struct {
	RequestID     string        `json:"request_id"`
	ParentID      string        `json:"parent_id"`
	Method        string        `json:"method"`
	Path          string        `json:"path"`
	Status        int           `json:"status"`
	DurationMS    int64         `json:"duration_ms"`
	RemoteAddr    string        `json:"remote_addr"`
	StartedAt     time.Time     `json:"started_at"`
	Headers       http.Header   `json:"headers"`
	IsAJAX        bool          `json:"is_ajax"`
	ContentType   string        `json:"content_type"`
	ResponseBytes int           `json:"response_bytes"`
	UserAgent     string        `json:"user_agent"`
	Route         string        `json:"route"`
	User          string        `json:"user"`
	UserMeta      []ValueMetric `json:"user_meta"`
	Queries       []QueryEntry  `json:"queries"`
	Templates     []string      `json:"templates"`
	Bundles       []string      `json:"bundles"`
	Versions      []ValueMetric `json:"versions"`
}

type Summary struct {
	Count          int   `json:"count"`
	AjaxCount      int   `json:"ajax_count"`
	ErrorCount     int   `json:"error_count"`
	AverageMS      int64 `json:"average_ms"`
	MaxMS          int64 `json:"max_ms"`
	LastStatus     int   `json:"last_status"`
	LastDurationMS int64 `json:"last_duration_ms"`
}

type Snapshot struct {
	Requests []RequestEntry `json:"requests"`
	Summary  Summary        `json:"summary"`
}

type Debugbar struct {
	mu      sync.Mutex
	entries []RequestEntry
	limit   int
}

type requestScope struct {
	mu        sync.Mutex
	requestID string
	parentID  string
	route     string
	user      string
	userMeta  map[string]string
	queries   []QueryEntry
	templates []string
	bundles   []string
	versions  map[string]string
}

func New(limit int) *Debugbar {
	if limit <= 0 {
		limit = 100
	}
	return &Debugbar{limit: limit}
}

func (d *Debugbar) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := nextRequestID()
		scope := &requestScope{
			requestID: requestID,
			parentID:  strings.TrimSpace(r.Header.Get("X-Gmcore-Parent-Request")),
			versions:  map[string]string{},
			userMeta:  map[string]string{},
		}
		ctx := context.WithValue(r.Context(), requestScopeKey, scope)
		r = r.WithContext(ctx)
		w.Header().Set("X-Gmcore-Request-Id", requestID)
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if shouldIgnoreProfilerPath(r.URL.Path) {
			return
		}
		entry := scope.snapshot()
		d.push(RequestEntry{
			RequestID:     requestID,
			ParentID:      entry.ParentID,
			Method:        r.Method,
			Path:          r.URL.RequestURI(),
			Status:        rec.status,
			DurationMS:    time.Since(start).Milliseconds(),
			RemoteAddr:    r.RemoteAddr,
			StartedAt:     start.UTC(),
			Headers:       r.Header.Clone(),
			IsAJAX:        strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") || strings.Contains(strings.ToLower(r.Header.Get("Accept")), "json"),
			ContentType:   rec.Header().Get("Content-Type"),
			ResponseBytes: rec.bytes,
			UserAgent:     r.UserAgent(),
			Route:         entry.Route,
			User:          entry.User,
			UserMeta:      entry.UserMeta,
			Queries:       entry.Queries,
			Templates:     entry.Templates,
			Bundles:       entry.Bundles,
			Versions:      entry.Versions,
		})
	})
}

func shouldIgnoreProfilerPath(path string) bool {
	path = strings.TrimSpace(path)
	return strings.HasPrefix(path, "/_debug") || strings.HasPrefix(path, "/_debugbar")
}

func (d *Debugbar) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(d.snapshot())
	})
}

func (d *Debugbar) SnapshotFor(requestID string) Snapshot {
	full := d.snapshot()
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return full
	}
	byID := map[string]RequestEntry{}
	for _, req := range full.Requests {
		byID[req.RequestID] = req
	}
	root, ok := byID[requestID]
	if !ok {
		return full
	}
	selected := map[string]struct{}{root.RequestID: {}}
	changed := true
	for changed {
		changed = false
		for _, req := range full.Requests {
			if _, ok := selected[req.ParentID]; ok {
				if _, exists := selected[req.RequestID]; !exists {
					selected[req.RequestID] = struct{}{}
					changed = true
				}
			}
		}
	}
	filtered := make([]RequestEntry, 0, len(selected))
	for _, req := range full.Requests {
		if _, ok := selected[req.RequestID]; ok {
			filtered = append(filtered, req)
		}
	}
	summary := Summary{Count: len(filtered)}
	var totalMS int64
	for _, entry := range filtered {
		totalMS += entry.DurationMS
		if entry.IsAJAX {
			summary.AjaxCount++
		}
		if entry.Status >= 400 {
			summary.ErrorCount++
		}
		if entry.DurationMS > summary.MaxMS {
			summary.MaxMS = entry.DurationMS
		}
		summary.LastStatus = entry.Status
		summary.LastDurationMS = entry.DurationMS
	}
	if len(filtered) > 0 {
		summary.AverageMS = totalMS / int64(len(filtered))
	}
	return Snapshot{Requests: filtered, Summary: summary}
}

func (d *Debugbar) PageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(PageHTML(strings.TrimSpace(r.URL.Query().Get("request_id")))))
	})
}

func PageHTML(requestID string) string {
	return debugbarPageHTML(strings.TrimSpace(requestID))
}

func InjectHTML(body []byte, requestID string) []byte {
	snippet := fmt.Sprintf(`<script>(function(){
if(window.__GMCORE_DEBUGBAR__) return;
window.__GMCORE_DEBUGBAR__=true;
var parentRequestId=%q;
var ajaxEntries=[];
var jsErrors=[];
var mounted=false;
var storageKey='gmcore_debug_client_'+(parentRequestId||'root');
function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(m){return({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[m]||m;});}
function nowISO(){try{return new Date().toISOString();}catch(e){return '';}}
function loadState(){
  try{
    var raw=window.localStorage.getItem(storageKey);
    if(!raw) return;
    var parsed=JSON.parse(raw);
    ajaxEntries=Array.isArray(parsed.ajax)?parsed.ajax:[];
    jsErrors=Array.isArray(parsed.errors)?parsed.errors:[];
  }catch(e){}
}
function persistState(){
  try{
    window.localStorage.setItem(storageKey, JSON.stringify({ajax:ajaxEntries,errors:jsErrors,updated_at:nowISO()}));
  }catch(e){}
}
function isDebugRequest(url){
  try{
    var u = new URL(url, window.location.origin);
    return u.pathname.indexOf('/_debug') === 0 || u.pathname.indexOf('/_debugbar') === 0;
  }catch(e){
    return String(url||'').indexOf('/_debug') === 0 || String(url||'').indexOf('/_debugbar') === 0;
  }
}
var refreshTimer=null;
function scheduleRefresh(){ if(refreshTimer) clearTimeout(refreshTimer); refreshTimer=setTimeout(function(){ refresh(); }, 180); }
function pushAjax(entry){ajaxEntries.unshift(entry); if(ajaxEntries.length>%d) ajaxEntries.length=%d; persistState(); renderClientPanels(); scheduleRefresh();}
function pushError(entry){jsErrors.unshift(entry); if(jsErrors.length>50) jsErrors.length=50; persistState(); renderClientPanels();}
window.addEventListener('error',function(e){pushError({message:e.message||'Error',source:e.filename||'',line:e.lineno||0,column:e.colno||0,at:nowISO()});});
window.addEventListener('unhandledrejection',function(e){pushError({message:(e.reason&&e.reason.message)||String(e.reason||'Unhandled rejection'),source:'promise',line:0,column:0,at:nowISO()});});
var originalFetch=window.fetch;
if(originalFetch){window.fetch=function(input,init){var started=performance.now();var method=(init&&init.method)||'GET';var url=(typeof input==='string')?input:((input&&input.url)||'');if(isDebugRequest(url)){return originalFetch.call(this,input,init);}init=init||{};init.headers=new Headers(init.headers||{});if(parentRequestId)init.headers.set('X-Gmcore-Parent-Request',parentRequestId);return originalFetch.call(this,input,init).then(function(response){pushAjax({kind:'fetch',method:method,status:response.status,url:url,duration_ms:Math.round(performance.now()-started),at:nowISO(),request_id:response.headers.get('X-Gmcore-Request-Id')||''});return response;}).catch(function(err){pushAjax({kind:'fetch',method:method,status:0,url:url,duration_ms:Math.round(performance.now()-started),error:String(err),at:nowISO()});throw err;});};}
var XHR=window.XMLHttpRequest;
if(XHR){var open=XHR.prototype.open;var send=XHR.prototype.send;var setHeader=XHR.prototype.setRequestHeader;XHR.prototype.open=function(method,url){this.__gm_method=method;this.__gm_url=url;return open.apply(this,arguments);};XHR.prototype.send=function(){if(isDebugRequest(this.__gm_url||'')){return send.apply(this,arguments);}var started=performance.now();try{if(parentRequestId)setHeader.call(this,'X-Gmcore-Parent-Request',parentRequestId);}catch(e){}this.addEventListener('loadend',function(){pushAjax({kind:'xhr',method:this.__gm_method||'GET',status:this.status||0,url:this.__gm_url||'',duration_ms:Math.round(performance.now()-started),at:nowISO(),request_id:(this.getResponseHeader&&this.getResponseHeader('X-Gmcore-Request-Id'))||''});});return send.apply(this,arguments);};}

function summaryHtml(server){
  var s=(server&&server.summary)||{};
  var versions=latestVersions(server);
  return '<div class="gmdb-grid">'+
    metric('Requests', s.count||0)+
    metric('AJAX', (s.ajax_count||0)+' / '+ajaxEntries.length)+
    metric('Errors', (s.error_count||0)+' / '+jsErrors.length)+
    metric('Avg / Max', (s.average_ms||0)+'ms / '+(s.max_ms||0)+'ms')+
    metric('Framework', esc(versions['framework']||'unknown'))+
    metric('App', esc(versions['app']||'unknown'))+
    metric('Page load', (window.performance&&performance.timing&&performance.timing.navigationStart)?Math.max(0,Date.now()-performance.timing.navigationStart)+'ms':'n/a')+
    metric('URL', esc(location.pathname))+
  '</div>';
}
function latestVersions(server){
  var requests=(server&&server.requests)||[];
  for(var i=requests.length-1;i>=0;i--){
    var row=requests[i];
    if(row && row.versions && row.versions.length){
      var out={};
      row.versions.forEach(function(v){ out[v.key]=v.value; });
      return out;
    }
  }
  return {};
}
function metric(label,value){return '<div class="gmdb-metric"><div class="gmdb-metric-label">'+label+'</div><div class="gmdb-metric-value">'+value+'</div></div>';}
function tableForRequests(rows){
  if(!rows||!rows.length) return '<div class="gmdb-empty">Sin requests</div>';
  return '<div class="gmdb-table-wrap"><table class="gmdb-table"><thead><tr><th>Method</th><th>Path</th><th>Status</th><th>ms</th><th>Type</th><th>Bytes</th></tr></thead><tbody>'+
    rows.map(function(r){return '<tr><td>'+esc(r.method||'')+'</td><td>'+esc(r.path||r.url||'')+'</td><td>'+esc(r.status||0)+'</td><td>'+esc(r.duration_ms||0)+'</td><td>'+esc(r.kind||((r.is_ajax)?'ajax':'http'))+'</td><td>'+esc(r.response_bytes||'')+'</td></tr>';}).join('')+
  '</tbody></table></div>';
}
function renderClientPanels(){
  var ajax=document.getElementById('gmdb-ajax-panel');
  var errs=document.getElementById('gmdb-errors-panel');
  if(ajax){ajax.innerHTML=tableForRequests(ajaxEntries);}
  if(errs){
    errs.innerHTML = jsErrors.length ? '<div class="gmdb-table-wrap"><table class="gmdb-table"><thead><tr><th>Message</th><th>Source</th><th>At</th></tr></thead><tbody>'+
      jsErrors.map(function(e){return '<tr><td>'+esc(e.message)+'</td><td>'+esc(e.source||'')+(e.line?(':'+e.line):'')+'</td><td>'+esc(e.at||'')+'</td></tr>';}).join('')+
      '</tbody></table></div>' : '<div class="gmdb-empty">Sin errores JS</div>';
  }
}
function mount(){
  if(mounted) return;
  mounted=true;
  loadState();
  var root=document.createElement('div');
  root.id='gmcore-debugbar';
  root.innerHTML='<style>%s</style><div class="gmdb-shell"><div class="gmdb-bar"><div class="gmdb-brand">gmcore</div><div class="gmdb-summary"></div><div class="gmdb-actions"><a class="gmdb-link" id="gmdb-open-link" target="_blank" rel="noopener noreferrer">open</a><div class="gmdb-user"><button type="button" class="gmdb-btn" id="gmdb-ajax-toggle">ajax</button><div class="gmdb-user-menu" id="gmdb-ajax-menu"></div></div><div class="gmdb-user"><button type="button" class="gmdb-btn" id="gmdb-user-toggle">user</button><div class="gmdb-user-menu" id="gmdb-user-menu"></div></div></div></div></div>';
  document.body.appendChild(root);
  root.querySelector('#gmdb-open-link').href='/_debug/'+encodeURIComponent(parentRequestId||'');
  root.querySelector('#gmdb-ajax-toggle').onclick=function(){ root.querySelector('#gmdb-ajax-menu').classList.toggle('open'); root.querySelector('#gmdb-user-menu').classList.remove('open'); };
  root.querySelector('#gmdb-user-toggle').onclick=function(){ root.querySelector('#gmdb-user-menu').classList.toggle('open'); };
  document.addEventListener('click', function(e){ if(!root.contains(e.target)){ root.querySelector('#gmdb-user-menu').classList.remove('open'); root.querySelector('#gmdb-ajax-menu').classList.remove('open'); } });
}
function render(server){
  var root=document.getElementById('gmcore-debugbar');
  if(!root) return;
  root.querySelector('.gmdb-summary').innerHTML=summaryHtml(server);
  var requests=(server&&server.requests)||[];
  var current=null;
  for(var i=requests.length-1;i>=0;i--){ if(!parentRequestId || requests[i].request_id===parentRequestId){ current=requests[i]; break; } }
  if(!current && requests.length) current=requests[requests.length-1];
  var meta=(current&&current.user_meta)||[];
  var rows=[['state', current&&current.user ? 'authenticated' : 'anonymous'], ['user', (current&&current.user)||'Anonymous']];
  meta.forEach(function(item){ rows.push([item.key, item.value]); });
  root.querySelector('#gmdb-user-menu').innerHTML='<div class="gmdb-user-card"><div class="gmdb-user-title">Current user</div>'+rows.map(function(row){ return '<div class="gmdb-user-row"><span>'+esc(row[0])+'</span><strong>'+esc(row[1]||'')+'</strong></div>'; }).join('')+'</div>';
  root.querySelector('#gmdb-ajax-menu').innerHTML='<div class="gmdb-user-card"><div class="gmdb-user-title">AJAX requests</div>'+(ajaxEntries.length?ajaxEntries.map(function(entry){ var id=entry.request_id||''; var href=id?('/_debug/'+encodeURIComponent(id)):'#'; return '<a class="gmdb-user-row gmdb-user-link" '+(id?('href="'+href+'" target="_blank" rel="noopener noreferrer"'):'')+'><span>'+esc(entry.method||entry.kind||'ajax')+' '+esc(entry.status||'')+'</span><strong>'+esc(entry.url||'')+'</strong></a>'; }).join(''):'<div class="gmdb-user-row"><span>No AJAX</span><strong>-</strong></div>')+'</div>';
}
function refresh(){var debugUrl = parentRequestId ? ('/_debug/data/' + encodeURIComponent(parentRequestId)) : '/_debugbar/data';fetch(debugUrl,{credentials:'same-origin'}).then(function(r){return r.json();}).then(render).catch(function(err){pushError({message:'Debugbar refresh failed: '+String(err),source:'debugbar',at:nowISO()});});}
if(document.readyState==='loading'){document.addEventListener('DOMContentLoaded',function(){mount();refresh();});}else{mount();refresh();}
})();</script>`, requestID, 200, 200, debugbarStyles())
	idx := bytes.LastIndex(bytes.ToLower(body), []byte("</body>"))
	if idx == -1 {
		return append(body, []byte(snippet)...)
	}
	out := make([]byte, 0, len(body)+len(snippet))
	out = append(out, body[:idx]...)
	out = append(out, []byte(snippet)...)
	out = append(out, body[idx:]...)
	return out
}

func debugbarStyles() string {
	return `.gmdb-shell{position:fixed;left:0;right:0;bottom:0;z-index:2147483000;font:12px/1.35 "Segoe UI",Inter,ui-sans-serif,system-ui,sans-serif;color:#e5e7eb;pointer-events:none}.gmdb-bar{pointer-events:auto;display:flex;align-items:center;gap:12px;justify-content:space-between;background:rgba(15,23,42,.98);border-top:1px solid rgba(255,255,255,.08);padding:8px 12px;box-shadow:0 -10px 35px rgba(15,23,42,.28);backdrop-filter:blur(12px)}.gmdb-brand{font-weight:800;letter-spacing:.08em;text-transform:uppercase;color:#93c5fd;font-size:10px;min-width:62px}.gmdb-summary{flex:1;min-width:0}.gmdb-grid{display:grid;grid-template-columns:repeat(8,minmax(0,1fr));gap:8px}.gmdb-metric{min-width:0;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.06);border-radius:8px;padding:6px 8px}.gmdb-metric-label{font-size:9px;text-transform:uppercase;letter-spacing:.08em;color:#94a3b8;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.gmdb-metric-value{margin-top:2px;font-size:11px;font-weight:700;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.gmdb-actions{display:flex;align-items:center;gap:6px;flex-shrink:0}.gmdb-link,.gmdb-btn{appearance:none;border:1px solid rgba(255,255,255,.12);background:#111827;color:#e5e7eb;border-radius:8px;padding:6px 8px;text-decoration:none;font-weight:600;font-size:11px;cursor:pointer}.gmdb-link:hover,.gmdb-btn:hover{background:#1f2937;color:#fff}.gmdb-user{position:relative}.gmdb-user-menu{display:none;position:absolute;right:0;bottom:calc(100% + 8px);width:340px;max-width:90vw}.gmdb-user-menu.open{display:block}.gmdb-user-card{background:#111827;border:1px solid rgba(255,255,255,.12);border-radius:12px;box-shadow:0 18px 50px rgba(15,23,42,.35);padding:12px;max-height:60vh;overflow:auto}.gmdb-user-title{font-size:10px;text-transform:uppercase;letter-spacing:.08em;color:#93c5fd;font-weight:800;margin-bottom:8px}.gmdb-user-row{display:flex;justify-content:space-between;gap:12px;padding:6px 0;border-bottom:1px solid rgba(255,255,255,.06)}.gmdb-user-row:last-child{border-bottom:0}.gmdb-user-row span{color:#94a3b8;max-width:35%%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.gmdb-user-row strong{color:#e5e7eb;text-align:right;word-break:break-word;flex:1}.gmdb-user-link{text-decoration:none}.gmdb-user-link:hover{background:rgba(255,255,255,.03)}.gmdb-empty{padding:10px;border:1px dashed rgba(255,255,255,.14);border-radius:10px;color:#94a3b8}.gmdb-table-wrap{overflow:auto;max-height:260px}.gmdb-table{width:100%%;border-collapse:collapse;color:#e5e7eb;font-size:12px}.gmdb-table th,.gmdb-table td{padding:8px 10px;border-bottom:1px solid rgba(255,255,255,.08);text-align:left;vertical-align:top}.gmdb-table th{color:#93c5fd;font-size:10px;text-transform:uppercase;letter-spacing:.08em}.gmdb-panels{display:none}@media (max-width:1400px){.gmdb-grid{grid-template-columns:repeat(4,minmax(0,1fr))}}@media (max-width:920px){.gmdb-bar{flex-direction:column;align-items:stretch}.gmdb-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.gmdb-actions{justify-content:flex-end}}`
}

func nextRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		bytes[6] = (bytes[6] & 0x0f) | 0x40
		bytes[8] = (bytes[8] & 0x3f) | 0x80
		return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
	}
	return "debug-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func scopeFromContext(ctx context.Context) *requestScope {
	scope, _ := ctx.Value(requestScopeKey).(*requestScope)
	return scope
}

func RecordRoute(ctx context.Context, route string) {
	scope := scopeFromContext(ctx)
	if scope == nil {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.route = strings.TrimSpace(route)
}

func RecordUser(ctx context.Context, user string) {
	scope := scopeFromContext(ctx)
	if scope == nil {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.user = strings.TrimSpace(user)
}

func RecordUserMetric(ctx context.Context, key, value string) {
	scope := scopeFromContext(ctx)
	if scope == nil || strings.TrimSpace(key) == "" {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.userMeta[key] = strings.TrimSpace(value)
}

func RecordTemplate(ctx context.Context, name string) {
	scope := scopeFromContext(ctx)
	if scope == nil || strings.TrimSpace(name) == "" {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.templates = append(scope.templates, name)
}

func RecordBundle(ctx context.Context, name string) {
	scope := scopeFromContext(ctx)
	name = strings.TrimSpace(name)
	if scope == nil || name == "" {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	for _, current := range scope.bundles {
		if current == name {
			return
		}
	}
	scope.bundles = append(scope.bundles, name)
}

func RecordVersion(ctx context.Context, key, value string) {
	scope := scopeFromContext(ctx)
	if scope == nil || strings.TrimSpace(key) == "" {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.versions[key] = value
}

func RecordSQL(ctx context.Context, sql string, args []interface{}, startedAt time.Time) {
	scope := scopeFromContext(ctx)
	if scope == nil {
		return
	}
	stringArgs := make([]string, 0, len(args))
	for _, arg := range args {
		stringArgs = append(stringArgs, fmt.Sprint(arg))
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.queries = append(scope.queries, QueryEntry{
		SQL:        strings.TrimSpace(sql),
		Args:       stringArgs,
		DurationMS: time.Since(startedAt).Milliseconds(),
		StartedAt:  startedAt.UTC(),
	})
}

func (s *requestScope) snapshot() RequestEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := make([]ValueMetric, 0, len(s.versions))
	for key, value := range s.versions {
		versions = append(versions, ValueMetric{Key: key, Value: value})
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Key < versions[j].Key })
	userMeta := make([]ValueMetric, 0, len(s.userMeta))
	for key, value := range s.userMeta {
		userMeta = append(userMeta, ValueMetric{Key: key, Value: value})
	}
	sort.Slice(userMeta, func(i, j int) bool { return userMeta[i].Key < userMeta[j].Key })
	queries := make([]QueryEntry, len(s.queries))
	copy(queries, s.queries)
	templates := append([]string(nil), s.templates...)
	bundles := append([]string(nil), s.bundles...)
	return RequestEntry{
		ParentID:  s.parentID,
		Route:     s.route,
		User:      s.user,
		UserMeta:  userMeta,
		Queries:   queries,
		Templates: templates,
		Bundles:   bundles,
		Versions:  versions,
	}
}

func debugbarPageHTML(requestID string) string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>gmcore request profiler</title>
  <link href="/assets/tabler/css/tabler.min.css" rel="stylesheet">
  <style>` + debugbarPageStyles() + `</style>
</head>
<body>
  <div class="profiler-layout">
    <aside class="profiler-app-sidebar p-3 p-xl-4">
      <div class="profiler-brand mb-4">
        gmcore
        <small>Request Profiler</small>
      </div>
      <nav class="d-grid gap-2" id="profiler-nav">
        <button class="tab profiler-app-link active text-start" data-panel="overview">Overview</button>
        <button class="tab profiler-app-link text-start" data-panel="request">Request</button>
        <button class="tab profiler-app-link text-start" data-panel="user">User</button>
        <button class="tab profiler-app-link text-start" data-panel="headers">Headers</button>
        <button class="tab profiler-app-link text-start" data-panel="query">Query</button>
        <button class="tab profiler-app-link text-start" data-panel="cookies">Cookies</button>
        <button class="tab profiler-app-link text-start" data-panel="performance">Performance</button>
        <button class="tab profiler-app-link text-start" data-panel="sql">ORM / SQL</button>
        <button class="tab profiler-app-link text-start" data-panel="templates">Templates</button>
        <button class="tab profiler-app-link text-start" data-panel="bundles">Bundles</button>
        <button class="tab profiler-app-link text-start" data-panel="versions">Versions</button>
        <button class="tab profiler-app-link text-start" data-panel="subrequests">Subrequests</button>
        <button class="tab profiler-app-link text-start" data-panel="ajax">AJAX</button>
        <button class="tab profiler-app-link text-start" data-panel="errors">Errors</button>
      </nav>
    </aside>
    <div class="profiler-app-main">
      <header class="profiler-app-topbar px-4 px-xl-5 py-3">
        <div class="d-flex flex-column flex-lg-row justify-content-between align-items-lg-center gap-3">
          <div>
            <div class="eyebrow mb-1">gmcore profiler</div>
            <div class="h4 mb-0">Request inspector</div>
          </div>
          <div class="text-secondary small">Tabler dashboard shell</div>
        </div>
      </header>
      <main class="container-fluid profiler-shell py-4 py-xl-5">
        <div class="row g-4">
          <div class="col-12">
            <div class="card profiler-hero border-0 shadow-sm">
              <div class="card-body p-4 p-xl-5">
                <div class="d-flex flex-column flex-xl-row gap-4 justify-content-between align-items-xl-center">
                  <div>
                    <div class="eyebrow mb-2">gmcore profiler</div>
                    <h1 class="display-6 mb-2">Request inspector</h1>
                    <p class="lead text-secondary mb-0">Una única request, sus subrequests y toda su telemetría: headers, cookies, SQL, templates, bundles, AJAX, errores y versiones.</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="col-12">
            <div id="summary" class="row g-3"></div>
          </div>
          <div class="col-12">
            <div class="row g-4">
              <div class="col-12">
                <div class="card border-0 shadow-sm mb-4">
                  <div class="card-body p-4">
                    <div id="request-head" class="row g-3"></div>
                  </div>
                </div>
                <div class="card border-0 shadow-sm">
                  <div class="card-body p-4 p-xl-5">
                    <div id="overview" class="panel-body active"></div>
                    <div id="request" class="panel-body"></div>
                    <div id="user" class="panel-body"></div>
                    <div id="headers" class="panel-body"></div>
                    <div id="query" class="panel-body"></div>
                    <div id="cookies" class="panel-body"></div>
                    <div id="performance" class="panel-body"></div>
                    <div id="sql" class="panel-body"></div>
                    <div id="templates" class="panel-body"></div>
                    <div id="bundles" class="panel-body"></div>
                    <div id="versions" class="panel-body"></div>
                    <div id="subrequests" class="panel-body"></div>
                    <div id="ajax" class="panel-body"></div>
                    <div id="errors" class="panel-body"></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
      <footer class="profiler-app-footer px-4 px-xl-5 pb-4">
        <div class="d-flex flex-column flex-lg-row justify-content-between gap-2">
          <div>gmcore request profiler</div>
          <div>UUID-scoped inspector · Tabler dashboard shell</div>
        </div>
      </footer>
    </div>
  </div>
  <script>
  (function(){
    function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(m){return({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[m]||m;});}
    function metric(label,value,sub){return '<div class="metric"><div class="metric-label">'+label+'</div><div class="metric-value">'+value+'</div>'+(sub?'<div class="metric-sub">'+sub+'</div>':'')+'</div>';}
    var currentRequestId = ` + strconv.Quote(requestID) + `;
    function clientState(){
      try{
        return JSON.parse(window.localStorage.getItem('gmcore_debug_client_'+currentRequestId) || '{}');
      }catch(e){
        return {};
      }
    }
    function tableRows(rows, columns){
      if(!rows || !rows.length) return '<div class="empty">No data</div>';
      return '<div class="table-wrap"><table><thead><tr>'+columns.map(function(col){return '<th>'+esc(col.label)+'</th>';}).join('')+'</tr></thead><tbody>'+rows.map(function(row){return '<tr>'+columns.map(function(col){return '<td>'+esc(row[col.key] || '')+'</td>';}).join('')+'</tr>';}).join('')+'</tbody></table></div>';
    }
    function parseQuery(path){
      try{
        var u = new URL(path, window.location.origin);
        var rows=[];
        u.searchParams.forEach(function(value,key){ rows.push({key:key,value:value}); });
        return rows;
      }catch(e){
        return [];
      }
    }
    function requestTree(rootId, rows){
      if(!rootId) return '<div class="empty">No request selected</div>';
      var byParent={};
      rows.forEach(function(r){
        var parent=r.parent_id||'';
        if(!byParent[parent]) byParent[parent]=[];
        byParent[parent].push(r);
      });
      function branch(parentId){
        var items = byParent[parentId] || [];
        if(!items.length) return '';
        return '<ul class="tree-list">'+items.map(function(r){
          return '<li><div class="tree-node"><a href="#'+encodeURIComponent(r.request_id)+'" data-request-link="'+encodeURIComponent(r.request_id)+'">'+esc(r.method)+' '+esc(r.path)+'</a><span class="tree-meta">'+esc(r.status)+' · '+esc(r.duration_ms)+'ms</span></div>'+branch(r.request_id)+'</li>';
        }).join('')+'</ul>';
      }
      var root = rows.find(function(r){ return r.request_id===rootId; });
      if(!root) return '<div class="empty">No request selected</div>';
      return '<div class="tree-root"><div class="tree-node"><strong>'+esc(root.method)+' '+esc(root.path)+'</strong><span class="tree-meta">'+esc(root.status)+' · '+esc(root.duration_ms)+'ms</span></div>'+branch(root.request_id)+'</div>';
    }
    function table(rows){
      if(!rows||!rows.length) return '<div class="empty">No data</div>';
      return '<div class="table-wrap"><table><thead><tr><th>Method</th><th>Path</th><th>Status</th><th>ms</th><th>AJAX</th><th>Bytes</th><th>At</th></tr></thead><tbody>'+rows.map(function(r){return '<tr data-request-row="'+esc(r.request_id||'')+'"><td>'+esc(r.method||'')+'</td><td>'+esc(r.path||r.url||'')+'</td><td>'+esc(r.status||0)+'</td><td>'+esc(r.duration_ms||0)+'</td><td>'+esc(r.is_ajax||r.kind||'')+'</td><td>'+esc(r.response_bytes||'')+'</td><td>'+esc(r.started_at||r.at||'')+'</td></tr>';}).join('')+'</tbody></table></div>';
    }
    function parseCookies(r){
      var cookieHeader = ((r.headers||{}).Cookie||[]).join('; ');
      if(!cookieHeader) return [];
      return cookieHeader.split(';').map(function(part){
        var idx = part.indexOf('=');
        if(idx === -1) return {name:part.trim(),value:''};
        return {name:part.slice(0, idx).trim(), value:part.slice(idx + 1).trim()};
      }).filter(function(item){ return item.name; });
    }
    function collectVersions(data){
      var acc={};
      ((data&&data.requests)||[]).forEach(function(r){(r.versions||[]).forEach(function(v){acc[v.key]=v.value;});});
      return Object.keys(acc).sort().map(function(k){return {key:k,value:acc[k]};});
    }
    function selectCurrent(data){
      var requests=(data&&data.requests)||[];
      if(!requests.length) return {current:null, children:[]};
      var root = currentRequestId ? requests.find(function(r){return r.request_id===currentRequestId;}) : requests[requests.length-1];
      if(!root) root = requests[requests.length-1];
      currentRequestId = root.request_id || currentRequestId;
      var children = requests.filter(function(r){ return r.parent_id === currentRequestId; });
      return {current:root, children:children};
    }
    function render(data){
      var s=(data&&data.summary)||{};
      var versions=collectVersions(data);
      var frameworkVersion=versions.find(function(v){return v.key==='framework';});
      var gatewayVersion=versions.find(function(v){return v.key==='app';});
      var selected = selectCurrent(data);
      var client = clientState();
      var ajaxEntries = Array.isArray(client.ajax) ? client.ajax : [];
      var jsErrors = Array.isArray(client.errors) ? client.errors : [];
      var r = selected.current;
      if(!r){
        document.getElementById('overview').innerHTML = '<div class="empty">No request selected</div>';
        return;
      }
      document.getElementById('summary').innerHTML=[
        metric('Request ID', currentRequestId, 'Current focus'),
        metric('Subrequests', selected.children.length, 'Direct children'),
        metric('AJAX', ajaxEntries.length, 'Client-side'),
        metric('Errors', jsErrors.length + (r.status>=400?1:0), 'Client + server'),
        metric('Duration', (r.duration_ms||0)+'ms', 'Server request'),
        metric('Framework', frameworkVersion ? frameworkVersion.value : 'unknown'),
        metric('Gateway', gatewayVersion ? gatewayVersion.value : 'unknown'),
        metric('Status', r.status||0, r.method+' '+(r.route||'')),
      ].map(function(item){return '<div class="col-12 col-md-6 col-xl-4 col-xxl-3">'+item+'</div>';}).join('');
      var userMeta=(r.user_meta||[]);
      var lastLogin=userMeta.find(function(v){return v.key==='last_login';});
      var authState=userMeta.find(function(v){return v.key==='authenticated';});
      document.getElementById('request-head').innerHTML=[
        metric('UUID', r.request_id || '', 'Primary request'),
        metric('Parent', r.parent_id || 'root'),
        metric('Method', r.method || ''),
        metric('Path', r.path || ''),
        metric('Route', r.route || ''),
        metric('User', r.user || 'anonymous', authState ? authState.value : 'guest'),
      ].map(function(item){return '<div class="col-12 col-md-6 col-xl-4">'+item+'</div>';}).join('');
      document.getElementById('overview').innerHTML='<div class="row g-3">'+[
        metric('Status', r.status || 0),
        metric('Duration', (r.duration_ms||0)+'ms'),
        metric('Bytes', r.response_bytes || 0),
        metric('Content-Type', r.content_type || ''),
        metric('Remote address', r.remote_addr || ''),
        metric('Started at', r.started_at || ''),
        metric('Last login', lastLogin ? lastLogin.value : 'n/a'),
      ].map(function(item){return '<div class="col-12 col-md-6 col-xl-4">'+item+'</div>';}).join('')+'</div>';
      document.getElementById('request').innerHTML='<div class="details-grid">'+
        metric('Method', r.method || '')+
        metric('Path', r.path || '')+
        metric('Route', r.route || 'n/a')+
        metric('Remote address', r.remote_addr || 'n/a')+
        metric('Content type', r.content_type || 'n/a')+
        metric('User agent', r.user_agent || 'n/a')+
      '</div>';
      document.getElementById('user').innerHTML=(function(){
        var rows=[{key:'principal',value:r.user||'anonymous'}].concat((r.user_meta||[]).map(function(v){return {key:v.key,value:v.value};}));
        return tableRows(rows,[{key:'key',label:'Field'},{key:'value',label:'Value'}]);
      })();
      document.getElementById('headers').innerHTML=tableRows(Object.keys(r.headers||{}).sort().map(function(k){return {header:k,value:(r.headers[k]||[]).join(", ")};}), [{key:'header',label:'Header'},{key:'value',label:'Value'}]);
      document.getElementById('query').innerHTML=tableRows(parseQuery(r.path||''), [{key:'key',label:'Parameter'},{key:'value',label:'Value'}]);
      document.getElementById('cookies').innerHTML=tableRows(parseCookies(r), [{key:'name',label:'Cookie'},{key:'value',label:'Value'}]);
      document.getElementById('performance').innerHTML='<div class="details-grid">'+metric('Duration', (r.duration_ms||0)+'ms')+metric('Bytes', r.response_bytes || 0)+metric('AJAX', r.is_ajax ? 'yes' : 'no')+metric('User agent', r.user_agent || '')+'</div>';
      document.getElementById('sql').innerHTML=(r.queries||[]).length?'<div class="table-wrap"><table><thead><tr><th>SQL</th><th>Args</th><th>ms</th><th>Started at</th></tr></thead><tbody>'+(r.queries||[]).map(function(q){return '<tr><td><code>'+esc(q.sql||'')+'</code></td><td>'+esc((q.args||[]).join(', '))+'</td><td>'+esc(q.duration_ms||0)+'</td><td>'+esc(q.started_at||'')+'</td></tr>';}).join('')+'</tbody></table></div>':'<div class="empty">No SQL queries recorded for this request</div>';
      document.getElementById('templates').innerHTML=(r.templates||[]).length?'<ul class="plain-list">'+r.templates.map(function(t){return '<li>'+esc(t)+'</li>';}).join('')+'</ul>':'<div class="empty">No templates recorded</div>';
      document.getElementById('bundles').innerHTML=(r.bundles||[]).length?'<ul class="plain-list">'+r.bundles.map(function(t){return '<li>'+esc(t)+'</li>';}).join('')+'</ul>':'<div class="empty">No bundles recorded</div>';
      document.getElementById('versions').innerHTML=versions.length?'<div class="table-wrap"><table><thead><tr><th>Component</th><th>Version</th></tr></thead><tbody>'+versions.map(function(v){return '<tr><td>'+esc(v.key)+'</td><td>'+esc(v.value)+'</td></tr>';}).join('')+'</tbody></table></div>':'<div class="empty">No version data</div>';
      document.getElementById('subrequests').innerHTML=requestTree(currentRequestId, (data&&data.requests)||[]);
      document.getElementById('ajax').innerHTML=ajaxEntries.length?table(ajaxEntries.map(function(a){return {request_id:a.request_id||'',method:a.method||a.kind||'AJAX',path:a.url||'',status:a.status||'',duration_ms:a.duration_ms||0,is_ajax:a.kind||'ajax',response_bytes:'',started_at:a.at||''};})):'<div class="empty">No AJAX telemetry stored for this request</div>';
      document.getElementById('errors').innerHTML=jsErrors.length?'<div class="table-wrap"><table><thead><tr><th>Message</th><th>Source</th><th>At</th></tr></thead><tbody>'+jsErrors.map(function(e){return '<tr><td>'+esc(e.message||'')+'</td><td>'+esc(e.source||'')+'</td><td>'+esc(e.at||'')+'</td></tr>';}).join('')+'</tbody></table></div>':'<div class="empty">No client-side errors stored for this request</div>';
    }
    function refresh(){fetch('/_debug/data/'+encodeURIComponent(currentRequestId),{credentials:'same-origin'}).then(function(r){return r.json();}).then(render);}
    document.addEventListener('click', function(e){
      var link = e.target.closest('[data-request-link]');
      if(!link) return;
      e.preventDefault();
      currentRequestId = link.getAttribute('data-request-link') || currentRequestId;
      window.history.replaceState({}, '', '/_debug/' + encodeURIComponent(currentRequestId));
      refresh();
    });
    document.querySelectorAll('.tab').forEach(function(tab){tab.onclick=function(){document.querySelectorAll('.tab,.panel-body').forEach(function(n){n.classList.remove('active');});tab.classList.add('active');document.getElementById(tab.getAttribute('data-panel')).classList.add('active');};});
    refresh();
  })();
  </script>
</body>
</html>`
}

func debugbarPageStyles() string {
	return `body{margin:0;background:#f5f7fb;color:#1f2937;font:14px/1.5 "Segoe UI",Inter,ui-sans-serif,system-ui,sans-serif}.profiler-layout{min-height:100vh;display:grid;grid-template-columns:280px 1fr}.profiler-app-sidebar{background:linear-gradient(180deg,#fff 0%,#f7f9fc 100%);border-right:1px solid #e6ebf2;position:sticky;top:0;align-self:start;height:100vh;padding-right:0}.profiler-brand{font-size:1.1rem;font-weight:800;letter-spacing:.04em;color:#0f172a}.profiler-brand small{display:block;font-size:.72rem;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:#64748b}.profiler-app-link{display:flex;align-items:center;gap:.75rem;padding:.85rem 1rem;border:1px solid transparent;border-radius:14px;color:#334155;text-decoration:none;font-weight:600;background:#fff}.profiler-app-link:hover{background:#eef4ff;border-color:#d7e3fb;color:#0f172a}.profiler-app-link.active{background:#0f172a;color:#fff;border-color:#0f172a;box-shadow:0 12px 24px rgba(15,23,42,.16)}.profiler-app-main{min-width:0;display:flex;flex-direction:column}.profiler-app-topbar{background:rgba(255,255,255,.92);backdrop-filter:blur(12px);border-bottom:1px solid #e6ebf2;position:sticky;top:0;z-index:20}.profiler-app-footer{color:#64748b;font-size:.9rem;padding-bottom:72px}.profiler-shell{max-width:1720px}.profiler-hero{border-radius:24px;background:linear-gradient(180deg,#ffffff 0%,#eef4ff 100%)}.eyebrow{text-transform:uppercase;letter-spacing:.1em;color:#2563eb;font-size:12px;font-weight:800}.lead{max-width:880px}.tab{border-radius:12px;border:1px solid #dbe3f0;background:#fff;color:#334155;font-weight:600;padding:.85rem 1rem;width:100%}.tab.active{background:#0f172a;color:#fff;border-color:#0f172a}.metric{height:100%;background:#fff;border:1px solid #e5eaf3;border-radius:18px;padding:1rem 1rem .95rem;box-shadow:0 10px 30px rgba(15,23,42,.05)}.metric-label{font-size:11px;text-transform:uppercase;letter-spacing:.08em;color:#64748b;font-weight:700}.metric-value{font-size:1.05rem;font-weight:800;margin-top:6px;word-break:break-word}.metric-sub{font-size:.78rem;color:#64748b;margin-top:6px}.panel-body{display:none}.panel-body.active{display:block}.table-wrap{overflow:auto;border:1px solid #e5eaf3;border-radius:18px;background:#fff}.table-wrap table{width:100%;border-collapse:collapse}.table-wrap th,.table-wrap td{padding:.85rem 1rem;border-bottom:1px solid #eef2f7;text-align:left;vertical-align:top}.table-wrap th{position:sticky;top:0;background:#f8fafc;color:#475569;font-size:.77rem;text-transform:uppercase;letter-spacing:.06em}.empty,.hint{padding:1rem 1.1rem;border:1px dashed #cbd5e1;border-radius:18px;background:#fff;color:#64748b}.details-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px}.plain-list{margin:0;padding-left:1.1rem}.tree-root,.tree-list{margin:0;padding:0;list-style:none}.tree-list{padding-left:1.25rem;border-left:2px solid #e2e8f0;margin-left:.5rem;margin-top:.75rem}.tree-node{display:flex;justify-content:space-between;gap:1rem;align-items:center;padding:.75rem 1rem;border:1px solid #e5eaf3;border-radius:14px;background:#fff;margin-bottom:.75rem}.tree-node a{text-decoration:none;color:#0f172a;font-weight:600}.tree-meta{font-size:.82rem;color:#64748b;white-space:nowrap}.card{border-radius:20px;border:1px solid #e7edf6}.text-secondary{color:#64748b !important}code{white-space:pre-wrap;word-break:break-word}@media (max-width:991.98px){.profiler-layout{grid-template-columns:1fr}.profiler-app-sidebar{position:static;height:auto;border-right:0;border-bottom:1px solid #e6ebf2}}`
}

func (d *Debugbar) push(entry RequestEntry) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries = append(d.entries, entry)
	if len(d.entries) > d.limit {
		d.entries = d.entries[len(d.entries)-d.limit:]
	}
}

func (d *Debugbar) snapshot() Snapshot {
	d.mu.Lock()
	defer d.mu.Unlock()
	requests := make([]RequestEntry, len(d.entries))
	copy(requests, d.entries)
	summary := Summary{Count: len(requests)}
	var totalMS int64
	for _, entry := range requests {
		totalMS += entry.DurationMS
		if entry.IsAJAX {
			summary.AjaxCount++
		}
		if entry.Status >= 400 {
			summary.ErrorCount++
		}
		if entry.DurationMS > summary.MaxMS {
			summary.MaxMS = entry.DurationMS
		}
		summary.LastStatus = entry.Status
		summary.LastDurationMS = entry.DurationMS
	}
	if len(requests) > 0 {
		summary.AverageMS = totalMS / int64(len(requests))
	}
	return Snapshot{
		Requests: requests,
		Summary:  summary,
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}
