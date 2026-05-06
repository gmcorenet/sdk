package gmcore_router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Routes []RouteConfig `yaml:"routes" json:"routes"`
}

type RouteConfig struct {
	Name    string   `yaml:"name" json:"name"`
	Path    string   `yaml:"path" json:"path"`
	Handler string   `yaml:"handler" json:"handler"`
	Methods []string `yaml:"methods" json:"methods"`
}

type HandlerRegistry interface {
	Get(name string) HandlerInfo
	CreateHandler(name string) http.HandlerFunc
}

type HandlerInfo struct {
	Controller string
	Method     string
}

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"routes.yaml", "routes.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func (c *Config) ApplyTo(router *Router, registry HandlerRegistry) error {
	if router == nil {
		return fmt.Errorf("router cannot be nil")
	}
	if registry == nil {
		return fmt.Errorf("handler registry cannot be nil")
	}

	for _, route := range c.Routes {
		handler := registry.CreateHandler(route.Handler)
		methods := normalizeMethods(route.Methods)
		if len(methods) == 0 {
			router.Add("", route.Path, route.Name, handler)
			continue
		}

		for _, method := range methods {
			router.Add(method, route.Path, route.Name, handler)
		}
	}
	return nil
}

func normalizeMethods(methods []string) []string {
	if len(methods) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		m := strings.ToUpper(strings.TrimSpace(method))
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		normalized = append(normalized, m)
	}
	return normalized
}

type DefaultHandlerRegistry struct {
	handlers map[string]func() http.HandlerFunc
}

func NewDefaultHandlerRegistry() *DefaultHandlerRegistry {
	return &DefaultHandlerRegistry{
		handlers: make(map[string]func() http.HandlerFunc),
	}
}

func (r *DefaultHandlerRegistry) Register(name string, factory func() http.HandlerFunc) {
	r.handlers[name] = factory
}

func (r *DefaultHandlerRegistry) Get(name string) HandlerInfo {
	return HandlerInfo{Controller: name, Method: "Handle"}
}

func (r *DefaultHandlerRegistry) CreateHandler(name string) http.HandlerFunc {
	if factory, ok := r.handlers[name]; ok {
		return factory()
	}
	return func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Handler not found: "+name, http.StatusInternalServerError)
	}
}
