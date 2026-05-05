package gmcore_log

import (
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Level    string            `yaml:"level" json:"level"`
	Handlers []HandlerConfig   `yaml:"handlers" json:"handlers"`
}

type HandlerConfig struct {
	Type   string                 `yaml:"type" json:"type"`
	Params map[string]interface{} `yaml:"params" json:"params"`
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
		filepath.Join(l.appPath, "config", "log.yaml"),
		filepath.Join(l.appPath, "config", "log.yml"),
		filepath.Join(l.appPath, "config", "logging.yaml"),
		filepath.Join(l.appPath, "config", "logging.yml"),
		filepath.Join(l.appPath, "log.yaml"),
		filepath.Join(l.appPath, "log.yml"),
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

func (c *Config) Build() (*Logger, error) {
	logger := New()

	if c.Level != "" {
		logger.SetLevel(ParseLevel(c.Level))
	}

	for _, h := range c.Handlers {
		handler, err := c.buildHandler(h)
		if err != nil {
			return nil, err
		}
		logger.AddHandler(handler)
	}

	return logger, nil
}

func (c *Config) buildHandler(cfg HandlerConfig) (Handler, error) {
	switch cfg.Type {
	case "console":
		return c.buildConsoleHandler(cfg.Params)
	case "file":
		return c.buildFileHandler(cfg.Params)
	case "rotating":
		return c.buildRotatingHandler(cfg.Params)
	case "syslog":
		return c.buildSyslogHandler(cfg.Params)
	default:
		return nil, nil
	}
}

func (c *Config) buildConsoleHandler(params map[string]interface{}) (Handler, error) {
	h := NewConsoleHandler(os.Stdout)

	if format, ok := params["format"].(string); ok {
		if format == "json" {
			return &ConsoleHandler{Writer: os.Stdout, Format: JSONFormat{}}, nil
		}
	}

	return h, nil
}

func (c *Config) buildFileHandler(params map[string]interface{}) (Handler, error) {
	filename, _ := params["filename"].(string)
	if filename == "" {
		return nil, nil
	}

	h, err := NewFileHandler(filename)
	if err != nil {
		return nil, err
	}

	if format, ok := params["format"].(string); ok && format == "json" {
		h.Format = JSONFormat{}
	}

	return h, nil
}

func (c *Config) buildRotatingHandler(params map[string]interface{}) (Handler, error) {
	filename, _ := params["filename"].(string)
	if filename == "" {
		return nil, nil
	}

	maxSize := int64(10485760) // 10MB default
	if ms, ok := params["max_size"].(int); ok {
		maxSize = int64(ms)
	}

	maxBackups := 5
	if mb, ok := params["max_backups"].(int); ok {
		maxBackups = mb
	}

	h, err := NewRotatingFileHandler(filename, maxSize, maxBackups)
	if err != nil {
		return nil, err
	}

	if format, ok := params["format"].(string); ok && format == "json" {
		h.Format = JSONFormat{}
	}

	return h, nil
}

func (c *Config) buildSyslogHandler(params map[string]interface{}) (Handler, error) {
	h, err := NewSyslogHandler()
	if err != nil {
		return nil, err
	}

	if facility, ok := params["facility"].(int); ok {
		h.Facility = facility
	}

	if format, ok := params["format"].(string); ok && format == "json" {
		h.Format = JSONFormat{}
	}

	return h, nil
}
