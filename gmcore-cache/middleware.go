package gmcore_cache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gmcorenet/framework/container"
	"github.com/gmcorenet/framework/router"
	"github.com/gmcorenet/framework/routing"
)

func init() {
	routing.RegisterMiddlewareProvider(func(ctr *container.Container, r *router.Router) (func(http.Handler) http.Handler, bool) {
		if mgr, err := ctr.Get("cache_manager"); err == nil && mgr != nil {
			if cm, ok := mgr.(CacheManager); ok {
				return NewCacheMiddleware(cm, ctr), true
			}
		}
		return nil, false
	})
}

type CacheAnnotation struct {
	TTL  int
	Tags []string
}

type cachedResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        *bytes.Buffer
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		body:           new(bytes.Buffer),
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.statusCode = statusCode
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) writeHeaders() map[string][]string {
	headers := make(map[string][]string)
	for key, vals := range r.ResponseWriter.Header() {
		headers[key] = vals
	}
	return headers
}

func NewCacheMiddleware(mgr CacheManager, ctr *container.Container) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller, action := parseCachePath(r.URL.Path)
			key := "cache." + controller + "." + action

			ttl := 0
			raw, err := ctr.Get(key)
			if err == nil && raw != nil {
				switch v := raw.(type) {
				case int:
					ttl = v
				case *CacheAnnotation:
					ttl = v.TTL
				case CacheAnnotation:
					ttl = v.TTL
				default:
					next.ServeHTTP(w, r)
					return
				}
			}

			if ttl <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			cacheKey := buildCacheKey(r)
			if cached, ok := mgr.Get(cacheKey); ok {
				resp := decodeCachedResponse(cached)
				if resp != nil {
					for k, vals := range resp.Headers {
						for _, v := range vals {
							w.Header().Add(k, v)
						}
					}
					w.Header().Set("X-Cache", "HIT")
					w.WriteHeader(resp.StatusCode)
					w.Write(resp.Body)
					return
				}
			}

			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)

			if rec.statusCode >= 200 && rec.statusCode < 400 {
				cached := cachedResponse{
					StatusCode: rec.statusCode,
					Headers:    rec.writeHeaders(),
					Body:       rec.body.Bytes(),
				}
				if err := storeCachedResponse(mgr, cacheKey, &cached, ttl); err == nil {
					w.Header().Set("X-Cache", "MISS")
				}
			}
		})
	}
}

func parseCachePath(path string) (controller, action string) {
	path = strings.Trim(path, "/")
	if path == "" {
		return "index", "index"
	}
	parts := strings.Split(path, "/")
	controller = parts[0]
	if controller == "" {
		controller = "index"
	}
	if len(parts) == 1 {
		action = "index"
	} else {
		action = parts[1]
	}
	if action == "" {
		action = "index"
	}
	return strings.ToLower(controller), strings.ToLower(action)
}

func buildCacheKey(r *http.Request) string {
	hash := md5.Sum([]byte(r.Method + ":" + r.URL.String()))
	return "http:" + hex.EncodeToString(hash[:])
}

func encodeCachedResponse(resp *cachedResponse) string {
	data, _ := json.Marshal(resp)
	return string(data)
}

func decodeCachedResponse(raw interface{}) *cachedResponse {
	var r cachedResponse
	switch v := raw.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &r); err != nil {
			return nil
		}
	case []byte:
		if err := json.Unmarshal(v, &r); err != nil {
			return nil
		}
	case *cachedResponse:
		return v
	case cachedResponse:
		return &v
	default:
		return nil
	}
	return &r
}

func storeCachedResponse(mgr CacheManager, key string, resp *cachedResponse, ttl int) error {
	if mgr == nil {
		return fmt.Errorf("cache manager is nil")
	}
	encoded := encodeCachedResponse(resp)
	return mgr.Set(key, encoded)
}
