package gmcore_security

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/gmcorenet/framework/container"
	"github.com/gmcorenet/framework/router"
	"github.com/gmcorenet/framework/routing"
	"gopkg.in/yaml.v3"
)

func init() {
	routing.RegisterMiddlewareProvider(func(ctr *container.Container, r *router.Router) (func(http.Handler) http.Handler, bool) {
		raw, err := ctr.Get("security.route_map")
		if err != nil || raw == nil {
			return nil, false
		}
		if _, ok := raw.(map[string]map[string]interface{}); !ok {
			return nil, false
		}
		return NewSecurityMiddleware(ctr, nil), true
	})
}

type contextKey string

const userContextKey contextKey = "gmcore_security_user"

func StoreUserContext(r *http.Request, user User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func UserFromContext(r *http.Request) User {
	if user, ok := r.Context().Value(userContextKey).(User); ok {
		return user
	}
	return nil
}

type SecurityRule struct {
	Roles      []string `yaml:"roles" json:"roles"`
	Strategy   string   `yaml:"strategy" json:"strategy"`
	Expression string   `yaml:"expression" json:"expression"`
}

func LoadSecurityRules(path string) (map[string]SecurityRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Security struct {
			Rules map[string]SecurityRule `yaml:"rules"`
		} `yaml:"security"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.Security.Rules, nil
}

func NewSecurityMiddleware(ctr *container.Container, userProvider func(r *http.Request) User) func(http.Handler) http.Handler {
	checker := NewSecurityChecker()
	checker.AddVoter(NewRoleVoter("ROLE_"))

	if checkerSvc, err := ctr.Get("security_checker"); err == nil {
		if sc, ok := checkerSvc.(*SecurityChecker); ok {
			checker = sc
		}
	}

	if userProvider == nil {
		userProvider = UserFromContext
	}

	rules := make(map[string]SecurityRule)
	if raw, err := ctr.Get("security_rules"); err == nil {
		if m, ok := raw.(map[string]SecurityRule); ok {
			rules = m
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := ctr.Get("security.route_map")
			if err != nil || raw == nil {
				next.ServeHTTP(w, r)
				return
			}

			routeMap, ok := raw.(map[string]map[string]interface{})
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			controller, action := resolveRouteAction(routeMap, r.URL.Path)
			if controller == "" {
				next.ServeHTTP(w, r)
				return
			}

			key := "security." + controller + "." + action
			raw, err = ctr.Get(key)
			if err != nil || raw == nil {
				next.ServeHTTP(w, r)
				return
			}

			var roles []string
			var strategy string
			var yamlKey string

			switch v := raw.(type) {
			case map[string]interface{}:
				if rl, ok := v["roles"].([]interface{}); ok {
					for _, r := range rl {
						if s, ok := r.(string); ok {
							roles = append(roles, s)
						}
					}
				}
				if s, ok := v["strategy"].(string); ok {
					strategy = s
				}
				if k, ok := v["key"].(string); ok {
					yamlKey = k
				}
			}

			if yamlKey != "" {
				if rule, ok := rules[yamlKey]; ok {
					roles = rule.Roles
					if rule.Strategy != "" {
						strategy = rule.Strategy
					}
				}
			}

			if len(roles) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			user := userProvider(r)
			if user == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			var granted bool
			switch strategy {
			case "all":
				granted = checker.IsGrantedAll(user, roles, r)
			default:
				granted = checker.IsGrantedAny(user, roles, r)
			}

			if !granted {
				http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func resolveRouteAction(routeMap map[string]map[string]interface{}, path string) (controller, action string) {
	path = "/" + strings.Trim(path, "/")
	if path == "/" || path == "" {
		path = "/"
	}

	if entry, ok := routeMap[path]; ok {
		if id, ok := entry["controller"].(string); ok {
			controller = id
		}
		if act, ok := entry["action"].(string); ok {
			action = act
		}
		return
	}

	for pattern, entry := range routeMap {
		parts := strings.Split(pattern, "/")
		pathParts := strings.Split(path, "/")
		if len(parts) != len(pathParts) {
			continue
		}
		match := true
		for i := range parts {
			if strings.HasPrefix(parts[i], "{") {
				continue
			}
			if parts[i] != pathParts[i] {
				match = false
				break
			}
		}
		if match {
			if id, ok := entry["controller"].(string); ok {
				controller = id
			}
			if act, ok := entry["action"].(string); ok {
				action = act
			}
			return
		}
	}

	return "", ""
}
