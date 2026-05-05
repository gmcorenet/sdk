package gmcore_i18n

import (
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	DefaultLocale string   `yaml:"default_locale" json:"default_locale"`
	Directories   []string `yaml:"directories" json:"directories"`
	FallbackLocale string `yaml:"fallback_locale" json:"fallback_locale"`
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
		filepath.Join(l.appPath, "config", "i18n.yaml"),
		filepath.Join(l.appPath, "config", "i18n.yml"),
		filepath.Join(l.appPath, "config", "translation.yaml"),
		filepath.Join(l.appPath, "config", "translation.yml"),
		filepath.Join(l.appPath, "i18n.yaml"),
		filepath.Join(l.appPath, "i18n.yml"),
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

func (c *Config) Build() (*Translator, error) {
	if len(c.Directories) == 0 {
		c.Directories = []string{
			filepath.Join(c.DefaultLocale, "messages.yaml"),
		}
	}

	defaultLocale := c.DefaultLocale
	if defaultLocale == "" {
		defaultLocale = "en"
	}

	return LoadDirs(c.Directories, defaultLocale)
}
