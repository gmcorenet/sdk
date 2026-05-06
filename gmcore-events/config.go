package gmcore_events

import (
	"context"

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

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"events.yaml", "events.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func (c *Config) ApplyTo(bus *Bus, registry HandlerRegistry) {
	for eventName, listeners := range c.Listeners {
		for _, listenerConfig := range listeners {
			handler := registry.Get(listenerConfig.Handler)
			if handler == nil {
				continue
			}

			handleFunc := func(ctx context.Context, event interface{}) error {
				return handler.Handle(ctx, event)
			}

			if listenerConfig.Async {
				// TODO: async event dispatch requires a worker pool with proper context propagation.
				// For now, register synchronously.
				bus.Subscribe(eventName, handleFunc)
			} else {
				bus.Subscribe(eventName, handleFunc)
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
