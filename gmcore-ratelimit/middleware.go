package gmcore_ratelimit

import (
	"net/http"
	"strings"
	"time"

	"github.com/gmcorenet/framework/container"
	"github.com/gmcorenet/framework/router"
	"github.com/gmcorenet/framework/routing"
)

func init() {
	routing.RegisterMiddlewareProvider(func(ctr *container.Container, r *router.Router) (func(http.Handler) http.Handler, bool) {
		return NewRateLimitMiddleware(ctr), true
	})
}

type RateLimitAnnotation struct {
	Key string
	Max int
	Per string
}

type rateLimitConfig struct {
	Name    string `yaml:"name"`
	Max     int    `yaml:"max"`
	Per     string `yaml:"per"`
	Enabled bool   `yaml:"enabled"`
}

func NewRateLimitMiddleware(ctr *container.Container) func(http.Handler) http.Handler {
	limiters := make(map[string]RateLimiter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller, action := parseRateLimitPath(r.URL.Path)
			key := "ratelimit." + controller + "." + action

			raw, err := ctr.Get(key)
			if err != nil || raw == nil {
				next.ServeHTTP(w, r)
				return
			}

			var ruleMax int
			var ruleWindow time.Duration
			var ruleName string

			switch v := raw.(type) {
			case *RateLimitAnnotation:
				if v.Key != "" {
					cfg := getRateLimitConfigFromContainer(ctr, v.Key)
					if cfg != nil {
						ruleMax = cfg.Max
						ruleWindow = parseDuration(cfg.Per)
						ruleName = cfg.Name
					}
				}
				if ruleMax == 0 && v.Max > 0 {
					ruleMax = v.Max
				}
				if ruleWindow == 0 && v.Per != "" {
					ruleWindow = parseDuration(v.Per)
				}
			case RateLimitAnnotation:
				if v.Key != "" {
					cfg := getRateLimitConfigFromContainer(ctr, v.Key)
					if cfg != nil {
						ruleMax = cfg.Max
						ruleWindow = parseDuration(cfg.Per)
						ruleName = cfg.Name
					}
				}
				if ruleMax == 0 && v.Max > 0 {
					ruleMax = v.Max
				}
				if ruleWindow == 0 && v.Per != "" {
					ruleWindow = parseDuration(v.Per)
				}
			case map[string]interface{}:
				if cfgKey, ok := v["key"].(string); ok && cfgKey != "" {
					cfg := getRateLimitConfigFromContainer(ctr, cfgKey)
					if cfg != nil {
						ruleMax = cfg.Max
						ruleWindow = parseDuration(cfg.Per)
						ruleName = cfg.Name
					}
				}
				if ruleMax == 0 {
					if maxVal, ok := v["max"].(float64); ok {
						ruleMax = int(maxVal)
					} else if maxVal, ok := v["max"].(int); ok {
						ruleMax = maxVal
					}
				}
				if ruleWindow == 0 {
					if perVal, ok := v["per"].(string); ok {
						ruleWindow = parseDuration(perVal)
					}
				}
			}

			if ruleMax <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			if ruleWindow <= 0 {
				ruleWindow = time.Minute
			}

			if ruleName == "" {
				ruleName = controller + "." + action
			}

			limiterKey := ruleName
			limiter, ok := limiters[limiterKey]
			if !ok {
				limiter = NewRateLimiter(ruleMax, ruleWindow)
				limiters[limiterKey] = limiter
			}

			clientIP := getClientIP(r)

			if !limiter.Allow(clientIP) {
				w.Header().Set("X-RateLimit-Limit", itoa(ruleMax))
				w.Header().Set("Retry-After", itoa(int(ruleWindow.Seconds())))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"code":429,"message":"Too Many Requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseRateLimitPath(path string) (controller, action string) {
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

func getRateLimitConfigFromContainer(ctr *container.Container, cfgKey string) *rateLimitConfig {
	raw, err := ctr.Get(cfgKey)
	if err != nil || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *rateLimitConfig:
		return v
	case rateLimitConfig:
		return &v
	case map[string]interface{}:
		cfg := &rateLimitConfig{Enabled: true}
		if name, ok := v["name"].(string); ok {
			cfg.Name = name
		}
		if maxVal, ok := v["max"].(float64); ok {
			cfg.Max = int(maxVal)
		} else if maxVal, ok := v["max"].(int); ok {
			cfg.Max = maxVal
		}
		if per, ok := v["per"].(string); ok {
			cfg.Per = per
		}
		return cfg
	}
	return nil
}

func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	switch strings.ToLower(s) {
	case "second", "seconds":
		return time.Second
	case "minute", "minutes":
		return time.Minute
	case "hour", "hours":
		return time.Hour
	case "day", "days":
		return 24 * time.Hour
	case "week", "weeks":
		return 7 * 24 * time.Hour
	case "month", "months":
		return 30 * 24 * time.Hour
	}
	return 0
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[:idx]
	}
	return addr
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
