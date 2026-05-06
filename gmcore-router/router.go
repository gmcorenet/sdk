package gmcore_router

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type routerContextKey struct{}

var paramsKey = routerContextKey{}

type Route struct {
	Method  string
	Path    string
	Name    string
	Handler http.HandlerFunc
}

type Router struct {
	routes   []Route
	notFound http.HandlerFunc
}

type Group struct {
	router     *Router
	pathPrefix string
	namePrefix string
}

func New() *Router {
	return &Router{}
}

func (r *Router) SetNotFound(handler http.HandlerFunc) {
	r.notFound = handler
}

func (r *Router) Add(method, path, name string, handler http.HandlerFunc) {
	r.routes = append(r.routes, Route{
		Method:  strings.ToUpper(strings.TrimSpace(method)),
		Path:    path,
		Name:    name,
		Handler: handler,
	})
}

func (r *Router) Group(pathPrefix, namePrefix string) *Group {
	return &Group{
		router:     r,
		pathPrefix: normalizePrefix(pathPrefix),
		namePrefix: strings.TrimSpace(namePrefix),
	}
}

func (g *Group) Add(method, path, name string, handler http.HandlerFunc) {
	g.router.Add(method, joinPath(g.pathPrefix, path), g.namePrefix+strings.TrimSpace(name), handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range r.routes {
		method := strings.ToUpper(req.Method)
		if method == http.MethodHead && route.Method == http.MethodGet {
			method = http.MethodGet
		}
		if route.Method != "" && route.Method != method {
			continue
		}
		params, ok := matchPath(route.Path, req.URL.Path)
		if !ok {
			continue
		}
		ctx := context.WithValue(req.Context(), paramsKey, params)
		route.Handler(w, req.WithContext(ctx))
		return
	}
	if r.notFound != nil {
		r.notFound(w, req)
		return
	}
	http.NotFound(w, req)
}

func Param(req *http.Request, key string) string {
	params, _ := req.Context().Value(paramsKey).(map[string]string)
	return params[key]
}

func (r *Router) URL(name string, params map[string]string) string {
	for _, route := range r.routes {
		if route.Name != name {
			continue
		}
		path := route.Path
		for key, value := range params {
			path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
		}
		return path
	}
	return ""
}

func (r *Router) Routes() []Route {
	if r == nil {
		return nil
	}
	out := append([]Route(nil), r.routes...)
	return out
}

func (r *Router) NamedRoutes() map[string]string {
	out := map[string]string{}
	if r == nil {
		return out
	}
	for _, route := range r.routes {
		name := strings.TrimSpace(route.Name)
		if name == "" {
			continue
		}
		out[name] = route.Path
	}
	return out
}

func (r *Router) NamedRoutesSorted() []Route {
	out := []Route{}
	if r == nil {
		return out
	}
	out = append(out, r.routes...)
	sort.SliceStable(out, func(i, j int) bool {
		return strings.TrimSpace(out[i].Name) < strings.TrimSpace(out[j].Name)
	})
	return out
}

func matchPath(pattern, path string) (map[string]string, bool) {
	pp, pTrailing := splitWithTrailing(pattern)
	cp, cTrailing := splitWithTrailing(path)

	if len(pp) != len(cp) {
		return nil, false
	}

	if len(pp) > 0 && !pTrailing && cTrailing {
		return nil, false
	}

	params := map[string]string{}
	for i := range pp {
		if key, value, ok := matchSegmentParam(pp[i], cp[i]); ok {
			params[key] = value
			continue
		}
		if pp[i] != cp[i] {
			return nil, false
		}
	}
	return params, true
}

func splitWithTrailing(path string) ([]string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return []string{}, false
	}
	hasTrailing := strings.HasSuffix(path, "/")
	return strings.Split(strings.Trim(path, "/"), "/"), hasTrailing
}

func split(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return []string{}
	}
	return strings.Split(strings.Trim(path, "/"), "/")
}

func matchSegmentParam(pattern, value string) (string, string, bool) {
	open := strings.Index(pattern, "{")
	close := strings.Index(pattern, "}")
	if open < 0 || close <= open {
		return "", "", false
	}
	prefix := pattern[:open]
	key := strings.TrimSpace(pattern[open+1 : close])
	suffix := pattern[close+1:]
	if key == "" || !strings.HasPrefix(value, prefix) {
		return "", "", false
	}
	if suffix != "" && !strings.HasSuffix(value, suffix) {
		return "", "", false
	}
	param := strings.TrimSuffix(strings.TrimPrefix(value, prefix), suffix)
	return key, param, true
}

func normalizePrefix(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return ""
	}
	return "/" + strings.Trim(path, "/")
}

func joinPath(prefix, path string) string {
	prefix = normalizePrefix(prefix)
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		if prefix == "" {
			return "/"
		}
		return prefix
	}
	if prefix == "" {
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}
