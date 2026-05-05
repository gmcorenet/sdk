package gmcore_events

import (
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Listeners map[string][]ListenerConfig `yaml:"listeners" json:"listeners"`
}

type ListenerConfig struct {
	Type    string `yaml:"type" json:"type"`
	Handler string `yaml:"handler" json:"handler"`
	Async   bool   `yaml:"async" json:"async"`
}

type ConfigLoader struct {
	appPath string
	env     map[string]string
}

func NewConfigLoader(appPath string) *ConfigLoader {
	return &ConfigLoader{
		appPath: appPath,
		env:     gmcore_config.LoadAppEnv(appPath),
	}
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
		filepath.Join(l.appPath, "config", "events.yaml"),
		filepath.Join(l.appPath, "config", "events.yml"),
		filepath.Join(l.appPath, "events.yaml"),
		filepath.Join(l.appPath, "events.yml"),
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

func (c *Config) ApplyTo(bus *Bus, registry HandlerRegistry) {
	for eventName, listeners := range c.Listeners {
		for _, listener := range listeners {
			handler := registry.Get(listener.Handler)
			if handler == nil {
				continue
			}

			listener := func(ctx context.Context, event interface{}) error {
				return handler.Handle(ctx, event)
			}

			if listener.Async {
				go func() {
					listener(context.Background(), nil)
				}()
			} else {
				bus.Subscribe(eventName, listener)
			}
		}
	}
}

type HandlerRegistry interface {
	Get(name string) EventHandler
}

type EventHandler interface {
	Handle(ctx context.Context, event interface{}) error
}

type DefaultHandlerRegistry struct {
	handlers map[string]func(context.Context, interface{}) error
}

func NewDefaultHandlerRegistry() *DefaultHandlerRegistry {
	return &DefaultHandlerRegistry{
		handlers: make(map[string]func(context.Context, interface{}) error),
	}
}

func (r *DefaultHandlerRegistry) Register(name string, handler func(context.Context, interface{}) error) {
	r.handlers[name] = handler
}

func (r *DefaultHandlerRegistry) Get(name string) EventHandler {
	if handler, ok := r.handlers[name]; ok {
		return &funcHandler{handler: handler}
	}
	return nil
}

type funcHandler struct {
	handler func(context.Context, interface{}) error
}

func (h *funcHandler) Handle(ctx context.Context, event interface{}) error {
	return h.handler(ctx, event)
}
