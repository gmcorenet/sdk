package gmcore_router

import (
	"net/http"
	"os"
	"path/filepath"

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
}

type HandlerInfo struct {
	Controller string
	Method    string
}

type ConfigLoader struct {
	appPath  string
	env      map[string]string
	registry HandlerRegistry
}

func NewConfigLoader(appPath string) *ConfigLoader {
	return &ConfigLoader{
		appPath: appPath,
		env:     gmcore_config.LoadAppEnv(appPath),
	}
}

func (l *ConfigLoader) SetRegistry(r HandlerRegistry) {
	l.registry = r
}

func (l *ConfigLoader) Load(path string) (*Config, error) {
	cfg := &Config{}

	opts := gmcore_config.Options{
		Env:        l.env,
		Parameters: map[string]string{},
		Strict:     false,
	}

	if err := gmcore_config.LoadYAML(path, cfg, opts); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (l *ConfigLoader) LoadDefault() (*Config, error) {
	candidates := []string{
		filepath.Join(l.appPath, "config", "routes.yaml"),
		filepath.Join(l.appPath, "config", "routes.yml"),
		filepath.Join(l.appPath, "routes.yaml"),
		filepath.Join(l.appPath, "routes.yml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return l.Load(path)
		}
	}

	return nil, nil
}

func LoadConfig(appPath string) (*Config, error) {
	loader := NewConfigLoader(appPath)
	return loader.LoadDefault()
}

func (c *Config) ApplyTo(router *Router, registry HandlerRegistry) {
	for _, route := range c.Routes {
		handler := registry.Get(route.Handler)
		router.Add(
			joinMethods(route.Methods),
			route.Path,
			route.Name,
			handler.Handle,
		)
	}
}

func joinMethods(methods []string) string {
	if len(methods) == 0 {
		return ""
	}
	result := methods[0]
	for i := 1; i < len(methods); i++ {
		result += "," + methods[i]
	}
	return result
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
